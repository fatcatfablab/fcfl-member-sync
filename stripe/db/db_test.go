package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
)

const (
	dsn = "root@/"
)

func getDb(t *testing.T, dbName string) *DB {
	sqlDB, err := sql.Open(dbDriver, dsn)
	if err != nil {
		t.Fatalf("can't connect to db: %s", err)
	}
	defer sqlDB.Close()

	_, err = sqlDB.Exec("CREATE OR REPLACE DATABASE " + dbName)
	if err != nil {
		t.Fatalf("can't create database %s: %s", dbName, err)
	}

	db, err := New(dsn + dbName)
	if err != nil {
		t.Fatalf("can't connect to test db: %s", err)
	}

	t.Cleanup(func() {
		db.db.Exec("DROP DATABASE ?", dbName)
	})

	return db
}

func equal(m types.Member, c types.Customer) bool {
	return m.CustomerId == c.CustomerId && m.Name == c.Name && m.Email == c.Email
}

func TestCreateMember(t *testing.T) {
	db := getDb(t, fmt.Sprintf("test_create_member_%d", time.Now().Unix()))

	for _, tt := range []struct {
		name       string
		customer   types.Customer
		shouldFail bool
	}{
		{
			name:     "Create member and find it",
			customer: types.Customer{CustomerId: "abc", Name: "name", Email: "email"},
		},
		{
			name:       "Create member with same email fails",
			customer:   types.Customer{CustomerId: "qwer", Name: "name2", Email: "email"},
			shouldFail: true,
		},
		{
			name:     "Create same member updates",
			customer: types.Customer{CustomerId: "abc", Name: "name", Email: "email2"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.CreateMember(tt.customer); err != nil && !tt.shouldFail {
				t.Fatalf("error creating member: %s", err)
			}

			if tt.shouldFail {
				return
			}

			m, err := db.FindMemberByCustomerId(tt.customer.CustomerId)
			if err != nil {
				t.Fatalf("error finding member: %s", err)
			}

			if m.Status != types.MemberStatusNotActive {
				t.Errorf("new member expected to be not_active: %+v", *m)
			}

			if !equal(*m, tt.customer) {
				t.Errorf(
					"member doesn't match customer. Got: %+v Want: %+v",
					*m,
					tt.customer,
				)
			}
		})
	}
}

func TestActivateDeactivateMember(t *testing.T) {
	db := getDb(t, fmt.Sprintf("test_activate_member_%d", time.Now().Unix()))
	for _, c := range []types.Customer{
		{CustomerId: "abc", Name: "name1", Email: "email1"},
		{CustomerId: "xyz", Name: "name2", Email: "email2"},
	} {
		if err := db.CreateMember(c); err != nil {
			t.Fatalf("error creating fixture: %s", err)
		}
	}

	for _, tt := range []struct {
		name       string
		shouldFail bool
		customerId string
		status     string
	}{
		{
			name:       "Activate unexistant member fails",
			shouldFail: true,
			customerId: "xxx",
			status:     types.MemberStatusActive,
		},
		{
			name:       "Activate member",
			customerId: "abc",
			status:     types.MemberStatusActive,
		},
		{
			name:       "Activate member 2",
			customerId: "xyz",
			status:     types.MemberStatusActive,
		},
		{
			name:       "Deactivate unexistant member fails",
			shouldFail: true,
			customerId: "xxx",
			status:     types.MemberStatusNotActive,
		},
		{
			name:       "Deactivate member",
			customerId: "abc",
			status:     types.MemberStatusNotActive,
		},
		{
			name:       "Deactivate member 2",
			customerId: "xyz",
			status:     types.MemberStatusNotActive,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var call func(string) error
			if tt.status == types.MemberStatusActive {
				call = db.ActivateMember
			} else {
				call = db.DeactivateMember
			}
			err := call(tt.customerId)
			if err != nil {
				if !tt.shouldFail {
					t.Errorf("unexpected error: %s", err)
				}
			}

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error, but didn't")
				}
				return
			}

			m, err := db.FindMemberByCustomerId(tt.customerId)
			if err != nil {
				t.Errorf("error finding member: %s", err)
			}

			if m.Status != tt.status {
				t.Errorf("unexpected member status: %+v", *m)
			}
		})
	}
}

func TestUpdateMemberAccess(t *testing.T) {
	db := getDb(t, fmt.Sprintf("test_update_member_access_%d", time.Now().Unix()))

	for _, c := range []types.Customer{
		{CustomerId: "abc", Name: "name1", Email: "email1"},
		{CustomerId: "xyz", Name: "name2", Email: "email2"},
	} {
		if err := db.CreateMember(c); err != nil {
			t.Fatalf("error creating fixture: %s", err)
		}
	}

	for _, tt := range []struct {
		name       string
		shouldFail bool
		customerId string
		accessId   string
	}{
		{
			name:       "Update unexisting member fails",
			shouldFail: true,
			customerId: "xxx",
			accessId:   "abc",
		},
		{
			name:       "Invalid accessId fails",
			shouldFail: true,
			customerId: "abc",
			accessId:   "weqrqwer",
		},
		{
			name:       "Regular update",
			customerId: "abc",
			accessId:   "1c897f18-2cb4-4644-a900-8ddfc23d6f77",
		},
		{
			name:       "Update with the same accessId fails",
			customerId: "xyz",
			accessId:   "1c897f18-2cb4-4644-a900-8ddfc23d6f77",
			shouldFail: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateMemberAccess(tt.customerId, tt.accessId)
			if err != nil {
				if !tt.shouldFail {
					t.Errorf("unexpected error: %s", err)
				}
			}

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error, but didn't")
				}
				return
			}

			m, err := db.FindMemberByCustomerId(tt.customerId)
			if err != nil {
				t.Errorf("error finding customer: %s", err)
			}

			if m.AccessId == nil || *m.AccessId != tt.accessId {
				t.Errorf("unexpected access_id: %s", *m.AccessId)
			}
		})
	}
}
