package db

import (
	"database/sql"
	"fmt"
)

const (
	dbDriver = "mysql"
)

type DB struct {
	db *sql.DB
}

func New(dsn string) (*DB, error) {
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't ping the database: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) Save() error {
	return nil
}
