package mysql

import (
	"database/sql"
	"fmt"
	"time"
	"unicode"

	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/store"
	"github.com/pkg/errors"
)

var (
	// DefaultDatabase is the database that the sql store will use if no database is provided.
	DefaultDatabase = "micro"
	// DefaultTable is the table that the sql store will use if no table is provided.
	DefaultTable = "micro"
)

type sqlStore struct {
	db *sql.DB

	database string
	table    string

	options store.Options

	readPrepare, writePrepare, deletePrepare *sql.Stmt
}

func (s *sqlStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	// reconfigure
	return s.configure()
}

func (s *sqlStore) Options() store.Options {
	return s.options
}

func (s *sqlStore) Close() error {
	return s.db.Close()
}

// List all the known records
func (s *sqlStore) List(opts ...store.ListOption) ([]string, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT `key`, value, expiry FROM %s.%s;", s.database, s.table))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var records []string
	var cachedTime time.Time

	for rows.Next() {
		record := &store.Record{}
		if err := rows.Scan(&record.Key, &record.Value, &cachedTime); err != nil {
			return nil, err
		}

		if cachedTime.Before(time.Now()) {
			// record has expired
			go s.Delete(record.Key)
		} else {
			records = append(records, record.Key)
		}
	}
	rowErr := rows.Close()
	if rowErr != nil {
		// transaction rollback or something
		return records, rowErr
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

	var records []*store.Record
	row := s.readPrepare.QueryRow(key)
	record := &store.Record{}
	var cachedTime time.Time

	if err := row.Scan(&record.Key, &record.Value, &cachedTime); err != nil {
		if err == sql.ErrNoRows {
			return records, store.ErrNotFound
		}
		return records, err
	}
	if cachedTime.Before(time.Now()) {
		// record has expired
		go s.Delete(key)
		return records, store.ErrNotFound
	}
	record.Expiry = time.Until(cachedTime)
	records = append(records, record)

	return records, nil
}

// Write records
func (s *sqlStore) Write(r *store.Record, opts ...store.WriteOption) error {
	timeCached := time.Now().Add(r.Expiry)
	_, err := s.writePrepare.Exec(r.Key, r.Value, timeCached, r.Value, timeCached)
	if err != nil {
		return errors.Wrap(err, "Couldn't insert record "+r.Key)
	}

	return nil
}

// Delete records with keys
func (s *sqlStore) Delete(key string, opts ...store.DeleteOption) error {
	result, err := s.deletePrepare.Exec(key)
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

	_, err = s.db.Exec(fmt.Sprintf("USE %s ;", s.database))
	if err != nil {
		return errors.Wrap(err, "Couldn't use database")
	}

	// Create a table for the namespace's prefix
	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (`key` varchar(255) primary key, value blob null, expiry timestamp not null);", s.table)
	_, err = s.db.Exec(createSQL)
	if err != nil {
		return errors.Wrap(err, "Couldn't create table")
	}

	// prepare
	s.readPrepare, _ = s.db.Prepare(fmt.Sprintf("SELECT `key`, value, expiry FROM %s.%s WHERE `key` = ?;", s.database, s.table))
	s.writePrepare, _ = s.db.Prepare(fmt.Sprintf("INSERT INTO %s.%s (`key`, value, expiry) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE `value`= ?, `expiry` = ?", s.database, s.table))
	s.deletePrepare, _ = s.db.Prepare(fmt.Sprintf("DELETE FROM %s.%s WHERE `key` = ?;", s.database, s.table))

	return nil
}

func (s *sqlStore) configure() error {
	nodes := s.options.Nodes
	if len(nodes) == 0 {
		nodes = []string{"localhost:3306"}
	}

	database := s.options.Database
	if len(database) == 0 {
		database = DefaultDatabase
	}

	table := s.options.Table
	if len(table) == 0 {
		table = DefaultTable
	}

	for _, r := range database {
		if !unicode.IsLetter(r) {
			return errors.New("store.namespace must only contain letters")
		}
	}

	source := nodes[0]
	// create source from first node
	db, err := sql.Open("mysql", source)
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
	s.database = database
	s.table = table

	// initialise the database
	return s.initDB()
}

func (s *sqlStore) String() string {
	return "mysql"
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
