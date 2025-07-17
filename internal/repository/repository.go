package repository

import (
	"runner/internal/repository/postgres/problem"
	"runner/internal/repository/postgres/submission"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool       *pgxpool.Pool
	Submission *submission.Postgres
	Problem    *problem.Postgres
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:       pool,
		Submission: submission.New(pool),
		Problem:    problem.New(pool),
	}
}
