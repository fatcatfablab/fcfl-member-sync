package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
)

const (
	dbDriver = "mysql"
)

//go:embed schema/members.sql
var createTable string

type sqldb interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

type DB struct {
	db sqldb
}

func New(dsn string) (*DB, error) {
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	db.SetConnMaxLifetime(60 * time.Second)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't ping the database: %w", err)
	}

	if _, err := db.Exec(createTable); err != nil {
		return nil, fmt.Errorf("error creating table: %w", err)
	}

	log.Printf("Connected to db")
	return &DB{db: db}, nil
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
	return d.setMemberStatus(customerId, types.MemberStatusActive)
}

func (d *DB) DeactivateMember(customerId string) error {
	return d.setMemberStatus(customerId, types.MemberStatusNotActive)
}

func (d *DB) setMemberStatus(customerId string, status string) error {
	r, err := d.db.Exec(
		"UPDATE members SET status=? WHERE customer_id=?",
		status,
		customerId,
	)
	if err != nil {
		return fmt.Errorf("error updating member status: %w", err)
	}

	num, err := r.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking update rows affected: %w", err)
	}

	if num != 1 {
		return fmt.Errorf("unexpected number of rows affected: %d", num)
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
