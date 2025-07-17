package postgres

import (
	"context"
	"fmt"
	"time"

	"runner/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(c *config.Postgres) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.User, c.Password, c.Host, c.Port, c.Name, c.ModeSSL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, err
	}

	return dbpool, nil
}
