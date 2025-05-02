package userlist

import (
	"context"
	"database/sql"
	"log"
	"path"
	"testing"

	pb "github.com/fatcatfablab/fcfl-member-sync/proto"
	_ "github.com/mattn/go-sqlite3"
)

const (
	driver        = "sqlite3"
	createContact = `CREATE TABLE civicrm_contact (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL
	) STRICT`
	createMembership = `CREATE TABLE civicrm_membership (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		contact_id INTEGER REFERENCES civicrm_contact (id),
		status_id INTEGER NOT NULL
	) STRICT`
	createEmail = `CREATE TABLE civicrm_email (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		contact_id INTEGER REFERENCES civicrm_contact (id),
		email TEXT NOT NULL,
		is_billing INTEGER NOT NULL
	) STRICT`
	createStripeCustomers = `CREATE TABLE civicrm_stripe_customers (
		email TEXT PRIMARY KEY,
		id TEXT NOT NULL
	) STRICT`
)

type dbEntry struct {
	contactId int
	firstName string
	lastName  string
	statusId  int
	email     string
	stripeId  string
}

func initDb(t *testing.T, entries []dbEntry) string {
	dsn := path.Join(t.TempDir(), "civicrm-tests.sqlite")
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatal(err)
	}

	for _, create := range []string{
		createContact, createMembership, createEmail, createStripeCustomers,
	} {
		_, err = db.Exec(create)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, e := range entries {
		err = insertEntry(db, e)
		if err != nil {
			t.Fatal(err)
		}
	}

	return dsn
}

func insertEntry(db *sql.DB, e dbEntry) error {
	_, err := db.Exec(
		`INSERT INTO civicrm_contact (id, first_name, last_name)
		VALUES (?, ?, ?)`,
		e.contactId,
		e.firstName,
		e.lastName,
	)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO civicrm_membership (contact_id, status_id)
		VALUES (?, ?)`,
		e.contactId,
		e.statusId,
	)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO civicrm_email (contact_id, email, is_billing)
		VALUES (?, ?, 1)`,
		e.contactId,
		e.email,
	)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO civicrm_stripe_customers (email, id) VALUES (?, ?)`,
		e.email,
		e.stripeId,
	)

	return err
}

func cmpMemberLists(want []*pb.Member, got []*pb.Member) bool {
	if len(want) != len(got) {
		log.Printf("want: %d items", len(want))
		log.Printf("got : %d items", len(got))
		return false
	}

	wMap := toMap(want)
	gMap := toMap(got)

	for k := range wMap {
		if wMap[k].Id != gMap[k].Id ||
			wMap[k].FirstName != gMap[k].FirstName ||
			wMap[k].LastName != gMap[k].LastName ||
			wMap[k].StripeId != gMap[k].StripeId {
			log.Printf("want: %+v", wMap[k])
			log.Printf("got : %+v", gMap[k])
			return false
		}
	}

	return true
}

func toMap(list []*pb.Member) map[int32]pb.Member {
	memberMap := make(map[int32]pb.Member)
	for _, m := range list {
		if m == nil {
			continue
		}
		memberMap[m.Id] = pb.Member{
			Id:        m.Id,
			FirstName: m.FirstName,
			LastName:  m.LastName,
			StripeId:  m.StripeId,
		}
	}

	return memberMap
}

func TestList(t *testing.T) {
	for _, tt := range []struct {
		name    string
		entries []dbEntry
		want    []*pb.Member
	}{
		{
			name: "Active member",
			entries: []dbEntry{
				{
					contactId: 1,
					firstName: "firstName",
					lastName:  "lastName",
					statusId:  2,
					email:     "firstname@lastname.com",
					stripeId:  "cust_abc",
				},
			},
			want: []*pb.Member{
				{
					Id:        1,
					FirstName: "firstName",
					LastName:  "lastName",
					StripeId:  "cust_abc",
				},
			},
		},
		{
			name: "Inactive member",
			entries: []dbEntry{
				{
					contactId: 1,
					firstName: "firstName",
					lastName:  "lastName",
					statusId:  4,
					email:     "firstname@lastname.com",
					stripeId:  "cust_abc",
				},
			},
			want: []*pb.Member{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dsn := initDb(t, tt.entries)
			err := Init(driver, dsn)
			if err != nil {
				t.Fatal(err)
			}
			list, err := List(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			if !cmpMemberLists(tt.want, list.GetMembers()) {
				t.Error("lists differ")
			}
		})
	}
}
