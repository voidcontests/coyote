package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voidcontests/coyote/internal/domain"
)

type ProblemRepository struct {
	pool *pgxpool.Pool
}

func NewProblemRepository(pool *pgxpool.Pool) *ProblemRepository {
	return &ProblemRepository{pool: pool}
}

func (r *ProblemRepository) GetTestCases(ctx context.Context, problemID int) ([]domain.TestCase, error) {
	query := `SELECT id, problem_id, input, output, is_example FROM test_cases WHERE problem_id = $1`
	rows, err := r.pool.Query(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []domain.TestCase
	for rows.Next() {
		var tc domain.TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output, &tc.IsExample); err != nil {
			return nil, err
		}
		testCases = append(testCases, tc)
	}

	return testCases, nil
}

func (r *ProblemRepository) GetTimeLimit(ctx context.Context, problemID int) (time.Duration, error) {
	var timeLimitMS int
	query := `SELECT time_limit_ms FROM problems WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, problemID).Scan(&timeLimitMS)
	return time.Duration(timeLimitMS) * time.Millisecond, err
}
