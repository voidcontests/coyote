package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidcontests/coyote/internal/domain"
)

type SubmissionRepository struct {
	pool *pgxpool.Pool
}

func NewSubmissionRepository(pool *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{pool: pool}
}

func (r *SubmissionRepository) UpdateVerdictAndStatus(ctx context.Context, submissionID int, verdict string, status string) error {
	query := `UPDATE submissions SET verdict = $1, status = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, verdict, status, submissionID)
	return err
}

func (r *SubmissionRepository) UpdateStatus(ctx context.Context, submissionID int, status string) error {
	query := `UPDATE submissions SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, submissionID)
	return err
}

func (r *SubmissionRepository) CreateTestingReportAndComplete(ctx context.Context, report *domain.TestingReport, status string, verdict string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `UPDATE submissions SET status = $1, verdict = $2 WHERE id = $3`
	_, err = tx.Exec(ctx, query, status, verdict, report.SubmissionID)
	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	query = `INSERT INTO testing_reports
		(submission_id, passed_tests_count, total_tests_count, first_failed_test_id, first_failed_test_output, stderr)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	err = tx.QueryRow(
		ctx,
		query,
		report.SubmissionID,
		report.PassedTestsCount,
		report.TotalTestsCount,
		report.FirstFailedTestID,
		report.FirstFailedTestOutput,
		report.Stderr,
	).Scan(&report.ID, &report.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert testing report: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
