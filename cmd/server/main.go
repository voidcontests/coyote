package main

import (
	"context"
	"log/slog"
	"os"

	"runner/internal/config"
	"runner/internal/lib/sl"
	"runner/internal/pkg/app"
)

func main() {
	err := os.MkdirAll("files", 0755)
	if err != nil {
		slog.Error("failed to create `./files/` directory", sl.Err(err))
		return
	}

	c := config.MustLoad()
	a := app.New(c)
	ctx := context.Background()

	a.Run(ctx)
}
