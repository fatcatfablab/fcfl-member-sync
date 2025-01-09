//go:build civicrm

package userlist

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
)

type queryRow struct {
	Id        int
	FirstName string
	LastName  string
	CardId    int
}

var (
	db    *sql.DB
	query = `
		SELECT co.id, co.first_name, co.last_name, ca.card_id
		FROM civicrm_contact co
		JOIN civicrm_membership m ON co.id=m.contact_id
		JOIN civicrm_accesscard_cards ca on co.id=ca.contact_id
		WHERE m.status_id < 4
		ORDER BY co.id;
	`
	initialized = false
)

func Init(dsn string) error {
	log.Printf("Setting up db connection")

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("couldn't connect to db: %w", err)
	}

	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("couldn't ping db: %w", err)
	}
	initialized = true
	return nil
}

func List(ctx context.Context) (*pb.MemberList, error) {
	if !initialized {
		panic("List called before Init")
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying db: %w", err)
	}

	res := pb.MemberList{}
	for rows.Next() {
		var id int
		var firstName, lastName, cardId *string

		if err := rows.Scan(&id, &firstName, &lastName, &cardId); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		m := pb.Member{}
		if firstName != nil {
			m.FirstName = *firstName
		}
		if lastName != nil {
			m.LastName = *lastName
		}
		if cardId != nil {
			m.CardId = *cardId
		}
		m.Id = int32(id)
		res.Members = append(res.Members, &m)
	}

	return &res, nil
}
