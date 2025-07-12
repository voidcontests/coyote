package models

import "time"

type TestCase struct {
	ID        int32  `db:"id"`
	ProblemID int32  `db:"problem_id"`
	Input     string `db:"input"`
	Output    string `db:"output"`
	IsExample bool   `db:"is_example"`
}

type Submission struct {
	ID               int32      `db:"id"`
	EntryID          int32      `db:"entry_id"`
	ProblemID        int32      `db:"problem_id"`
	Verdict          string     `db:"verdict"`
	Answer           string     `db:"answer"`
	Code             string     `db:"code"`
	Language         string     `db:"language"`
	PassedTestsCount int32      `db:"passed_tests_count"`
	Stderr           string     `db:"stderr"`
	LockedAt         *time.Time `db:"locked_at"`
	CreatedAt        time.Time  `db:"created_at"`
}
