// Copyright 2020 Asim Aslam
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Original source: github.com/micro/go-plugins/v3/store/cockroach/cockroach.go

// Package postgres implements the postgres store
package postgres

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/store"
)

// DefaultDatabase is the namespace that the sql store
// will use if no namespace is provided.
var (
	DefaultDatabase = "micro"
	DefaultTable    = "micro"
	ErrNoConnection = errors.New("Database connection not initialised")
)

var (
	re = regexp.MustCompile("[^a-zA-Z0-9]+")

	// alternative ordering
	orderAsc  = "ORDER BY key ASC"
	orderDesc = "ORDER BY key DESC"

	// the sql statements we prepare and use
	statements = map[string]string{
		"list":          "SELECT key, value, metadata, expiry FROM %s.%s WHERE key LIKE $1 ORDER BY key ASC LIMIT $2 OFFSET $3;",
		"read":          "SELECT key, value, metadata, expiry FROM %s.%s WHERE key = $1;",
		"readMany":      "SELECT key, value, metadata, expiry FROM %s.%s WHERE key LIKE $1 ORDER BY key ASC;",
		"readOffset":    "SELECT key, value, metadata, expiry FROM %s.%s WHERE key LIKE $1 ORDER BY key ASC LIMIT $2 OFFSET $3;",
		"write":         "INSERT INTO %s.%s(key, value, metadata, expiry) VALUES ($1, $2::bytea, $3, $4) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, metadata = EXCLUDED.metadata, expiry = EXCLUDED.expiry;",
		"delete":        "DELETE FROM %s.%s WHERE key = $1;",
		"deleteExpired": "DELETE FROM %s.%s WHERE expiry < now();",
		"showTables":    "SELECT schemaname, tablename FROM pg_catalog.pg_tables WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema';",
	}
)

type sqlStore struct {
	options store.Options
	dbConn  *sql.DB

	sync.RWMutex
	// known databases
	databases map[string]bool
}

func (s *sqlStore) getDB(database, table string) (string, string) {
	if len(database) == 0 {
		if len(s.options.Database) > 0 {
			database = s.options.Database
		} else {
			database = DefaultDatabase
		}
	}

	if len(table) == 0 {
		if len(s.options.Table) > 0 {
			table = s.options.Table
		} else {
			table = DefaultTable
		}
	}

	// store.namespace must only contain letters, numbers and underscores
	database = re.ReplaceAllString(database, "_")
	table = re.ReplaceAllString(table, "_")

	return database, table
}

// createDB ensures that the DB and table have been created. It's used for lazy initialisation
// and will record which tables have been created to reduce calls to the DB
func (s *sqlStore) createDB(database, table string) error {
	database, table = s.getDB(database, table)

	s.Lock()
	defer s.Unlock()

	if _, ok := s.databases[database+":"+table]; ok {
		return nil
	}

	if err := s.initDB(database, table); err != nil {
		return err
	}

	s.databases[database+":"+table] = true
	return nil
}

// db returns a valid connection to the DB
func (s *sqlStore) db() (*sql.DB, error) {
	if s.dbConn == nil {
		return nil, ErrNoConnection
	}

	if err := s.dbConn.Ping(); err != nil {
		if !isBadConnError(err) {
			return nil, err
		}
		logger.Errorf("Error with DB connection, will reconfigure: %s", err)
		if err := s.configure(); err != nil {
			logger.Errorf("Error while reconfiguring client: %s", err)
			return nil, err
		}
	}

	return s.dbConn, nil
}

// isBadConnError returns true if the error is related to having a bad connection such that you need to reconnect
func isBadConnError(err error) bool {
	if err == nil {
		return false
	}
	if err == driver.ErrBadConn {
		return true
	}

	// heavy handed crude check for "connection reset by peer"
	if strings.Contains(err.Error(), syscall.ECONNRESET.Error()) {
		return true
	}

	// otherwise iterate through the error types
	switch t := err.(type) {
	case syscall.Errno:
		return t == syscall.ECONNRESET || t == syscall.ECONNABORTED || t == syscall.ECONNREFUSED
	case *net.OpError:
		return !t.Temporary()
	case net.Error:
		return !t.Temporary()
	}

	return false
}

func (s *sqlStore) initDB(database, table string) error {
	db, err := s.db()
	if err != nil {
		return err
	}
	// Create the namespace's database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", database))
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	var version string
	if err = db.QueryRow("select version()").Scan(&version); err == nil {
		if strings.Contains(version, "PostgreSQL") {
			_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", database))
			if err != nil {
				return err
			}
		}
	}

	// Create a table for the namespace's prefix
	_, err = db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s
	(
		key text NOT NULL,
		value bytea,
		metadata JSONB,
		expiry timestamp with time zone,
		CONSTRAINT %s_pkey PRIMARY KEY (key)
	);`, database, table, table))
	if err != nil {
		return errors.Wrap(err, "Couldn't create table")
	}

	// Create Index
	_, err = db.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%s" ON %s.%s USING btree ("key");`, "key_index_"+table, database, table))
	if err != nil {
		return err
	}

	// Create Metadata Index
	_, err = db.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%s" ON %s.%s USING GIN ("metadata");`, "metadata_index_"+table, database, table))
	if err != nil {
		return err
	}

	return nil
}

