package submission

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/voidcontests/coyote/internal/domain"
)

type Service struct {
	submissionRepo   domain.SubmissionRepository
	problemRepo      domain.ProblemRepository
	codeRunner       domain.Runner
	languageProvider domain.LanguageProvider
	outputMatcher    domain.OutputMatcher
}

func New(submissionRepo domain.SubmissionRepository, problemRepo domain.ProblemRepository, codeRunner domain.Runner, languageProvider domain.LanguageProvider, outputMatcher domain.OutputMatcher) *Service {
	return &Service{
		submissionRepo:   submissionRepo,
		problemRepo:      problemRepo,
		codeRunner:       codeRunner,
		languageProvider: languageProvider,
		outputMatcher:    outputMatcher,
	}
}

func (s *Service) ProcessSubmission(ctx context.Context, submission domain.Submission) error {
	slog.Debug("processing submission", slog.Int("submission_id", int(submission.ID)))

	go func() {
		s.submissionRepo.UpdateVerdict(ctx, submission.ID, domain.VerdictRunning)
	}()

	lang, err := s.languageProvider.GetLanguage(submission.Language)
	if err != nil {
		return fmt.Errorf("get language: %w", err)
	}

	filebase := fmt.Sprintf("%d", time.Now().UnixNano())
	defer s.codeRunner.Cleanup(filebase)

	testCases, err := s.problemRepo.GetTestCases(ctx, submission.ProblemID)
	if err != nil {
		return fmt.Errorf("get test cases: %w", err)
	}

	verdict, passedCount, stderr, failedTest := s.executeTestCases(
		filebase,
		lang,
		submission.Code,
		testCases,
	)

	err = s.submissionRepo.SetResult(ctx, submission.ID, verdict, int32(passedCount), stderr)
	if err != nil {
		return fmt.Errorf("update verdict: %w", err)
	}

	if failedTest != nil {
		err = s.submissionRepo.CreateFailedTest(
			ctx,
			submission.ID,
			failedTest.Input,
			failedTest.ExpectedOutput,
			failedTest.ActualOutput,
		)
		if err != nil {
			return fmt.Errorf("save failed test: %w", err)
		}
	}

	return nil
}

func (s *Service) executeTestCases(filebase string, lang domain.Language, code string, testCases []domain.TestCase) (verdict string, passedCount int, stderr string, failedTest *domain.FailedTest) {
	verdict = domain.VerdictRunning

	for _, tc := range testCases {
		req := domain.ExecutionRequest{
			Filebase:    filebase,
			Language:    lang.Name,
			Code:        code,
			Input:       tc.Input,
			TimeLimitMS: 2000,
		}

		report, err := s.codeRunner.Execute(req)
		if err != nil {
			slog.Error("execution error", slog.String("error", err.Error()))
			verdict = domain.VerdictRuntimeError
			stderr = err.Error()
			failedTest = &domain.FailedTest{
				Input:          tc.Input,
				ExpectedOutput: tc.Output,
				ActualOutput:   report.Stdout,
			}
			break
		}

		if lang.Kind == domain.Compiled && report.ExitCode != 0 && report.Stderr != "" {
			if strings.Contains(report.Stderr, "error:") || strings.Contains(report.Stderr, "fatal error:") {
				verdict = domain.VerdictCompilationError
				stderr = report.Stderr
				break
			}
		}

		if report.ExitCode == 124 {
			verdict = domain.VerdictTimeLimitExceeded
			failedTest = &domain.FailedTest{
				Input:          tc.Input,
				ExpectedOutput: tc.Output,
				ActualOutput:   report.Stdout,
			}
			break
		}

		match := s.outputMatcher.Match(report.Stdout, tc.Output)
		if report.ExitCode == 0 && match {
			passedCount++
			continue
		} else if report.ExitCode != 0 {
			verdict = domain.VerdictRuntimeError
			stderr = report.Stderr
			failedTest = &domain.FailedTest{
				Input:          tc.Input,
				ExpectedOutput: tc.Output,
				ActualOutput:   report.Stdout,
			}
			break
		} else {
			verdict = domain.VerdictWrongAnswer
			failedTest = &domain.FailedTest{
				Input:          tc.Input,
				ExpectedOutput: tc.Output,
				ActualOutput:   report.Stdout,
			}
			break
		}
	}

	if passedCount == len(testCases) {
		verdict = domain.VerdictOK
	}

	return verdict, passedCount, stderr, failedTest
}
