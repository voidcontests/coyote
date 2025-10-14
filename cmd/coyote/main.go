package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"runner/internal/config"
	httpdelivery "runner/internal/delivery/http"
	"runner/internal/delivery/queue"
	"runner/internal/repository/postgres"
	"runner/internal/repository/redis"
	"runner/internal/usecase/submission"
	"runner/internal/version"
	"runner/pkg/language"
	"runner/pkg/logger"
	"runner/pkg/matcher"
	"runner/pkg/runner"
)

func main() {
	c := config.MustLoad()

	logLevel := slog.LevelInfo
	if c.Env == config.EnvDevelopment || c.Env == config.EnvLocal {
		logLevel = slog.LevelDebug
	}
	log := logger.Setup(c.Env, logLevel)
	slog.SetDefault(log)

	slog.Info("runner: starting...", slog.String("env", c.Env), version.CommitAttr, version.BranchAttr)

	pool, err := postgres.NewPool(c.Postgres)
	if err != nil {
		slog.Error("postgresql: could not establish connection", logger.Err(err))
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("postgresql: connection established")

	submissionRepo := postgres.NewSubmissionRepository(pool)
	problemRepo := postgres.NewProblemRepository(pool)

	messageQueue := redis.NewMessageQueue(c.Redis)
	defer messageQueue.Close()

	codeRunner := runner.New(c.Runner.DockerImage)
	languageProvider := language.NewProvider()
	outputMatcher := matcher.NewOutputMatcher()

	submissionService := submission.New(
		submissionRepo,
		problemRepo,
		codeRunner,
		languageProvider,
		outputMatcher,
	)

	queueHandler := queue.NewHandler(submissionService, messageQueue)
	defer queueHandler.Close()

	httpHandler := httpdelivery.NewHandler()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", c.HTTP.Port),
		Handler: httpHandler,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error, 1)
	go func() {
		if err := queueHandler.Listen(ctx, c.Runner.Channel); err != nil && err != context.Canceled {
			errs <- err
		}
	}()

	go func() {
		slog.Info("http: server starting...", slog.String("port", c.HTTP.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errs <- err
		}
	}()

	slog.Info("runner: listening for submissions...", slog.String("channel", c.Runner.Channel))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-quit:
		slog.Info("runner: received shutdown signal")
	case err := <-errs:
		slog.Error("runner: fatal error", logger.Err(err))
	}

	cancel()
	slog.Info("runner: shutting down...")

	// shadow previous context to shutdown context
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("http: server shutdown error", logger.Err(err))
	} else {
		slog.Info("http: server stopped")
	}

	slog.Info("runner: stopped")
}
