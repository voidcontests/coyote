package domain

import "context"

type SubmissionRepository interface {
	UpdateVerdictAndStatus(ctx context.Context, submissionID int, verdict string, status string) error
	UpdateStatus(ctx context.Context, submissionID int, status string) error
	CreateTestingReportAndComplete(ctx context.Context, report *TestingReport, status string, verdict string) error
}

type ProblemRepository interface {
	GetTestCases(ctx context.Context, problemID int) ([]TestCase, error)
}

type MessageQueue interface {
	Subscribe(ctx context.Context, channel string) (<-chan Submission, error)
	Close() error
}
