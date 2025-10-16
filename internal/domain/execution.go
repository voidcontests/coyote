package domain

type ExecutionReport struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type ExecutionRequest struct {
	Filebase    string
	Language    string
	Code        string
	Input       string
	TimeLimitMS int
}
