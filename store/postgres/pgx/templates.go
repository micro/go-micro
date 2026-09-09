package pgx

// init

const createSchema = "CREATE SCHEMA IF NOT EXISTS %s"
const createTable = `CREATE TABLE IF NOT EXISTS %s.%s
(
	key text primary key,
	value bytea,
	metadata JSONB,
	expiry timestamp with time zone
)`
const createMDIndex = `create index if not exists idx_md_%s ON %s.%s USING GIN (metadata)`
const createExpiryIndex = `create index if not exists idx_expiry_%s on %s.%s (expiry) where (expiry IS NOT NULL)`

// base queries
const (
	list     = "SELECT key FROM %s.%s WHERE key LIKE $1 and (expiry < now() or expiry isnull)"
	readOne  = "SELECT key, value, metadata, expiry FROM %s.%s WHERE key = $1 and (expiry < now() or expiry isnull)"
	readMany = "SELECT key, value, metadata, expiry FROM %s.%s WHERE key LIKE $1 and (expiry < now() or expiry isnull)"
	write    = `INSERT INTO %s.%s(key, value, metadata, expiry)
VALUES ($1, $2::bytea, $3, $4)
ON CONFLICT (key)
DO UPDATE
SET value = EXCLUDED.value, metadata = EXCLUDED.metadata, expiry = EXCLUDED.expiry`
	deleteRecord  = "DELETE FROM %s.%s WHERE key = $1"
	deleteExpired = "DELETE FROM %s.%s WHERE expiry < now()"
)

// suffixes
const (
	limit = " LIMIT $2 OFFSET $3"
	asc   = " ORDER BY key ASC"
	desc  = " ORDER BY key DESC"
)
