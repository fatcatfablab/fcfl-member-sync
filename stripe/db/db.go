package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
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

	db.SetConnMaxLifetime(60 * time.Second)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't ping the database: %w", err)
	}

	log.Printf("Connected to db")
	return &DB{db: db, dryRun: dryRun}, nil
}

func (d *DB) CreateMember(c types.Customer) error {
	if _, err := d.db.Exec(
		"INSERT INTO members "+
			"(customer_id, name, email) VALUES (?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE name=VALUE(name), email=VALUE(email)",
		c.CustomerId,
		c.Name,
		c.Email,
	); err != nil {
		return fmt.Errorf("error inserting member: %w", err)
	}

	return nil
}

func (d *DB) ActivateMember(customerId string) error {
	if _, err := d.db.Exec(
		"UPDATE members SET status='active' WHERE customer_id=?",
		customerId,
	); err != nil {
		return fmt.Errorf("error updating member status: %w", err)
	}

	return nil
}

func (d *DB) UpdateMemberAccess(memberId int64, accessId string) error {
	if _, err := d.db.Exec(
		"UPDATE members SET access_id=? WHERE member_id=?",
		accessId,
		memberId,
	); err != nil {
		return fmt.Errorf("error updating member's access id: %w", err)
	}
	return nil
}

func (d *DB) DeactivateMember(customerId string) error {
	if _, err := d.db.Exec(
		"UPDATE members SET status='not_active' WHERE customer_id=?",
		customerId,
	); err != nil {
		return fmt.Errorf("error removing member %s: %w", customerId, err)
	}
	return nil
}

func (d *DB) FindMemberByCustomerId(customerId string) (*types.Member, error) {
	r := d.db.QueryRow(
		"SELECT member_id, customer_id, access_id, name, email, status "+
			"FROM members WHERE customer_id=?",
		customerId,
	)

	var m types.Member
	if err := r.Scan(
		&m.MemberId,
		&m.CustomerId,
		&m.AccessId,
		&m.Name,
		&m.Email,
		&m.Status,
	); err != nil {
		return nil, fmt.Errorf(
			"error querying customer_id %q: %w",
			customerId,
			err,
		)
	}

	return &m, nil
}
