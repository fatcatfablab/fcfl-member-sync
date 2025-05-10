package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

const (
	dbDriver = "mysql"

	createTableDeactivations = `CREATE TABLE IF NOT EXISTS deactivations (
		stripe_id VARCHAR(255),
		dt DATETIME NOT NULL,
		PRIMARY KEY (stripe_id)
	);`
)

var insertDeactivation = "INSERT INTO deactivations (stripe_id, dt) VALUES (?, ?)"

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
	for _, stmt := range []string{createTableDeactivations} {
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	return &DB{db: db, dryRun: dryRun}, nil
}

func (d *DB) Save(stripeID string, t time.Time) error {
	if _, err := d.db.Exec(insertDeactivation, stripeID, t); err != nil {
		return fmt.Errorf("error inserting deactivation record: %w", err)
	}

	return nil
}
