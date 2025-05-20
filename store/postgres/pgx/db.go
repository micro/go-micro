package pgx

import "github.com/jackc/pgx/v4/pgxpool"

type DB struct {
	conn   *pgxpool.Pool
	tables map[string]Queries
}
