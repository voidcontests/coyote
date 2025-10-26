package submission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	docker "github.com/docker/docker/client"

	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/internal/domain/status"
	"github.com/voidcontests/coyote/internal/domain/verdict"
	"github.com/voidcontests/coyote/pkg/container"
	"github.com/voidcontests/coyote/pkg/judge"
	"github.com/voidcontests/coyote/pkg/language"
	"github.com/voidcontests/coyote/pkg/logger"
)

type Service struct {
	submissionRepo domain.SubmissionRepository
	problemRepo    domain.ProblemRepository
	client         *docker.Client // TODO: Try to remove this direct dependency on *docker.Client
}

func New(submissionRepo domain.SubmissionRepository, problemRepo domain.ProblemRepository, dc *docker.Client) *Service {
	return &Service{
		submissionRepo: submissionRepo,
		problemRepo:    problemRepo,
		client:         dc,
	}
}

func (s *Service) ProcessSubmission(ctx context.Context, submission domain.Submission) error {
	go func() {
		if err := s.submissionRepo.UpdateStatus(ctx, submission.ID, status.Judging); err != nil {
			slog.Error("failed to update submission status to judging",
				slog.Int("submission_id", submission.ID),
				logger.Err(err),
			)
		}
	}()

	l, ok := language.Get(submission.Language)
	if !ok {
		return fmt.Errorf("unknown language: %s", submission.Language)
	}

	tcs, err := s.problemRepo.GetTestCases(ctx, submission.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to get test cases: %w", err)
	}

	timeout, memoryLimitMB, err := s.problemRepo.GetConstraints(ctx, submission.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to get time limit: %w", err)
	}

	tr, err := s.runTests(ctx, submission.Code, l, tcs, timeout, memoryLimitMB)
	if err != nil {
		if err := s.submissionRepo.UpdateVerdictAndStatus(ctx, submission.ID, verdict.IE, status.Failed); err != nil {
			slog.Error("failed to update submission verdict", logger.Err(err))
		}
		return err
	}

	params := domain.TestingReport{
		SubmissionID:     submission.ID,
		PassedTestsCount: tr.passed,
		TotalTestsCount:  len(tcs),
		Stderr:           tr.stderr,
	}

	if tr.failedTestCase != nil {
		params.FirstFailedTestID = &tr.failedTestCase.ID
		params.FirstFailedTestOutput = &tr.failedTestOutput
	}

	err = s.submissionRepo.CreateTestingReportAndComplete(ctx, &params, status.Success, tr.verdict)
	if err != nil {
		return fmt.Errorf("failed to create testing report or update submission's status/verdict: %w", err)
	}

	return nil
}

type report struct {
	verdict          string
	passed           int
	total            int
	stderr           string
	failedTestCase   *domain.TestCase
	failedTestOutput string
}

type containerPaths struct {
	source string
	build  string
	input  string
}

func (s *Service) runTests(ctx context.Context, code string, l language.Language, tcs []domain.TestCase, timeout time.Duration, memoryLimitMB int) (report, error) {
	cc, err := container.New(ctx, s.client, memoryLimitMB)
	if err != nil {
		return report{}, err
	}
	defer cc.Flush(ctx)

	paths := containerPaths{
		source: fmt.Sprintf("/sandbox/solution.%s", l.Extension),
		build:  "/sandbox/solution",
		input:  "/sandbox/input.txt",
	}

	err = cc.WriteFile(ctx, paths.source, code)
	if err != nil {
		return report{}, err
	}

	tt := len(tcs)
	if l.IsCompiled {
		cr, ok := s.compile(ctx, cc, l, paths, tt)
		if !ok {
			return cr, nil
		}
	}

	return s.executeTests(ctx, cc, l, paths, tcs, timeout)
}

func (s *Service) compile(ctx context.Context, cc *container.Context, l language.Language, paths containerPaths, tt int) (report, bool) {
	cmd, ok := language.GetCompilationCommand(l, paths.source, paths.build)
	if !ok {
		return report{}, false
	}

	pr, err := cc.Execute(ctx, cmd)
	if err != nil {
		return report{}, false
	}

	if !pr.Ok {
		return report{
			verdict:          verdict.CE,
			passed:           0,
			total:            tt,
			stderr:           pr.Stderr,
			failedTestCase:   nil,
			failedTestOutput: "",
		}, false
	}

	return report{}, true
}

func (s *Service) executeTests(ctx context.Context, cc *container.Context, l language.Language, paths containerPaths, tcs []domain.TestCase, timeout time.Duration) (report, error) {
	cmd, ok := language.GetExecutionCommand(l, paths.source, paths.input, paths.build)
	if !ok {
		return report{}, fmt.Errorf("no execution command for language: %s", l.Name)
	}

	tt := len(tcs)
	for i, tc := range tcs {
		err := cc.WriteFile(ctx, paths.input, tc.Input)
		if err != nil {
			return report{}, err
		}

		pr, err := cc.ExecuteWithTimeout(ctx, cmd, timeout)
		if errors.Is(err, context.DeadlineExceeded) {
			return report{
				verdict:        verdict.TLE,
				passed:         i,
				total:          tt,
				failedTestCase: &tc,
			}, nil
		}
		if err != nil {
			return report{}, err
		}

		if pr.ExitCode == 137 && strings.Contains(pr.Stderr, "Killed") {
			// NOTE: sleep briefly to allow docker to update the container state;
			// without this, checking OOMKilled immediately may yield false negatives.
			time.Sleep(300 * time.Millisecond)
			killed, err := cc.IsKilledByOOM(ctx)
			if err != nil {
				return report{}, err
			}

			if killed {
				return report{
					verdict:          verdict.MLE,
					passed:           i,
					total:            tt,
					stderr:           pr.Stderr,
					failedTestCase:   &tc,
					failedTestOutput: pr.Stdout,
				}, nil
			}
		}

		if !pr.Ok {
			return report{
				verdict:          verdict.RE,
				passed:           i,
				total:            tt,
				stderr:           pr.Stderr,
				failedTestCase:   &tc,
				failedTestOutput: pr.Stdout,
			}, nil
		}

		// TODO: Introduce judge message for testing report
		jr := judge.Tokens(pr.Stdout, tc.Output)
		if jr.Verdict != verdict.OK {
			return report{
				verdict:          jr.Verdict,
				passed:           i,
				total:            tt,
				stderr:           pr.Stderr,
				failedTestCase:   &tc,
				failedTestOutput: pr.Stdout,
			}, nil
		}
	}

	return report{
		verdict:          verdict.OK,
		passed:           tt,
		total:            tt,
		stderr:           "",
		failedTestCase:   nil,
		failedTestOutput: "",
	}, nil
}
