package problem

import (
	"context"

	"runner/internal/repository/models"

	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) GetTestCases(ctx context.Context, problemID int32) ([]models.TestCase, error) {
	var err error
	var tcs []models.TestCase

	query := `SELECT * FROM test_cases WHERE problem_id = $1`
	err = p.db.SelectContext(ctx, &tcs, query, problemID)
	if err != nil {
		return nil, err
	}

	return tcs, nil
}
