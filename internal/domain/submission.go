package domain

import "time"

type Submission struct {
	ID               int32
	EntryID          int32
	ProblemID        int32
	Verdict          string
	Answer           string
	Code             string
	Language         string
	PassedTestsCount int32
	Stderr           string
	CreatedAt        time.Time
}

type TestCase struct {
	ID        int32
	ProblemID int32
	Input     string
	Output    string
	IsExample bool
}

type FailedTest struct {
	Input          string
	ExpectedOutput string
	ActualOutput   string
}
