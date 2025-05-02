package db

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	dbDriver = "mysql"
)

type DB struct {
	db     *sql.DB
	dryRun bool
}

func New(dsn string, dryRun bool) (*DB, error) {
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't ping the database: %w", err)
	}

	log.Printf("Connected to db")
	return &DB{db: db, dryRun: dryRun}, nil
}

func (d *DB) Save() error {
	return nil
}
