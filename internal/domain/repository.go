package domain

import "context"

type SubmissionRepository interface {
	UpdateVerdict(ctx context.Context, id int32, verdict string, passedTestsCount int32, stderr string) error
	CreateFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput string) error
	GetPending(ctx context.Context) (Submission, error)
}

type ProblemRepository interface {
	GetTestCases(ctx context.Context, problemID int32) ([]TestCase, error)
}

type MessageQueue interface {
	Subscribe(ctx context.Context, channel string) (<-chan Submission, error)
	Close() error
}

type Runner interface {
	Execute(req ExecutionRequest) (ExecutionReport, error)
	Cleanup(filebase string) error
}

type LanguageProvider interface {
	GetLanguage(name string) (Language, error)
}

type OutputMatcher interface {
	Match(actual, expected string) bool
}
