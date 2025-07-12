package sl

import "log/slog"

// Err returns an error attribute for slog.Record
func Err(err error) slog.Attr {
	var msg string
	if err == nil {
		msg = "nil"
	} else {
		msg = err.Error()
	}

	return slog.Attr{Key: "error", Value: slog.StringValue(msg)}
}
