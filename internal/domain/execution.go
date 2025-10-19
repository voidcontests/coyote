package domain

type ExecutionReport struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type ExecutionRequest struct {
	Filebase string
	Language string
	CodeB64  string
	InputB64 string
}
