package submission

import (
	"context"
	"runner/internal/repository/models"

	"github.com/jmoiron/sqlx"
)

const (
	VerdictPending           = "pending"
	VerdictRunning           = "running"
	VerdictOK                = "ok"
	VerdictWrongAnswer       = "wrong_answer"
	VerdictRuntimeError      = "runtime_error"
	VerdictCompilationError  = "compilation_error"
	VerdictTimeLimitExceeded = "time_limit_exceeded"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) UpdateVerdict(ctx context.Context, id int32, verdict string, passedTestsCount int32, stderr string) error {
	query := "UPDATE submissions SET verdict = $1, passed_tests_count = $2, stderr = $3, locked_at = NULL WHERE id = $4"
	_, err := p.db.ExecContext(ctx, query, verdict, passedTestsCount, stderr, id)
	return err
}

func (p *Postgres) CreateFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput string) error {
	query := "INSERT INTO failed_tests (submission_id, input, expected_output, actual_output) VALUES ($1, $2, $3, $4)"
	_, err := p.db.ExecContext(ctx, query, submissionID, input, expectedOutput, actualOutput)
	return err
}

func (p *Postgres) GetPending(ctx context.Context) (models.Submission, error) {
	query := `WITH next_jobs AS (
			SELECT id
			FROM submissions
			WHERE verdict = 'pending' AND locked_at IS NULL
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE submissions
		SET verdict = 'running', locked_at = now()
		FROM next_jobs
		WHERE submissions.id = next_jobs.id
		RETURNING submissions.*`
	var submission models.Submission
	err := p.db.GetContext(ctx, &submission, query)
	return submission, err
}
