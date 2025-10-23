package submission

import (
	"context"
	"fmt"
	"log/slog"

	docker "github.com/docker/docker/client"
	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/internal/domain/status"
	"github.com/voidcontests/coyote/internal/domain/verdict"
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
		err := s.submissionRepo.UpdateStatus(ctx, submission.ID, status.Judging)
		if err != nil {
			slog.Error("failed to update submission status", logger.Err(err))
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

	tr, err := s.runTests(ctx, submission.Code, l, tcs)
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
