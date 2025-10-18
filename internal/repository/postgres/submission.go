package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionRepository struct {
	pool *pgxpool.Pool
}

func NewSubmissionRepository(pool *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{pool: pool}
}

func (r *SubmissionRepository) SetResult(ctx context.Context, id int32, verdict string, passedTestsCount int32, stderr string) error {
	query := `UPDATE submissions SET verdict = $1, passed_tests_count = $2, stderr = $3 WHERE id = $4`
	_, err := r.pool.Exec(ctx, query, verdict, passedTestsCount, stderr, id)
	return err
}

func (r *SubmissionRepository) UpdateVerdict(ctx context.Context, id int32, verdict string) error {
	query := `UPDATE submissions SET verdict = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, verdict, id)
	return err
}

func (r *SubmissionRepository) CreateFailedTest(ctx context.Context, submissionID int32, input, expectedOutput, actualOutput string) error {
	query := `INSERT INTO failed_tests(submission_id, input, expected_output, actual_output) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, submissionID, input, expectedOutput, actualOutput)
	return err
}
