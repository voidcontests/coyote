package submission

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/client"
	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/pkg/container"
	"github.com/voidcontests/coyote/pkg/language"
	"github.com/voidcontests/coyote/pkg/matcher"
)

type Service struct {
	submissionRepo domain.SubmissionRepository
	problemRepo    domain.ProblemRepository
	client         *client.Client
}

func New(submissionRepo domain.SubmissionRepository, problemRepo domain.ProblemRepository, c *client.Client) *Service {
	return &Service{
		submissionRepo: submissionRepo,
		problemRepo:    problemRepo,
		client:         c,
	}
}

func (s *Service) ProcessSubmission(ctx context.Context, submission domain.Submission) error {
	slog.Debug("processing submission", slog.Int("submission_id", int(submission.ID)))

	go func() {
		s.submissionRepo.UpdateVerdict(ctx, submission.ID, domain.VerdictRunning)
	}()

	l, ok := language.Get(submission.Language)
	if !ok {
		return fmt.Errorf("unknown language: %s", submission.Language)
	}

	tcs, err := s.problemRepo.GetTestCases(ctx, submission.ProblemID)
	if err != nil {
		return fmt.Errorf("get test cases: %w", err)
	}

	r, err := s.TestSolution(ctx, submission.Code, l, tcs)
	if err != nil {
		return err
	}

	err = s.submissionRepo.SetResult(ctx, submission.ID, r.verdict, int32(r.passed), r.stderr)
	if err != nil {
		return fmt.Errorf("update verdict: %w", err)
	}

	if r.failed != nil {
		err = s.submissionRepo.CreateFailedTest(
			ctx,
			submission.ID,
			r.failed.Input,
			r.failed.ExpectedOutput,
			r.failed.ActualOutput,
		)
		if err != nil {
			return fmt.Errorf("save failed test: %w", err)
		}
	}

	return nil
}

type TestingReport struct {
	verdict string
	passed  int
	total   int
	stderr  string
	failed  *domain.FailedTest
}

func (s *Service) TestSolution(ctx context.Context, code string, l language.Language, tcs []domain.TestCase) (TestingReport, error) {
	cc, err := container.New(ctx, s.client)
	if err != nil {
		return TestingReport{}, err
	}

	path := struct {
		source string
		build  string
		input  string
	}{
		source: fmt.Sprintf("solution.%s", l.Extension),
		build:  "/sandbox/solution",
		input:  "/sandbox/input.txt",
	}

	err = cc.WriteFile(ctx, path.source, code)
	if err != nil {
		return TestingReport{}, err
	}

	tt := len(tcs)
	if l.IsCompiled {
		cmd, ok := language.GetCompilationCommand(l, path.source, path.build)
		if !ok {
			return TestingReport{}, fmt.Errorf("no compilation command for language: %s", l.Name)
		}

		pr, err := cc.Execute(ctx, cmd)
		if err != nil {
			return TestingReport{}, err
		}

		if !pr.Ok {
			return TestingReport{
				verdict: domain.VerdictCompilationError,
				passed:  0,
				total:   tt,
				stderr:  pr.Stderr,
				failed:  nil,
			}, nil
		}
	}

	cmd, ok := language.GetExecutionCommand(l, path.source, path.input, path.build)
	if !ok {
		return TestingReport{}, fmt.Errorf("no execution command for language: %s", l.Name)
	}

	for i, tc := range tcs {
		err = cc.WriteFile(ctx, path.input, tc.Input)
		if err != nil {
			return TestingReport{}, err
		}

		pr, err := s.executeWithTimeout(ctx, cc, cmd, 2*time.Second)
		if err == context.DeadlineExceeded {
			return TestingReport{
				verdict: domain.VerdictTimeLimitExceeded,
				passed:  i,
				total:   tt,
				failed: &domain.FailedTest{
					Input:          tc.Input,
					ExpectedOutput: tc.Output,
				},
			}, nil
		}
		if err != nil {
			return TestingReport{}, err
		}

		if !pr.Ok {
			return TestingReport{
				verdict: domain.VerdictRuntimeError,
				passed:  i,
				total:   tt,
				stderr:  pr.Stderr,
				failed: &domain.FailedTest{
					Input:          tc.Input,
					ExpectedOutput: tc.Output,
					ActualOutput:   pr.Stdout,
				},
			}, nil
		}

		ok := matcher.Match(pr.Stdout, tc.Output)
		if !ok {
			return TestingReport{
				verdict: domain.VerdictWrongAnswer,
				passed:  i,
				total:   tt,
				stderr:  pr.Stderr,
				failed: &domain.FailedTest{
					Input:          tc.Input,
					ExpectedOutput: tc.Output,
					ActualOutput:   pr.Stdout,
				},
			}, nil
		}
	}

	return TestingReport{
		verdict: domain.VerdictOK,
		passed:  tt,
		total:   tt,
		stderr:  "",
		failed:  nil,
	}, nil
}

func (s *Service) executeWithTimeout(ctx context.Context, cc *container.Context, cmd string, timeout time.Duration) (container.ProcessResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prch := make(chan container.ProcessResult, 1)
	errch := make(chan error, 1)

	go func() {
		pr, err := cc.Execute(ctx, cmd)
		if err != nil {
			errch <- err
			return
		}
		prch <- pr
	}()

	select {
	case pr := <-prch:
		return pr, nil
	case err := <-errch:
		return container.ProcessResult{}, err
	case <-ctx.Done():
		return container.ProcessResult{}, context.DeadlineExceeded
	}
}
