package main

import (
	"context"
	"log/slog"
	"os"

	"runner/internal/config"
	"runner/internal/pkg/app"
)

func main() {
	err := os.MkdirAll("files", 0755)
	if err != nil {
		slog.Error("failed to create `./files/` directory", slog.Any("error", err))
		return
	}

	c := config.MustLoad()
	a := app.New(c)
	ctx := context.Background()

	a.Run(ctx)
}
