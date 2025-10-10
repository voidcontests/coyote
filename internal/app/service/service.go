package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runner/internal/judge"
	"runner/internal/lib/sl"
	"runner/internal/repository"
	"runner/internal/repository/models"
	"runner/internal/runner"
	"runner/internal/runner/language"
	"time"

	"github.com/redis/go-redis/v9"
)

type Summary struct {
	Verdict          string
	PassedTestsCount int
	Stderr           string
}

type FailedTest struct {
	Input          string
	ExpectedOutput string
	ActualOutput   string
}

type Service struct {
	repository *repository.Repository
	rc         *redis.Client
}

func New(r *repository.Repository, rc *redis.Client) *Service {
	return &Service{
		repository: r,
		rc:         rc,
	}
}

func (s *Service) Listen(ctx context.Context) error {
	sub := s.rc.Subscribe(ctx, "submissions")
	defer sub.Close()

	_, err := sub.Receive(ctx)
	if err != nil {
		return err
	}

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("subscription channel closed")
			}

			var submission models.Submission
			if err := json.Unmarshal([]byte(msg.Payload), &submission); err != nil {
				slog.Info("raw message", slog.String("channel", msg.Channel), slog.String("payload", msg.Payload))
				continue
			}

			err := s.executeSolution(ctx, submission)
			if err != nil {
				slog.Error("can't execute solution", sl.Err(err))
			}
		}
	}
}

func (s *Service) executeSolution(ctx context.Context, submission models.Submission) error {
	slog.Debug("running submission", slog.Int("submission_id", int(submission.ID)))

	l, ok := language.Get(submission.Language)
	if !ok {
		return language.ErrUnknownLanguage
	}

	filebase := fmt.Sprintf("%d", time.Now().UnixNano())
	filepath := fmt.Sprintf("./sandbox/%s.%s", filebase, l.Extension)
	if err := os.WriteFile(filepath, []byte(submission.Code), 0644); err != nil {
		return fmt.Errorf("write code to file: %w", err)
	}
	defer runner.Flush(filebase)

	var report runner.Report
	var err error

	if l.Kind == language.Compiled {
		report, err = runner.Compile(filebase, l.Name)
		if err != nil {
			return fmt.Errorf("compilation error: %w", err)
		}
		if report.ExitCode != 0 {
			_ = s.repository.Submission.UpdateVerdict(ctx, submission.ID, judge.VerdictCompilationError, 0, report.Stderr)
			return nil
		}
	}

	tcs, err := s.repository.Problem.GetTestCases(ctx, submission.ProblemID)
	if err != nil {
		return fmt.Errorf("get test cases: %w", err)
	}

	summary := Summary{Verdict: judge.VerdictRunning}
	failedTest := FailedTest{}

	for _, tc := range tcs {
		report, err = runner.Exec(filebase, l.Name, 2000, tc.Input)
		if err != nil {
			return fmt.Errorf("execution error: %w", err)
		}

		if report.ExitCode == 124 {
			summary.Verdict = judge.VerdictTimeLimitExceeded
			failedTest = FailedTest{tc.Input, tc.Output, report.Stdout}
			break
		}

		match := judge.Match(report.Stdout, tc.Output)
		if report.ExitCode == 0 && match {
			summary.PassedTestsCount++
			continue
		} else if report.ExitCode != 0 {
			summary.Verdict = judge.VerdictRuntimeError
			summary.Stderr = report.Stderr
			failedTest = FailedTest{tc.Input, tc.Output, report.Stdout}
			break
		} else {
			summary.Verdict = judge.VerdictWrongAnswer
			failedTest = FailedTest{tc.Input, tc.Output, report.Stdout}
			break
		}
	}

	if summary.PassedTestsCount == len(tcs) {
		summary.Verdict = judge.VerdictOK
	}

	err = s.repository.Submission.UpdateVerdict(ctx, submission.ID, summary.Verdict, int32(summary.PassedTestsCount), summary.Stderr)
	if err != nil {
		return fmt.Errorf("update verdict: %w", err)
	}

	switch summary.Verdict {
	case judge.VerdictRuntimeError, judge.VerdictTimeLimitExceeded, judge.VerdictWrongAnswer:
		err = s.repository.Submission.CreateFailedTest(ctx, submission.ID, failedTest.Input, failedTest.ExpectedOutput, failedTest.ActualOutput)
		if err != nil {
			return fmt.Errorf("save failed test: %w", err)
		}
	}

	return nil
}
