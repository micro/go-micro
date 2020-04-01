// Package cockroach implements the cockroach store
package cockroach

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

// DefaultNamespace is the namespace that the sql store
// will use if no namespace is provided.
var (
	DefaultNamespace = "micro"
	DefaultPrefix    = "micro"
)

type sqlStore struct {
	db *sql.DB

	database string
	table    string

	list       *sql.Stmt
	readOne    *sql.Stmt
	readMany   *sql.Stmt
	readOffset *sql.Stmt
	write      *sql.Stmt
	delete     *sql.Stmt

	options store.Options
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
	rows, err := s.list.Query()
	var keys []string
	var timehelper pq.NullTime
	if err != nil {
		if err == sql.ErrNoRows {
			return keys, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		record := &store.Record{}
		if err := rows.Scan(&record.Key, &record.Value, &timehelper); err != nil {
			return keys, err
		}
		if timehelper.Valid {
			if timehelper.Time.Before(time.Now()) {
				// record has expired
				go s.Delete(record.Key)
			} else {
				record.Expiry = time.Until(timehelper.Time)
				keys = append(keys, record.Key)
			}
		} else {
			keys = append(keys, record.Key)
		}

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

// Read a single key
func (s *sqlStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var options store.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	if options.Prefix || options.Suffix {
		return s.read(key, options)
	}

	var records []*store.Record
	var timehelper pq.NullTime

	row := s.readOne.QueryRow(key)
	record := &store.Record{}
	if err := row.Scan(&record.Key, &record.Value, &timehelper); err != nil {
		if err == sql.ErrNoRows {
			return records, store.ErrNotFound
		}
		return records, err
	}
	if timehelper.Valid {
		if timehelper.Time.Before(time.Now()) {
			// record has expired
			go s.Delete(key)
			return records, store.ErrNotFound
		}
		record.Expiry = time.Until(timehelper.Time)
		records = append(records, record)
	} else {
		records = append(records, record)
	}

	return records, nil
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
	var err error
	if options.Limit != 0 {
		rows, err = s.readOffset.Query(pattern, options.Limit, options.Offset)
	} else {
		rows, err = s.readMany.Query(pattern)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return []*store.Record{}, nil
		}
		return []*store.Record{}, errors.Wrap(err, "sqlStore.read failed")
	}
	defer rows.Close()
	var records []*store.Record
	var timehelper pq.NullTime

	for rows.Next() {
		record := &store.Record{}
		if err := rows.Scan(&record.Key, &record.Value, &timehelper); err != nil {
			return records, err
		}
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
	var err error
	if r.Expiry != 0 {
		_, err = s.write.Exec(r.Key, r.Value, time.Now().Add(r.Expiry))
	} else {
		_, err = s.write.Exec(r.Key, r.Value, nil)
	}

	if err != nil {
		return errors.Wrap(err, "Couldn't insert record "+r.Key)
	}

	return nil
}

// Delete records with keys
func (s *sqlStore) Delete(key string, opts ...store.DeleteOption) error {
	result, err := s.delete.Exec(key)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (s *sqlStore) initDB() error {
	// Create the namespace's database
	_, err := s.db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s ;", s.database))
	if err != nil {
		return err
	}

	_, err = s.db.Exec(fmt.Sprintf("SET DATABASE = %s ;", s.database))
	if err != nil {
		return errors.Wrap(err, "Couldn't set database")
	}

	// Create a table for the namespace's prefix
	_, err = s.db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
	(
		key text NOT NULL,
		value bytea,
		expiry timestamp with time zone,
		CONSTRAINT %s_pkey PRIMARY KEY (key)
	);`, s.table, s.table))
	if err != nil {
		return errors.Wrap(err, "Couldn't create table")
	}

	// Create Index
	_, err = s.db.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%s" ON %s.%s USING btree ("key");`, "key_index_"+s.table, s.database, s.table))
	if err != nil {
		return err
	}

	list, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s;", s.database, s.table))
	if err != nil {
		return errors.Wrap(err, "List statement couldn't be prepared")
	}
	if s.list != nil {
		s.list.Close()
	}
	s.list = list
	readOne, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s WHERE key = $1;", s.database, s.table))
	if err != nil {
		return errors.Wrap(err, "ReadOne statement couldn't be prepared")
	}
	if s.readOne != nil {
		s.readOne.Close()
	}
	s.readOne = readOne
	readMany, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s WHERE key LIKE $1;", s.database, s.table))
	if err != nil {
		return errors.Wrap(err, "ReadMany statement couldn't be prepared")
	}
	if s.readMany != nil {
		s.readMany.Close()
	}
	s.readMany = readMany
	readOffset, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s WHERE key LIKE $1 ORDER BY key DESC LIMIT $2 OFFSET $3;", s.database, s.table))
	if err != nil {
		return errors.Wrap(err, "ReadOffset statement couldn't be prepared")
	}
	if s.readOffset != nil {
		s.readOffset.Close()
	}
	s.readOffset = readOffset
	write, err := s.db.Prepare(fmt.Sprintf(`INSERT INTO %s.%s(key, value, expiry)
		VALUES ($1, $2::bytea, $3)
		ON CONFLICT (key)
		DO UPDATE
		SET value = EXCLUDED.value, expiry = EXCLUDED.expiry;`, s.database, s.table))
	if err != nil {
		return errors.Wrap(err, "Write statement couldn't be prepared")
	}
	if s.write != nil {
		s.write.Close()
	}
	s.write = write
	delete, err := s.db.Prepare(fmt.Sprintf("DELETE FROM %s.%s WHERE key = $1;", s.database, s.table))
	if err != nil {
		return errors.Wrap(err, "Delete statement couldn't be prepared")
	}
	if s.delete != nil {
		s.delete.Close()
	}
	s.delete = delete

	return nil
}

func (s *sqlStore) configure() error {
	if len(s.options.Nodes) == 0 {
		s.options.Nodes = []string{"localhost:26257"}
	}

	namespace := s.options.Namespace
	if len(namespace) == 0 {
		namespace = DefaultNamespace
	}

	prefix := s.options.Prefix
	if len(prefix) == 0 {
		prefix = DefaultPrefix
	}

	// store.namespace must only contain letters, numbers and underscores
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return errors.New("error compiling regex for namespace")
	}
	namespace = reg.ReplaceAllString(namespace, "_")
	prefix = reg.ReplaceAllString(prefix, "_")

	source := s.options.Nodes[0]
	// check if it is a standard connection string eg: host=%s port=%d user=%s password=%s dbname=%s sslmode=disable
	// if err is nil which means it would be a URL like postgre://xxxx?yy=zz
	_, err = url.Parse(source)
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

	if s.db != nil {
		s.db.Close()
	}

	// save the values
	s.db = db
	s.database = namespace
	s.table = prefix

	// initialise the database
	return s.initDB()
}

func (s *sqlStore) String() string {
	return "cockroach"
}

func (s *sqlStore) Options() store.Options {
	return s.options
}

// NewStore returns a new micro Store backed by sql
func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	// new store
	s := new(sqlStore)
	// set the options
	s.options = options

	// best-effort configure the store
	s.configure()

	// return store
	return s
}
