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

	insertCustomer = "INSERT INTO customers " +
		"(customer_id, name, email) VALUES (?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE name=VALUE(name)"

	insertMember       = "INSERT INTO members (customer_id) VALUES (?)"
	updateMemberAccess = "UPDATE members SET access_id=? WHERE member_id=?"
	removeMember       = "DELETE FROM members WHERE customer_id=?"
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

func (d *DB) CreateCustomer(c types.Customer) error {
	if _, err := d.db.Exec(
		insertCustomer,
		c.CustomerId,
		c.Name,
		c.Email,
	); err != nil {
		return fmt.Errorf("error inserting customer: %w", err)
	}

	return nil
}

func (d *DB) CreateMember(customerId string) (int64, error) {
	r, err := d.db.Exec(insertMember, customerId)
	if err != nil {
		return 0, fmt.Errorf("error inserting member: %w", err)
	}

	return r.LastInsertId()
}

func (d *DB) UpdateMemberAccess(memberId int64, accessId string) error {
	if _, err := d.db.Exec(updateMemberAccess, accessId, memberId); err != nil {
		return fmt.Errorf("error updating member's access id: %w", err)
	}
	return nil
}

func (d *DB) RemoveMember(customerId string) error {
	if _, err := d.db.Exec(removeMember, customerId); err != nil {
		return fmt.Errorf("error removing member %s: %w", customerId, err)
	}
	return nil
}

func (d *DB) FindCustomer(customerId string) (*types.Customer, error) {
	r := d.db.QueryRow(
		"SELECT customer_id, name, email "+
			"FROM customers WHERE customer_id=?",
		customerId,
	)

	var c types.Customer
	if err := r.Scan(
		&c.CustomerId,
		&c.Name,
		&c.Email,
	); err != nil {
		return nil, fmt.Errorf(
			"error querying customer %q: %w",
			customerId,
			err,
		)
	}

	return &c, nil
}
