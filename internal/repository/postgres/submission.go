package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/coyote/internal/domain"
)

type SubmissionRepository struct {
	pool *pgxpool.Pool
}

func NewSubmissionRepository(pool *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{pool: pool}
}

func (r *SubmissionRepository) UpdateVerdict(ctx context.Context, id int32, verdict string, passedTestsCount int32, stderr string) error {
	query := `UPDATE submissions SET verdict = $1, passed_tests_count = $2, stderr = $3 WHERE id = $4`
	_, err := r.pool.Exec(ctx, query, verdict, passedTestsCount, stderr, id)
	return err
}

func (r *SubmissionRepository) CreateFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput string) error {
	query := `INSERT INTO failed_tests(submission_id, input, expected_output, actual_output) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, submissionID, input, expectedOutput, actualOutput)
	return err
}

func (r *SubmissionRepository) GetPending(ctx context.Context) (domain.Submission, error) {
	query := `WITH next_jobs AS (
			SELECT id
			FROM submissions
			WHERE verdict = 'pending'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE submissions
		SET verdict = 'running'
		FROM next_jobs
		WHERE submissions.id = next_jobs.id
		RETURNING submissions.*`

	var s domain.Submission
	err := r.pool.QueryRow(ctx, query).Scan(
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
