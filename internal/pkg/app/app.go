package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"runner/internal/config"
	"runner/internal/judge"
	"runner/internal/lib/sl"
	"runner/internal/repository"
	"runner/internal/repository/models"
	"runner/internal/repository/postgres"
	"runner/internal/runner"
	"runner/internal/runner/language"
)

type App struct {
	config *config.Config
}

func New(config *config.Config) *App {
	return &App{config}
}

func (a *App) Run(ctx context.Context) {
	logger := setupLogger(a.config.Env)
	slog.SetDefault(logger)
	slog.Info("runner: starting...", slog.String("env", a.config.Env))

	db, err := postgres.New(&a.config.Postgres)
	if err != nil {
		slog.Error("postgresql: could not establish connection", sl.Err(err))
		return
	}
	defer db.Close()

	slog.Info("postgresql: ok")
	repo := repository.New(db)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workers := 5
	for i := 0; i < workers; i++ {
		go a.worker(ctx, repo, i)
	}

	slog.Info("runner: started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("runner: stopping...")
	slog.Info("runner: stopped")
}

func (a *App) worker(ctx context.Context, repo *repository.Repository, id int) {
	delay := 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			slog.Debug("worker stopped", slog.Int("worker_id", id))
			return
		default:
			submission, err := repo.Submission.GetPending(ctx)
			if errors.Is(err, sql.ErrNoRows) {
				time.Sleep(delay)
				continue
			}
			if err != nil {
				slog.Error("can't get pending submissions", slog.Int("worker_id", id), sl.Err(err))
				time.Sleep(delay)
				continue
			}

			if err := executeSolution(ctx, repo, submission, id); err != nil {
				slog.Error("submission execution failed", slog.Int("worker_id", id), sl.Err(err))
			}
		}
	}
}

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

func executeSolution(ctx context.Context, r *repository.Repository, submission models.Submission, workerID int) error {
	slog.Debug("running submission", slog.Int("worker_id", workerID), slog.Int("submission_id", int(submission.ID)))

	l, ok := language.Get(submission.Language)
	if !ok {
		return language.ErrUnknownLanguage
	}

	filebase := fmt.Sprintf("%d", time.Now().UnixNano())
	filepath := fmt.Sprintf("./files/%s.%s", filebase, l.Extension)
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
			_ = r.Submission.UpdateVerdict(ctx, submission.ID, judge.VerdictCompilationError, 0, report.Stderr)
			return nil
		}
	}

	tcs, err := r.Problem.GetTestCases(ctx, submission.ProblemID)
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

	err = r.Submission.UpdateVerdict(ctx, submission.ID, summary.Verdict, int32(summary.PassedTestsCount), summary.Stderr)
	if err != nil {
		return fmt.Errorf("update verdict: %w", err)
	}

	switch summary.Verdict {
	case judge.VerdictRuntimeError, judge.VerdictTimeLimitExceeded, judge.VerdictWrongAnswer:
		err = r.Submission.CreateFailedTest(ctx, submission.ID, failedTest.Input, failedTest.ExpectedOutput, failedTest.ActualOutput)
		if err != nil {
			return fmt.Errorf("save failed test: %w", err)
		}
	}

	return nil
}

func setupLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == config.EnvDevelopment || env == config.EnvLocal {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
