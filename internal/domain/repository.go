package domain

import "context"

type SubmissionRepository interface {
	SetResult(ctx context.Context, submissionID int32, status string, verdict string, passedTestsCount int32, stderr string) error
	UpdateVerdictAndStatus(ctx context.Context, submissionID int32, verdict string, status string) error
	UpdateStatus(ctx context.Context, submissionID int32, status string) error
	CreateFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput string) error
}

type ProblemRepository interface {
	GetTestCases(ctx context.Context, problemID int32) ([]TestCase, error)
}

type MessageQueue interface {
	Subscribe(ctx context.Context, channel string) (<-chan Submission, error)
	Close() error
}
