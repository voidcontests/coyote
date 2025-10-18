package submission

import (
	"context"
	"errors"
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

	verdict, passedCount, stderr, failedTest := s.executeTestCases(filebase, lang, submission.Code, testCases)

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

	// TODO: move this timeout somewhere
	timeout := 2 * time.Second

	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		req := domain.ExecutionRequest{
			Filebase: filebase,
			Language: lang.Name,
			Code:     code,
			Input:    tc.Input,
		}

		report, err := s.executeWithTimeout(ctx, req)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				verdict = domain.VerdictTimeLimitExceeded
				failedTest = &domain.FailedTest{
					Input:          tc.Input,
					ExpectedOutput: tc.Output,
					ActualOutput:   "",
				}
				return
			}

			// TODO: introduce verdict `cancelled`, and set it
			// when unexpected error happened while execution
			slog.Error("execution error", slog.String("error", err.Error()))
			break
		}

		v, ft, se := s.processReport(report, tc, lang)
		if v != domain.VerdictOK {
			verdict = v
			failedTest = ft
			stderr = se
			break
		}

		passedCount++
	}

	if passedCount == len(testCases) {
		verdict = domain.VerdictOK
	}

	return
}

func (s *Service) executeWithTimeout(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionReport, error) {
	errch := make(chan error, 1)
	reportch := make(chan domain.ExecutionReport, 1)

	go func() {
		report, err := s.codeRunner.Execute(req)
		if err != nil {
			errch <- err
			return
		}
		reportch <- report
	}()

	select {
	case <-ctx.Done():
		return domain.ExecutionReport{}, ctx.Err() // returns context.DeadlineExceeded
	case err := <-errch:
		return domain.ExecutionReport{}, err
	case report := <-reportch:
		return report, nil
	}
}

func (s *Service) processReport(report domain.ExecutionReport, tc domain.TestCase, lang domain.Language) (verdict string, failedTest *domain.FailedTest, stderr string) {
	// TODO: This is VERY dump check for `compilation_error`.
	// But we can't do much with it for now:
	//  - precompilation: unavailable - requires volumes to keep binary
	if lang.Kind == domain.Compiled && report.ExitCode != 0 && report.Stderr != "" {
		// compilation error
		if strings.Contains(report.Stderr, "error:") || strings.Contains(report.Stderr, "fatal error:") {
			return domain.VerdictCompilationError, nil, report.Stderr
		}
	}

	match := s.outputMatcher.Match(report.Stdout, tc.Output)

	if report.ExitCode == 0 && match {
		// test passed
		return domain.VerdictOK, nil, ""
	} else if report.ExitCode != 0 {
		// runtime error
		failedTest = &domain.FailedTest{
			Input:          tc.Input,
			ExpectedOutput: tc.Output,
			ActualOutput:   report.Stdout,
		}
		return domain.VerdictRuntimeError, failedTest, report.Stderr
	} else {
		// wrong answer (exit code 0 but output doesn't match)
		failedTest = &domain.FailedTest{
			Input:          tc.Input,
			ExpectedOutput: tc.Output,
			ActualOutput:   report.Stdout,
		}
		return domain.VerdictWrongAnswer, failedTest, ""
	}
}
