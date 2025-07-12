package repository

import (
	"runner/internal/repository/postgres/problem"
	"runner/internal/repository/postgres/submission"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db         *sqlx.DB
	Submission *submission.Postgres
	Problem    *problem.Postgres
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		db:         db,
		Submission: submission.New(db),
		Problem:    problem.New(db),
	}
}

func (r *Repository) Shutdown() error {
	return r.db.Close()
}
