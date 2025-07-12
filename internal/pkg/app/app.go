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
	var logger *slog.Logger
	switch a.config.Env {
	case config.EnvDevelopment, config.EnvLocal:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	case config.EnvProduction:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	slog.SetDefault(logger)

	slog.Info("runner: starting...", slog.String("env", a.config.Env))

	db, err := postgres.New(&a.config.Postgres)
	if err != nil {
		slog.Error("postgresql: could not connect establish connection", sl.Err(err))
		return
	}

	slog.Info("postgresql: ok")

	repo := repository.New(db)

	delay := 500 * time.Millisecond
	for i := 0; i < 5; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					slog.Debug("worker stopped via context")
					return
				default:
					submission, err := repo.Submission.GetPending(ctx)
					if errors.Is(err, sql.ErrNoRows) {
						time.Sleep(delay)
						continue
					}
					if err != nil {
						slog.Error("can't get pending submissions", sl.Err(err))
						time.Sleep(delay)
						continue
					}

					if err := ExecuteSolution(ctx, repo, submission); err != nil {
						slog.Error("submission execution failed", sl.Err(err))
					}
				}
			}
		}()
	}

	slog.Info("runner: started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("runner: stopped")
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

func ExecuteSolution(ctx context.Context, r *repository.Repository, submission models.Submission) error {
	slog.Debug("running submission", slog.Int("id", int(submission.ID)))

	l, ok := language.Get(submission.Language)
	if !ok {
		return language.ErrUnknownLanguage
	}

	filebase := fmt.Sprintf("%d", time.Now().Unix())
	filepath := fmt.Sprintf("./files/%s.%s", filebase, l.Extension)
	source, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer source.Close()
	defer runner.Flush(filebase)

	if _, err := source.WriteString(submission.Code); err != nil {
		return err
	}

	var report runner.Report
	if l.Kind == language.Compiled {
		report, err = runner.Compile(filebase, l.Name)
		if err != nil {
			return err
		}

		if report.ExitCode != 0 {
			return r.Submission.UpdateVerdict(ctx, submission.ID, judge.VerdictCompilationError, 0, report.Stderr)
		}
	}

	tcs, err := r.Problem.GetTestCases(ctx, submission.ProblemID)
	if err != nil {
		return err
	}

	failedTest := FailedTest{}
	summary := Summary{
		Verdict:          judge.VerdictRunning,
		PassedTestsCount: 0,
	}

	for _, tc := range tcs {
		report, err = runner.Exec(filebase, l.Name, 2000, tc.Input)
		if err != nil {
			return err
		}

		// if exit code 124, assume timeout
		if report.ExitCode == 124 {
			summary.Verdict = judge.VerdictTimeLimitExceeded
			failedTest.Input = tc.Input
			failedTest.ExpectedOutput = tc.Output
			failedTest.ActualOutput = report.Stdout
			break
		}

		match := judge.Match(report.Stdout, tc.Output)
		if report.ExitCode == 0 && match {
			summary.PassedTestsCount++
		} else if report.ExitCode != 0 {
			summary.Verdict = judge.VerdictRuntimeError
			summary.Stderr = report.Stderr
			failedTest.Input = tc.Input
			failedTest.ExpectedOutput = tc.Output
			failedTest.ActualOutput = report.Stdout
			break
		} else if !match {
			summary.Verdict = judge.VerdictWrongAnswer
			failedTest.Input = tc.Input
			failedTest.ExpectedOutput = tc.Output
			failedTest.ActualOutput = report.Stdout
			break
		}
	}

	err = r.Submission.UpdateVerdict(ctx, submission.ID, summary.Verdict, int32(summary.PassedTestsCount), summary.Stderr)
	if err != nil {
		return err
	}

	err = r.Submission.CreateFailedTest(ctx, submission.ID, failedTest.Input, failedTest.ExpectedOutput, failedTest.ActualOutput)
	if err != nil {
		return err
	}

	return nil
}
