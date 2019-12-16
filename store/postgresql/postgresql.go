// Package postgresql implements a micro Store backed by sql
package postgresql

import (
	"database/sql"
	"fmt"
	"time"
	"unicode"

	"github.com/lib/pq"
	"github.com/micro/go-micro/store"
	"github.com/micro/go-micro/util/log"
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

func (s *sqlStore) initDB() {
	// Create "micro" schema
	schema, err := s.db.Prepare(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s ;", s.database))
	if err != nil {
		log.Fatal(err)
	}

	_, err = schema.Exec()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Couldn't create database"))
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
		log.Fatal(errors.Wrap(err, "SQL statement preparation failed"))
	}

	_, err = tableq.Exec()
	if err != nil {
		log.Fatal(errors.Wrap(err, "Couldn't create table"))
	}
}

// New returns a new micro Store backed by sql
func New(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	nodes := options.Nodes
	if len(nodes) == 0 {
		nodes = []string{"localhost:26257"}
	}

	namespace := options.Namespace
	if len(namespace) == 0 {
		namespace = DefaultNamespace
	}

	prefix := options.Prefix
	if len(prefix) == 0 {
		prefix = DefaultPrefix
	}

	for _, r := range namespace {
		if !unicode.IsLetter(r) {
			log.Fatal("store.namespace must only contain letters")
		}
	}

	// create source from first node
	source := fmt.Sprintf("host=%s", nodes[0])
	db, err := sql.Open("pq", source)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	s := &sqlStore{
		db:       db,
		database: namespace,
		table:    prefix,
	}

	return s
}
