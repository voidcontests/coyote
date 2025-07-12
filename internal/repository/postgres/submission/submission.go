package submission

import (
	"context"
	"runner/internal/repository/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) UpdateVerdict(ctx context.Context, id int32, verdict string, passedTestsCount int32, stderr string) error {
	query := `UPDATE submissions
		SET verdict = $1,
			passed_tests_count = $2,
			stderr = $3,
			locked_at = NULL
		WHERE id = $4`
	_, err := p.pool.Exec(ctx, query, verdict, passedTestsCount, stderr, id)
	return err
}

func (p *Postgres) CreateFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput string) error {
	query := `INSERT INTO failed_tests(submission_id, input, expected_output, actual_output)
		VALUES ($1, $2, $3, $4)`
	_, err := p.pool.Exec(ctx, query, submissionID, input, expectedOutput, actualOutput)
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
	var s models.Submission
	err := p.pool.QueryRow(ctx, query).Scan(
		&s.ID,
		&s.EntryID,
		&s.ProblemID,
		&s.Verdict,
		&s.Answer,
		&s.Code,
		&s.Language,
		&s.PassedTestsCount,
		&s.Stderr,
		&s.LockedAt,
		&s.CreatedAt,
	)
	return s, err
}