func (s *sqlStore) configure() error {
	if len(s.options.Nodes) == 0 {
		s.options.Nodes = []string{"postgresql://root@localhost:26257?sslmode=disable"}
	}

	source := s.options.Nodes[0]
	// check if it is a standard connection string eg: host=%s port=%d user=%s password=%s dbname=%s sslmode=disable
	// if err is nil which means it would be a URL like postgre://xxxx?yy=zz
	_, err := url.Parse(source)
	if err != nil {
		if !strings.Contains(source, " ") {
			source = fmt.Sprintf("host=%s", source)
		}
	}

	// create source from first node
	db, err := sql.Open("postgres", source)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	if s.dbConn != nil {
		s.dbConn.Close()
	}

	// save the values
	s.dbConn = db

	// get DB
	database, table := s.getDB(s.options.Database, s.options.Table)

	// initialise the database
	return s.initDB(database, table)
}

func (s *sqlStore) prepare(database, table, query string) (*sql.Stmt, error) {
	st, ok := statements[query]
	if !ok {
		return nil, errors.New("unsupported statement")
	}



	// get DB
	database, table = s.getDB(database, table)

	q := fmt.Sprintf(st, database, table)

	db, err := s.db()
	if err != nil {
		return nil, err
	}
	stmt, err := db.Prepare(q)
	if err != nil {
		return nil, err
	}
	return stmt, nil
}

func (s *sqlStore) Close() error {
	if s.dbConn != nil {
		return s.dbConn.Close()
	}
	return nil
}

func (s *sqlStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	// reconfigure
	return s.configure()
}

// List all the known records
func (s *sqlStore) List(opts ...store.ListOption) ([]string, error) {
	options := store.ListOptions{}

	for _, o := range opts {
		o(&options)
	}

	// create the db if not exists
	if err := s.createDB(options.Database, options.Table); err != nil {
		return nil, err
	}
	limit := sql.NullInt32{}
	offset := 0
	pattern := "%"
	if options.Prefix != "" || options.Suffix != "" {
		if options.Prefix != "" {
			pattern = options.Prefix + pattern
		}
		if options.Suffix != "" {
			pattern = pattern + options.Suffix
		}
	}
	if options.Offset > 0 {
		offset = int(options.Offset)
	}
	if options.Limit > 0 {
		limit = sql.NullInt32{Int32: int32(options.Limit), Valid: true}
	}

	st, err := s.prepare(options.Database, options.Table, "list")
	if err != nil {
		return nil, err
	}
	defer st.Close()

	rows, err := st.Query(pattern, limit, offset)
	if err != nil {

		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var keys []string
	records, err := s.rowsToRecords(rows)
	if err != nil {
		return nil, err
	}
	for _, k := range records {
		keys = append(keys, k.Key)
	}
	rowErr := rows.Close()
	if rowErr != nil {
		// transaction rollback or something
		return keys, rowErr
	}
	if err := rows.Err(); err != nil {
		return keys, err
	}
	return keys, nil
}

// rowToRecord converts from sql.Row to a store.Record. If the record has expired it will issue a delete in a separate goroutine
func (s *sqlStore) rowToRecord(row *sql.Row) (*store.Record, error) {
	var timehelper pq.NullTime
	record := &store.Record{}
	metadata := make(Metadata)

	if err := row.Scan(&record.Key, &record.Value, &metadata, &timehelper); err != nil {
		if err == sql.ErrNoRows {
			return record, store.ErrNotFound
		}
		return nil, err
	}

	// set the metadata
	record.Metadata = toMetadata(&metadata)
	if timehelper.Valid {
		if timehelper.Time.Before(time.Now()) {
			// record has expired
			go s.Delete(record.Key)
			return nil, store.ErrNotFound
		}
		record.Expiry = time.Until(timehelper.Time)

	}
	return record, nil
}

// rowsToRecords converts from sql.Rows to  []*store.Record. If a record has expired it will issue a delete in a separate goroutine
func (s *sqlStore) rowsToRecords(rows *sql.Rows) ([]*store.Record, error) {
	var records []*store.Record
	var timehelper pq.NullTime

	for rows.Next() {
		record := &store.Record{}
		metadata := make(Metadata)

		if err := rows.Scan(&record.Key, &record.Value, &metadata, &timehelper); err != nil {
			return records, err
		}

		// set the metadata
		record.Metadata = toMetadata(&metadata)

		if timehelper.Valid {
			if timehelper.Time.Before(time.Now()) {
				// record has expired
				go s.Delete(record.Key)
			} else {
				record.Expiry = time.Until(timehelper.Time)
				records = append(records, record)
			}
		} else {
			records = append(records, record)
		}
	}
	return records, nil
}

// Read a single key
func (s *sqlStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	options := store.ReadOptions{}
	for _, o := range opts {
		o(&options)
	}

	// create the db if not exists
	if err := s.createDB(options.Database, options.Table); err != nil {
		return nil, err
	}

	if options.Prefix || options.Suffix {
		return s.read(key, options)
	}

	st, err := s.prepare(options.Database, options.Table, "read")
	if err != nil {
		return nil, err
	}
	defer st.Close()

	row := st.QueryRow(key)
	record, err := s.rowToRecord(row)
	if err != nil {
		return nil, err
	}
	var records []*store.Record
	return append(records, record), nil
}

// Read Many records
func (s *sqlStore) read(key string, options store.ReadOptions) ([]*store.Record, error) {
	pattern := "%"
	if options.Prefix {
		pattern = key + pattern
	}
	if options.Suffix {
		pattern = pattern + key
	}

	var rows *sql.Rows
	var st *sql.Stmt
	var err error

	if options.Limit != 0 {
		st, err = s.prepare(options.Database, options.Table, "readOffset")
		if err != nil {
			return nil, err
		}
		defer st.Close()

		rows, err = st.Query(pattern, options.Limit, options.Offset)
	} else {
		st, err = s.prepare(options.Database, options.Table, "readMany")
		if err != nil {
			return nil, err
		}
		defer st.Close()

		rows, err = st.Query(pattern)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return []*store.Record{}, nil
		}
		return []*store.Record{}, errors.Wrap(err, "sqlStore.read failed")
	}

	defer rows.Close()

	records, err := s.rowsToRecords(rows)
	if err != nil {
		return nil, err
	}
	rowErr := rows.Close()
	if rowErr != nil {
		// transaction rollback or something
		return records, rowErr
	}
	if err := rows.Err(); err != nil {
		return records, err
	}

	return records, nil
}

