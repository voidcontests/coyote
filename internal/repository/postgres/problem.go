package postgres

import (
	"context"
	"fmt"

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

func (r *ProblemRepository) GetByID(ctx context.Context, problemID int) (domain.Problem, error) {
	query := `SELECT
			p.id, p.writer_id, p.title, p.statement,
			p.difficulty, p.time_limit_ms, p.memory_limit_mb, p.created_at,
			u.username AS writer_username, p.checker
		FROM problems p
		JOIN users u ON u.id = p.writer_id
		WHERE p.id = $1`

	row := r.pool.QueryRow(ctx, query, problemID)

	var problem domain.Problem
	err := row.Scan(
		&problem.ID, &problem.WriterID, &problem.Title, &problem.Statement,
		&problem.Difficulty, &problem.TimeLimitMS, &problem.MemoryLimitMB, &problem.CreatedAt,
		&problem.WriterUsername, &problem.Checker,
	)

	return problem, err
}
