package domain

import "context"

type SubmissionRepository interface {
	SetResult(context.Context, int32, string, int32, string) error
	UpdateVerdict(context.Context, int32, string) error
	CreateFailedTest(context.Context, int32, string, string, string) error
}

type ProblemRepository interface {
	GetTestCases(context.Context, int32) ([]TestCase, error)
}

type MessageQueue interface {
	Subscribe(context.Context, string) (<-chan Submission, error)
	Close() error
}

type Runner interface {
	Execute(ExecutionRequest) (ExecutionReport, error)
	Cleanup(string) error
}

type LanguageProvider interface {
	GetLanguage(string) (Language, error)
}

type OutputMatcher interface {
	Match(string, string) bool
}
