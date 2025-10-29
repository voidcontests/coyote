package domain

import "time"

type Problem struct {
	ID             int       `db:"id"`
	Charcode       string    `db:"charcode"`
	WriterID       int       `db:"writer_id"`
	WriterUsername string    `db:"writer_username"`
	Title          string    `db:"title"`
	Statement      string    `db:"statement"`
	Difficulty     string    `db:"difficulty"`
	Checker        string    `db:"checker"`
	TimeLimitMS    int       `db:"time_limit_ms"`
	MemoryLimitMB  int       `db:"memory_limit_mb"`
	CreatedAt      time.Time `db:"created_at"`
}
