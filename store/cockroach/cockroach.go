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
	log "github.com/micro/go-micro/v2/logger"
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
func (s *sqlStore) List() ([]*store.Record, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s;", s.database, s.table))
	var records []*store.Record
	var timehelper pq.NullTime
	if err != nil {
		if err == sql.ErrNoRows {
			return records, nil
		}
		return nil, err
	}
	defer rows.Close()
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

// Read all records with keys
func (s *sqlStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var options store.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	// TODO: make use of options.Prefix using WHERE key LIKE = ?

	q, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s WHERE key = $1;", s.database, s.table))
	if err != nil {
		return nil, err
	}

	var records []*store.Record
	var timehelper pq.NullTime

	row := q.QueryRow(key)
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

// Write records
func (s *sqlStore) Write(r *store.Record) error {
	q, err := s.db.Prepare(fmt.Sprintf(`INSERT INTO %s.%s(key, value, expiry)
		VALUES ($1, $2::bytea, $3)
		ON CONFLICT (key)
		DO UPDATE
		SET value = EXCLUDED.value, expiry = EXCLUDED.expiry;`, s.database, s.table))
	if err != nil {
		return err
	}

	if r.Expiry != 0 {
		_, err = q.Exec(r.Key, r.Value, time.Now().Add(r.Expiry))
	} else {
		_, err = q.Exec(r.Key, r.Value, nil)
	}

	if err != nil {
		return errors.Wrap(err, "Couldn't insert record "+r.Key)
	}

	return nil
}

// Delete records with keys
func (s *sqlStore) Delete(key string) error {
	q, err := s.db.Prepare(fmt.Sprintf("DELETE FROM %s.%s WHERE key = $1;", s.database, s.table))
	if err != nil {
		return err
	}

	result, err := q.Exec(key)
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

	return nil
}

func (s *sqlStore) configure() error {
	nodes := s.options.Nodes
	if len(nodes) == 0 {
		nodes = []string{"localhost:26257"}
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

	source := nodes[0]
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

// New returns a new micro Store backed by sql
func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	// new store
	s := new(sqlStore)
	// set the options
	s.options = options

	// configure the store
	if err := s.configure(); err != nil {
		log.Fatal(err)
	}

	// return store
	return s
}
