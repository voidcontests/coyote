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

	docker "github.com/docker/docker/client"
	"github.com/voidcontests/coyote/internal/config"
	httpdelivery "github.com/voidcontests/coyote/internal/delivery/http"
	qdelivery "github.com/voidcontests/coyote/internal/delivery/queue"
	"github.com/voidcontests/coyote/internal/repository/postgres"
	"github.com/voidcontests/coyote/internal/repository/redis"
	"github.com/voidcontests/coyote/internal/usecase/submission"
	"github.com/voidcontests/coyote/internal/version"
	"github.com/voidcontests/coyote/pkg/logger"
)

func main() {
	c := config.MustLoad()

	logLevel := slog.LevelInfo
	if c.Env == config.EnvDevelopment || c.Env == config.EnvLocal {
		logLevel = slog.LevelDebug
	}
	log := logger.Setup(c.Env, logLevel)
	slog.SetDefault(log)

	slog.Info("coyote: starting...", slog.String("env", c.Env), version.CommitAttr, version.BranchAttr)

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

	dc, err := docker.NewClientWithOpts(
		docker.WithHost(docker.DefaultDockerHost),
		docker.WithAPIVersionNegotiation(),
	)

	ss := submission.New(submissionRepo, problemRepo, dc)

	queueHandler := qdelivery.NewHandler(ss, messageQueue)
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

	slog.Info("coyote: listening for submissions...", slog.String("channel", c.Runner.Channel))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case s := <-quit:
		slog.Info("coyote: received shutdown signal", slog.String("signal", s.String()))
	case err := <-errs:
		slog.Error("coyote: fatal error", logger.Err(err))
	}

	cancel()
	slog.Info("coyote: shutting down...")

	// shadow previous context to shutdown context
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("http: error happened while shutting down the server", logger.Err(err))
	} else {
		slog.Info("http: server stopped")
	}

	slog.Info("coyote: stopped")
}
