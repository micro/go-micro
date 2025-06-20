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

// Package pgx implements the postgres store with pgx driver
package pgx

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/store"
)

const defaultDatabase = "micro"
const defaultTable = "micro"

type sqlStore struct {
	options store.Options
	re      *regexp.Regexp
	sync.Mutex
	// known databases
	databases map[string]DB
}

func (s *sqlStore) getDB(database, table string) (string, string) {
	if len(database) == 0 {
		if len(s.options.Database) > 0 {
			database = s.options.Database
		} else {
			database = defaultDatabase
		}
	}

	if len(table) == 0 {
		if len(s.options.Table) > 0 {
			table = s.options.Table
		} else {
			table = defaultTable
		}
	}

	// store.namespace must only contain letters, numbers and underscores
	database = s.re.ReplaceAllString(database, "_")
	table = s.re.ReplaceAllString(table, "_")

	return database, table
}

func (s *sqlStore) db(database, table string) (*pgxpool.Pool, Queries, error) {
	s.Lock()
	defer s.Unlock()

	database, table = s.getDB(database, table)

	if _, ok := s.databases[database]; !ok {
		err := s.initDB(database)
		if err != nil {
			return nil, Queries{}, err
		}
	}
	dbObj := s.databases[database]
	if _, ok := dbObj.tables[table]; !ok {
		err := s.initTable(database, table)
		if err != nil {
			return nil, Queries{}, err
		}
	}

	return dbObj.conn, dbObj.tables[table], nil
}

func (s *sqlStore) initTable(database, table string) error {
	db := s.databases[database].conn

	_, err := db.Exec(s.options.Context, fmt.Sprintf(createTable, database, table))
	if err != nil {
		return errors.Wrap(err, "cannot create table")
	}

	_, err = db.Exec(s.options.Context, fmt.Sprintf(createMDIndex, table, database, table))
	if err != nil {
		return errors.Wrap(err, "cannot create metadata index")
	}

	_, err = db.Exec(s.options.Context, fmt.Sprintf(createExpiryIndex, table, database, table))
	if err != nil {
		return errors.Wrap(err, "cannot create expiry index")
	}

	s.databases[database].tables[table] = NewQueries(database, table)

	return nil
}

func (s *sqlStore) initDB(database string) error {
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

	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		return err
	}

	db, err := pgxpool.ConnectConfig(s.options.Context, config)
	if err != nil {
		return err
	}

	if err = db.Ping(s.options.Context); err != nil {
		return err
	}

	_, err = db.Exec(s.options.Context, fmt.Sprintf(createSchema, database))
	if err != nil {
		return err
	}

	if len(database) == 0 {
		if len(s.options.Database) > 0 {
			database = s.options.Database
		} else {
			database = defaultDatabase
		}
	}

	// save the values
	s.databases[database] = DB{
		conn:   db,
		tables: make(map[string]Queries),
	}

	return nil
}

func (s *sqlStore) Close() error {
	for _, obj := range s.databases {
		obj.conn.Close()
	}
	return nil
}

func (s *sqlStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	_, _, err := s.db(s.options.Database, s.options.Table)
	return err
}

