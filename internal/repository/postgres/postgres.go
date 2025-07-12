package postgres

import (
	"fmt"

	"runner/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func New(c *config.Postgres) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Name, c.Password, c.ModeSSL))
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, err
}
