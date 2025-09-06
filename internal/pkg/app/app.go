package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"runner/internal/app/service"
	"runner/internal/config"
	"runner/internal/lib/sl"
	"runner/internal/repository"
	"runner/internal/repository/postgres"

	"github.com/redis/go-redis/v9"
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
	repository := repository.New(db)

	rc := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	s := service.New(repository, rc)

	go s.Listen(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	slog.Info("runner: listening for submissions...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("runner: stopping...")
	slog.Info("runner: stopped")
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
