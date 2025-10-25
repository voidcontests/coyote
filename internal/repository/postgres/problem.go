package postgres

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("failed to query test cases: %w", err)
	}
	defer rows.Close()

	testCases := make([]domain.TestCase, 0, 10)
	for rows.Next() {
		var tc domain.TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output, &tc.IsExample); err != nil {
			return nil, fmt.Errorf("failed to scan test case: %w", err)
		}
		testCases = append(testCases, tc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating test cases: %w", err)
	}

	return testCases, nil
}

func (r *ProblemRepository) GetTimeLimit(ctx context.Context, problemID int) (time.Duration, error) {
	var timeLimitMS int
	query := `SELECT time_limit_ms FROM problems WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, problemID).Scan(&timeLimitMS)
	if err != nil {
		return 0, fmt.Errorf("failed to get time limit: %w", err)
	}
	return time.Duration(timeLimitMS) * time.Millisecond, nil
}
