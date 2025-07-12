package judge

import "strings"

const (
	VerdictPending           = "pending"
	VerdictRunning           = "running"
	VerdictOK                = "ok"
	VerdictWrongAnswer       = "wrong_answer"
	VerdictRuntimeError      = "runtime_error"
	VerdictCompilationError  = "compilation_error"
	VerdictTimeLimitExceeded = "time_limit_exceeded"
)

func Match(actual, expected string) bool {
	suffix := "\n"
	return strings.TrimSuffix(actual, suffix) == strings.TrimSuffix(expected, suffix)
}
