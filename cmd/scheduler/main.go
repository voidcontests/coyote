package main

import (
	"context"

	"runner/internal/config"
	"runner/internal/pkg/app"
)

func main() {
	c := config.MustLoad()
	a := app.New(c)
	ctx := context.Background()

	a.Run(ctx)
}
