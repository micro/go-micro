// Package postgresql implements a micro Store backed by sql
package postgresql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
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
	options.Options
}

// List all the known records
func (s *sqlStore) List() ([]*store.Record, error) {
	q, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s;", s.database, s.table))
	if err != nil {
		return nil, err
	}
	var records []*store.Record
	var timehelper pq.NullTime
	rows, err := q.Query()
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
func (s *sqlStore) Read(keys ...string) ([]*store.Record, error) {
	q, err := s.db.Prepare(fmt.Sprintf("SELECT key, value, expiry FROM %s.%s WHERE key = $1;", s.database, s.table))
	if err != nil {
		return nil, err
	}
	var records []*store.Record
	var timehelper pq.NullTime
	for _, key := range keys {
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
	}
	return records, nil
}

// Write records
func (s *sqlStore) Write(rec ...*store.Record) error {
	q, err := s.db.Prepare(fmt.Sprintf(`INSERT INTO %s.%s(key, value, expiry)
		VALUES ($1, $2::bytea, $3)
		ON CONFLICT (key)
		DO UPDATE
		SET value = EXCLUDED.value, expiry = EXCLUDED.expiry;`, s.database, s.table))
	if err != nil {
		return err
	}
	for _, r := range rec {
		var err error
		if r.Expiry != 0 {
			_, err = q.Exec(r.Key, r.Value, time.Now().Add(r.Expiry))
		} else {
			_, err = q.Exec(r.Key, r.Value, nil)
		}
		if err != nil {
			return errors.Wrap(err, "Couldn't insert record "+r.Key)
		}
	}

	return nil
}

// Delete records with keys
func (s *sqlStore) Delete(keys ...string) error {
	q, err := s.db.Prepare(fmt.Sprintf("DELETE FROM %s.%s WHERE key = $1;", s.database, s.table))
	if err != nil {
		return err
	}
	for _, key := range keys {
		result, err := q.Exec(key)
		if err != nil {
			return err
		}
		_, err = result.RowsAffected()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlStore) initDB(options options.Options) error {
	// Get the store.namespace option, or use sql.DefaultNamespace
	namespaceOpt, found := options.Values().Get("store.namespace")
	if !found {
		s.database = DefaultNamespace
	} else {
		if namespace, ok := namespaceOpt.(string); ok {
			s.database = namespace
		} else {
			return errors.New("store.namespace option must be a string")
		}
	}
	// Get the store.namespace option, or use sql.DefaultNamespace
	prefixOpt, found := options.Values().Get("store.prefix")
	if !found {
		s.table = DefaultPrefix
	} else {
		if prefix, ok := prefixOpt.(string); ok {
			s.table = prefix
		} else {
			return errors.New("store.namespace option must be a string")
		}
	}

	// Create "micro" schema
	schema, err := s.db.Prepare(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s ;", s.database))
	if err != nil {
		return err
	}
	_, err = schema.Exec()
	if err != nil {
		return errors.Wrap(err, "Couldn't create database")
	}

	// Create a table for the Store namespace
	tableq, err := s.db.Prepare(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s
	(
		key text COLLATE "default" NOT NULL,
		value bytea,
		expiry timestamp with time zone,
		CONSTRAINT %s_pkey PRIMARY KEY (key)
	);`, s.database, s.table, s.table))
	if err != nil {
		return errors.Wrap(err, "SQL statement preparation failed")
	}
	_, err = tableq.Exec()
	if err != nil {
		return errors.Wrap(err, "Couldn't create table")
	}

	return nil
}

// New returns a new micro Store backed by sql
func New(opts ...options.Option) (store.Store, error) {
	options := options.NewOptions(opts...)
	driver, dataSourceName, err := validateOptions(options)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(dataSourceName, " ") {
		dataSourceName = fmt.Sprintf("host=%s", dataSourceName)
	}
	db, err := sql.Open(driver, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &sqlStore{
		db: db,
	}

	return s, s.initDB(options)
}

// validateOptions checks whether the provided options are valid, then returns the driver
// and data source name.
func validateOptions(options options.Options) (driver, dataSourceName string, err error) {
	driverOpt, found := options.Values().Get("store.sql.driver")
	if !found {
		return "", "", errors.New("No store.sql.driver option specified")
	}
	nodesOpt, found := options.Values().Get("store.nodes")
	if !found {
		return "", "", errors.New("No store.nodes option specified (expected a database connection string)")
	}
	driver, ok := driverOpt.(string)
	if !ok {
		return "", "", errors.New("store.sql.driver option must be a string")
	}
	nodes, ok := nodesOpt.([]string)
	if !ok {
		return "", "", errors.New("store.nodes option must be a []string")
	}
	if len(nodes) != 1 {
		return "", "", errors.New("expected only 1 store.nodes option")
	}
	namespaceOpt, found := options.Values().Get("store.namespace")
	if found {
		namespace, ok := namespaceOpt.(string)
		if !ok {
			return "", "", errors.New("store.namespace must me a string")
		}
		for _, r := range namespace {
			if !unicode.IsLetter(r) {
				return "", "", errors.New("store.namespace must only contain letters")
			}
		}
	}
	return driver, nodes[0], nil
}
