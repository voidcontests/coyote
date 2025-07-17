package problem

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

func (p *Postgres) GetTestCases(ctx context.Context, problemID int32) ([]models.TestCase, error) {
	query := `SELECT id, problem_id, input, output, is_example FROM test_cases WHERE problem_id = $1`
	rows, err := p.pool.Query(ctx, query, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tcs []models.TestCase
	for rows.Next() {
		var tc models.TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output, &tc.IsExample); err != nil {
			return nil, err
		}
		tcs = append(tcs, tc)
	}

	return tcs, nil
}
