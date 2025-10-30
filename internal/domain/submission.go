package domain

import "time"

type Submission struct {
	ID        int       `db:"id"`
	EntryID   int       `db:"entry_id"`
	ProblemID int       `db:"problem_id"`
	Status    string    `db:"status"`
	Verdict   string    `db:"verdict"`
	Code      string    `db:"code"`
	Language  string    `db:"language"`
	CreatedAt time.Time `db:"created_at"`
}

type TestingReport struct {
	ID                    int       `db:"id"`
	SubmissionID          int       `db:"submission_id"`
	PassedTestsCount      int       `db:"passed_tests_count"`
	TotalTestsCount       int       `db:"total_tests_count"`
	FirstFailedTestID     *int      `db:"first_failed_test_id"`
	FirstFailedTestOutput *string   `db:"first_failed_test_output"`
	Stderr                string    `db:"stderr"`
	CreatedAt             time.Time `db:"created_at"`
}

type TestCase struct {
	ID        int
	ProblemID int
	Input     string
	Output    string
	IsExample bool
}