// List all the known records
func (s *sqlStore) List(opts ...store.ListOption) ([]string, error) {
	options := store.ListOptions{}

	for _, o := range opts {
		o(&options)
	}
	db, queries, err := s.db(options.Database, options.Table)
	if err != nil {
		return nil, err
	}
	pattern := "%"
	if options.Prefix != "" {
		pattern = options.Prefix + pattern
	}
	if options.Suffix != "" {
		pattern = pattern + options.Suffix
	}

	var rows pgx.Rows
	if options.Limit > 0 {
		rows, err = db.Query(s.options.Context, queries.ListAscLimit, pattern, options.Limit, options.Offset)

	} else {

		rows, err = db.Query(s.options.Context, queries.ListAsc, pattern)

	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	keys := make([]string, 0, 10)
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// rowToRecord converts from pgx.Row to a store.Record
func (s *sqlStore) rowToRecord(row pgx.Row) (*store.Record, error) {
	var expiry *time.Time
	record := &store.Record{}
	metadata := make(Metadata)

	if err := row.Scan(&record.Key, &record.Value, &metadata, &expiry); err != nil {
		if err == sql.ErrNoRows {
			return record, store.ErrNotFound
		}
		return nil, err
	}

	// set the metadata
	record.Metadata = toMetadata(&metadata)
	if expiry != nil {
		record.Expiry = time.Until(*expiry)
	}

	return record, nil
}

// rowsToRecords converts from pgx.Rows to []*store.Record
func (s *sqlStore) rowsToRecords(rows pgx.Rows) ([]*store.Record, error) {
	var records []*store.Record

	for rows.Next() {
		var expiry *time.Time
		record := &store.Record{}
		metadata := make(Metadata)

		if err := rows.Scan(&record.Key, &record.Value, &metadata, &expiry); err != nil {
			return records, err
		}

		// set the metadata
		record.Metadata = toMetadata(&metadata)
		if expiry != nil {
			record.Expiry = time.Until(*expiry)
		}
		records = append(records, record)
	}
	return records, nil
}

// Read a single key
func (s *sqlStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	options := store.ReadOptions{}
	for _, o := range opts {
		o(&options)
	}

	db, queries, err := s.db(options.Database, options.Table)
	if err != nil {
		return nil, err
	}

	// read one record
	if !options.Prefix && !options.Suffix {
		row := db.QueryRow(s.options.Context, queries.ReadOne, key)
		record, err := s.rowToRecord(row)
		if err != nil {
			return nil, err
		}
		return []*store.Record{record}, nil
	}

	// read by pattern
	pattern := "%"
	if options.Prefix {
		pattern = key + pattern
	}
	if options.Suffix {
		pattern = pattern + key
	}

	var rows pgx.Rows
	if options.Limit > 0 {

		rows, err = db.Query(s.options.Context, queries.ListAscLimit, pattern, options.Limit, options.Offset)

	} else {

		rows, err = db.Query(s.options.Context, queries.ListAsc, pattern)

	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	return s.rowsToRecords(rows)
}

// Write records
func (s *sqlStore) Write(r *store.Record, opts ...store.WriteOption) error {
	var options store.WriteOptions
	for _, o := range opts {
		o(&options)
	}

	db, queries, err := s.db(options.Database, options.Table)
	if err != nil {
		return err
	}

	metadata := make(Metadata)
	for k, v := range r.Metadata {
		metadata[k] = v
	}

	if r.Expiry != 0 {
		_, err = db.Exec(s.options.Context, queries.Write, r.Key, r.Value, metadata, time.Now().Add(r.Expiry))
	} else {
		_, err = db.Exec(s.options.Context, queries.Write, r.Key, r.Value, metadata, nil)
	}
	if err != nil {
		return errors.Wrap(err, "cannot upsert record "+r.Key)
	}

	return nil
}

// Delete records with keys
func (s *sqlStore) Delete(key string, opts ...store.DeleteOption) error {
	var options store.DeleteOptions
	for _, o := range opts {
		o(&options)
	}

	db, queries, err := s.db(options.Database, options.Table)
	if err != nil {
		return err
	}

	_, err = db.Exec(s.options.Context, queries.Delete, key)
	return err
}

func (s *sqlStore) Options() store.Options {
	return s.options
}

func (s *sqlStore) String() string {
	return "pgx"
}

// NewStore returns a new micro Store backed by sql
func NewStore(opts ...store.Option) store.Store {
	options := store.Options{
		Database: defaultDatabase,
		Table:    defaultTable,
	}

	for _, o := range opts {
		o(&options)
	}

	// new store
	s := new(sqlStore)
	s.options = options
	s.databases = make(map[string]DB)
	s.re = regexp.MustCompile("[^a-zA-Z0-9]+")

	go s.expiryLoop()
	// return store
	return s
}

func (s *sqlStore) expiryLoop() {
	for {
		err := s.expireRows()
		if err != nil {
			logger.Errorf("error cleaning up %s", err)
		}
		time.Sleep(1 * time.Hour)
	}
}

func (s *sqlStore) expireRows() error {
	for database, dbObj := range s.databases {
		db := dbObj.conn
		for table, queries := range dbObj.tables {
			res, err := db.Exec(s.options.Context, queries.DeleteExpired)
			if err != nil {
				logger.Errorf("Error cleaning up %s", err)
				return err
			}
			logger.Infof("Cleaning up %s %s: %d rows deleted", database, table, res.RowsAffected())
		}
	}

	return nil
}
