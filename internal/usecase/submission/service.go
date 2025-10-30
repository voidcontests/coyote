package submission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/internal/domain/status"
	"github.com/voidcontests/coyote/internal/domain/verdict"
	"github.com/voidcontests/coyote/pkg/container"
	"github.com/voidcontests/coyote/pkg/judge"
	"github.com/voidcontests/coyote/pkg/language"
	"github.com/voidcontests/coyote/pkg/logger"
)

type Service struct {
	submissionRepo    domain.SubmissionRepository
	problemRepo       domain.ProblemRepository
	containerProvider container.Provider
}

func New(sr domain.SubmissionRepository, pr domain.ProblemRepository, cp container.Provider) *Service {
	return &Service{
		submissionRepo:    sr,
		problemRepo:       pr,
		containerProvider: cp,
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

	problem, err := s.problemRepo.GetByID(ctx, submission.ProblemID)
	if err != nil {
		return fmt.Errorf("failed to get time limit: %w", err)
	}

	opts := options{
		language:      l,
		code:          submission.Code,
		tests:         tcs,
		timeLimit:     time.Duration(problem.TimeLimitMS) * time.Millisecond,
		memoryLimitMB: problem.MemoryLimitMB,
		checker:       problem.Checker,
		paths: contpaths{
			source: fmt.Sprintf("/sandbox/solution.%s", l.Extension),
			build:  "/sandbox/solution",
			input:  "/sandbox/input.txt",
		},
	}

	// extract time & memory limit, language, checker into options
	tr, err := s.runTests(ctx, opts)
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

type contpaths struct {
	source string
	build  string
	input  string
}

type options struct {
	language      language.Language
	code          string
	tests         []domain.TestCase
	timeLimit     time.Duration
	memoryLimitMB int
	checker       string
	paths         contpaths
}

func (s *Service) runTests(ctx context.Context, opts options) (report, error) {
	cc, err := s.containerProvider.CreateContainer(ctx, opts.memoryLimitMB)
	if err != nil {
		return report{}, err
	}
	defer cc.Flush(ctx)

	err = cc.WriteFile(ctx, opts.paths.source, opts.code)
	if err != nil {
		return report{}, err
	}

	if opts.language.IsCompiled {
		cr, ok := s.compile(ctx, cc, opts)
		if !ok {
			return cr, nil
		}
	}

	return s.executeTests(ctx, cc, opts)
}

func (s *Service) compile(ctx context.Context, cc *container.Context, opts options) (report, bool) {
	cmd, ok := language.GetCompilationCommand(opts.language, opts.paths.source, opts.paths.build)
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
			total:            len(opts.tests),
			stderr:           pr.Stderr,
			failedTestCase:   nil,
			failedTestOutput: "",
		}, false
	}

	return report{}, true
}

func (s *Service) executeTests(ctx context.Context, cc *container.Context, opts options) (report, error) {
	cmd, ok := language.GetExecutionCommand(opts.language, opts.paths.source, opts.paths.input, opts.paths.build)
	if !ok {
		return report{}, fmt.Errorf("no execution command for language: %s", opts.language.Name)
	}

	tt := len(opts.tests)
	for i, tc := range opts.tests {
		err := cc.WriteFile(ctx, opts.paths.input, tc.Input)
		if err != nil {
			return report{}, err
		}

		pr, err := cc.ExecuteWithTimeout(ctx, cmd, opts.timeLimit)
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
		jr := judge.Check(opts.checker, pr.Stdout, tc.Output)
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
