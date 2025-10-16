package logger

import (
	"log/slog"
	"os"
)

func Setup(env string, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

func Err(err error) slog.Attr {
	var msg string
	if err == nil {
		msg = "nil"
	} else {
		msg = err.Error()
	}
	return slog.Attr{Key: "error", Value: slog.StringValue(msg)}
}