// Write records
func (s *sqlStore) Write(r *store.Record, opts ...store.WriteOption) error {
	var options store.WriteOptions
	for _, o := range opts {
		o(&options)
	}

	// create the db if not exists
	if err := s.createDB(options.Database, options.Table); err != nil {
		return err
	}

	st, err := s.prepare(options.Database, options.Table, "write")
	if err != nil {
		return err
	}
	defer st.Close()

	metadata := make(Metadata)
	for k, v := range r.Metadata {
		metadata[k] = v
	}

	var expiry time.Time
	if r.Expiry != 0 {
		expiry = time.Now().Add(r.Expiry)
	}

	if expiry.IsZero() {
		_, err = st.Exec(r.Key, r.Value, metadata, nil)
	} else {
		_, err = st.Exec(r.Key, r.Value, metadata, expiry)
	}

	if err != nil {
		return errors.Wrap(err, "Couldn't insert record "+r.Key)
	}

	return nil
}

// Delete records with keys
func (s *sqlStore) Delete(key string, opts ...store.DeleteOption) error {
	var options store.DeleteOptions
	for _, o := range opts {
		o(&options)
	}

	// create the db if not exists
	if err := s.createDB(options.Database, options.Table); err != nil {
		return err
	}

	st, err := s.prepare(options.Database, options.Table, "delete")
	if err != nil {
		return err
	}
	defer st.Close()

	result, err := st.Exec(key)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (s *sqlStore) Options() store.Options {
	return s.options
}

func (s *sqlStore) String() string {
	return "cockroach"
}

// NewStore returns a new micro Store backed by sql
func NewStore(opts ...store.Option) store.Store {
	options := store.Options{
		Database: DefaultDatabase,
		Table:    DefaultTable,
	}

	for _, o := range opts {
		o(&options)
	}

	// new store
	s := new(sqlStore)
	// set the options
	s.options = options
	// mark known databases
	s.databases = make(map[string]bool)
	// best-effort configure the store
	if err := s.configure(); err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error("Error configuring store ", err)
		}
	}
	go s.expiryLoop()
	// return store
	return s
}

func (s *sqlStore) expiryLoop() {
	for {
		s.expireRows()
		time.Sleep(1 * time.Hour)
	}
}

func (s *sqlStore) expireRows() error {
	db, err := s.db()
	if err != nil {
		logger.Errorf("Error getting DB connection %s", err)
		return err
	}
	stmt, err := db.Prepare(statements["showTables"])
	if err != nil {
		logger.Errorf("Error prepping show tables query %s", err)
		return err
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		logger.Errorf("Error running show tables query %s", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			logger.Errorf("Error parsing result %s", err)
			return err
		}
		db, err = s.db()
		if err != nil {
			logger.Errorf("Error prepping delete expired query %s", err)
			return err
		}
		delStmt, err := db.Prepare(fmt.Sprintf(statements["deleteExpired"], schemaName, tableName))
		if err != nil {
			logger.Errorf("Error prepping delete expired query %s", err)
			return err
		}
		defer delStmt.Close()
		res, err := delStmt.Exec()
		if err != nil {
			logger.Errorf("Error cleaning up %s", err)
			return err
		}

		r, _ := res.RowsAffected()
		logger.Infof("Cleaning up %s %s: %d rows deleted", schemaName, tableName, r)

	}
	return nil
}
