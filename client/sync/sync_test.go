package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/fatcatfablab/fcfl-member-sync/client/types"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	m1 = Member{
		FirstName: "m1",
		Id:        1,
		Status:    types.StatusActive,
	}
	m2 = Member{
		FirstName: "m2",
		Id:        2,
		Status:    types.StatusActive,
	}
	m3 = Member{
		FirstName: "m3",
		Id:        3,
		Status:    types.StatusActive,
	}
	m4 = Member{
		FirstName: "m4",
		Id:        4,
		Status:    types.StatusActive,
	}

	d1 = Member{
		FirstName: "m1",
		Id:        1,
		Status:    types.StatusDeactivated,
	}
	d2 = Member{
		FirstName: "m2",
		Id:        2,
		Status:    types.StatusDeactivated,
	}

	x1 = Member{
		FirstName: "x1",
		Id:        101,
		Status:    types.StatusDeactivated,
	}
)

type mockUpdater struct {
	t       *testing.T
	add     MemberSet
	disable MemberMap
	update  MemberMap
}

func (u *mockUpdater) Add(m MemberSet) error {
	if u.add == nil && m != nil {
		u.t.Errorf("unexpected add: %v", m)
	}
	if m != nil && u.add != nil && !m.Equal(u.add) {
		u.t.Errorf("member set to add does not match")
	}
	return nil
}

func (u *mockUpdater) Disable(m MemberMap) error {
	if u.disable == nil && m != nil {
		u.t.Errorf("unexpected disable: %v", m)
	}
	if m != nil && u.disable != nil && !types.Equal(m, u.disable) {
		u.t.Errorf("member map to disable does not match")
	}
	return nil
}

func (u *mockUpdater) Update(m MemberMap) error {
	if u.update == nil && m != nil {
		u.t.Errorf("unexpected update: %v", m)
	}
	if m != nil && u.update != nil && !types.Equal(m, u.update) {
		log.Printf("want: %+v", u.update)
		log.Printf("got:  %+v", m)
		u.t.Errorf("member map to update does not match")
	}
	return nil
}

type SQLiteUpdater struct {
	db    *sql.DB
	UUIDs map[int32]string
}

func NewSQLiteUpdater(path string) (*SQLiteUpdater, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	return &SQLiteUpdater{db: db, UUIDs: make(map[int32]string)}, nil
}

func (s *SQLiteUpdater) Init(m MemberMap) error {
	_, err := s.db.Exec(`CREATE TABLE members (
		id TEXT PRIMARY KEY,
		first_name TEXT NOT NULL,
		last_name TEXT,
		employee_id INTEGER,
		status TEXT,
		UNIQUE (employee_id)
	) STRICT`)
	if err != nil {
		return fmt.Errorf("couldn't initialize db: %w", err)
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO members (id, first_name, last_name, employee_id, status) ` +
			`VALUES (?, ?, ?, ?, "ACTIVE")`,
	)
	if err != nil {
		return fmt.Errorf("error preparing member insert: %w", err)
	}

	for id, member := range m {
		stmt.Exec(id, member.FirstName, member.LastName, member.Id)
	}

	return tx.Commit()
}

func (s *SQLiteUpdater) List() (MemberMap, error) {
	memberMap := make(map[string]Member)
	r, err := s.db.Query("SELECT id, first_name, last_name, employee_id, status FROM members")
	if err != nil {
		return nil, fmt.Errorf("error querying table members: %w", err)
	}

	for r.Next() {
		var id string
		m := Member{}
		err := r.Scan(&id, &m.FirstName, &m.LastName, &m.Id, &m.Status)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		memberMap[id] = m
	}

	return memberMap, nil
}

func (s *SQLiteUpdater) Add(m MemberSet) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO members (id, first_name, last_name, employee_id, status) ` +
			`VALUES (?, ?, ?, ?, "ACTIVE")`,
	)
	if err != nil {
		return fmt.Errorf("error preparing member insert: %w", err)
	}

	for member := range m.Iter() {
		u := uuid.New().String()
		stmt.Exec(u, member.FirstName, member.LastName, member.Id)
		s.UUIDs[member.Id] = u
	}

	return tx.Commit()
}

func (s *SQLiteUpdater) update(m MemberMap, status string) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}

	stmt, err := tx.Prepare(
		`UPDATE members SET first_name=?, last_name=?, employee_id=?, status=? ` +
			`WHERE id=?`,
	)
	if err != nil {
		return fmt.Errorf("error preparing member update: %w", err)
	}

	for id, member := range m {
		stmt.Exec(member.FirstName, member.LastName, strconv.Itoa(int(member.Id)), status, id)
	}

	return tx.Commit()
}

func (s *SQLiteUpdater) Update(m MemberMap) error {
	return s.update(m, "ACTIVE")
}

func (s *SQLiteUpdater) Disable(m MemberMap) error {
	return s.update(m, "DEACTIVATED")
}

func TestReconcile(t *testing.T) {
	for _, tt := range []struct {
		name        string
		remote      MemberSet
		local       MemberMap
		wantAdd     MemberSet
		wantDisable MemberMap
		wantUpdate  MemberMap
	}{
		{
			name:   "Nothing to do",
			remote: types.NewMemberSet([]Member{m1}...),
			local:  MemberMap{"uaid1": m1},
		},
		{
			name:    "Single member gets added",
			remote:  types.NewMemberSet([]Member{m1}...),
			local:   MemberMap{},
			wantAdd: types.NewMemberSet([]Member{m1}...),
		},
		{
			name:    "Multiple members get added",
			remote:  types.NewMemberSet([]Member{m1, m2, m3}...),
			local:   MemberMap{"uaid1": m1},
			wantAdd: types.NewMemberSet([]Member{m2, m3}...),
		},
		{
			name:       "Member gets updated",
			remote:     types.NewMemberSet([]Member{m1, m2, {Id: 3, FirstName: "xx", Status: types.StatusActive}}...),
			local:      MemberMap{"uaid1": m1, "uaid2": m2, "uaid3": m3},
			wantUpdate: MemberMap{"uaid3": {Id: 3, FirstName: "xx", Status: types.StatusActive}},
		},
		{
			name:        "Member gets disabled",
			remote:      types.NewMemberSet([]Member{m1, m2, m3}...),
			local:       MemberMap{"uaid1": m1, "uaid2": m2, "uaid3": m3, "uaid4": m4},
			wantDisable: MemberMap{"uaid4": m4},
		},
		{
			name:        "Multiple operations",
			remote:      types.NewMemberSet([]Member{m1, m2, m4}...),
			local:       MemberMap{"uaid1": m1, "uaid2": {Id: 2, FirstName: "xx", Status: types.StatusActive}, "uaid3": m3},
			wantAdd:     types.NewMemberSet([]Member{m4}...),
			wantDisable: MemberMap{"uaid3": m3},
			wantUpdate:  MemberMap{"uaid2": m2},
		},
		{
			name:   "Nothing to do with deactivated member",
			remote: types.NewMemberSet([]Member{m1, m2}...),
			local:  MemberMap{"uaid1": m1, "uaid2": m2, "uaid101": x1},
		},
		{
			name:       "Disabled member gets updated",
			remote:     types.NewMemberSet([]Member{m1}...),
			local:      MemberMap{"uaid1": d1},
			wantUpdate: MemberMap{"uaid1": m1},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := Reconcile(tt.remote, tt.local, &mockUpdater{
				t:       t,
				add:     tt.wantAdd,
				disable: tt.wantDisable,
				update:  tt.wantUpdate,
			})
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestReconcileSQLite(t *testing.T) {
	u, err := NewSQLiteUpdater(t.TempDir() + "disable-enable-test.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Init(MemberMap{}); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name   string
		remote MemberSet
		want   []Member
	}{
		{
			name:   "Nothing to do",
			remote: types.NewMemberSet(),
			want:   []Member{},
		},
		{
			name:   "Add a member",
			remote: types.NewMemberSet([]Member{m1}...),
			want:   []Member{m1},
		},
		{
			name:   "Add two more members",
			remote: types.NewMemberSet([]Member{m1, m2, m3}...),
			want:   []Member{m1, m2, m3},
		},
		{
			name:   "Member gets updated",
			remote: types.NewMemberSet([]Member{m1, m2, {Id: 3, FirstName: "xx", Status: types.StatusActive}}...),
			want:   []Member{m1, m2, {Id: 3, FirstName: "xx", Status: types.StatusActive}},
		},
		{
			name:   "Member gets disabled",
			remote: types.NewMemberSet([]Member{m1, m2}...),
			want:   []Member{m1, m2, {Id: 3, FirstName: "xx", Status: types.StatusDeactivated}},
		},
		{
			name:   "Nothing to do with member deactivated",
			remote: types.NewMemberSet([]Member{m1, m2}...),
			want:   []Member{m1, m2, {Id: 3, FirstName: "xx", Status: types.StatusDeactivated}},
		},
		{
			name:   "Multiple operations",
			remote: types.NewMemberSet([]Member{{Id: 1, FirstName: "xx", Status: types.StatusActive}, m3, m4}...),
			want:   []Member{{Id: 1, FirstName: "xx", Status: types.StatusActive}, d2, m3, m4},
		},
		{
			name:   "Everyone active",
			remote: types.NewMemberSet([]Member{m1, m2, m3, m4}...),
			want:   []Member{m1, m2, m3, m4},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			local, err := u.List()
			if err != nil {
				t.Fatal(err)
			}

			if err = Reconcile(tt.remote, local, u); err != nil {
				t.Fatal(err)
			}

			got, err := u.List()
			if err != nil {
				t.Fatal(err)
			}

			want := make(MemberMap)
			for _, w := range tt.want {
				want[u.UUIDs[w.Id]] = w
			}
			if !types.Equal(want, got) {
				log.Printf("want: %+v", want)
				log.Printf("got:  %+v", got)
				t.Fatal("want and got differ")
			}
		})
	}
}

func TestDisableThenEnable(t *testing.T) {
	local := MemberMap{"uaid1": m1}
	u, err := NewSQLiteUpdater(t.TempDir() + "disable-enable-test.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Init(local); err != nil {
		t.Fatal(err)
	}

	t.Run("Disable member", func(t *testing.T) {
		remote := types.NewMemberSet()
		err = Reconcile(remote, local, u)
		if err != nil {
			t.Fatal(err)
		}

		m1Copy := m1
		m1Copy.Status = "DEACTIVATED"
		want := map[string]Member{"uaid1": m1Copy}
		got, err := u.List()
		if err != nil {
			t.Fatal(err)
		}

		if !types.Equal(want, got) {
			log.Printf("want: %+v", want)
			log.Printf("got:  %+v", got)
			t.Fatal("Disable member didn't work")
		}
	})

	t.Run("Re-enable member", func(t *testing.T) {
		local, err = u.List()
		if err != nil {
			t.Fatal(err)
		}
		remote := types.NewMemberSet([]Member{m1}...)
		if err := Reconcile(remote, local, u); err != nil {
			t.Fatal(err)
		}

		m1Copy := m1
		m1Copy.Status = "ACTIVE"
		want := map[string]Member{"uaid1": m1Copy}
		got, err := u.List()
		if err != nil {
			t.Fatal(err)
		}
		if !types.Equal(want, got) {
			log.Printf("want: %+v", want)
			log.Printf("got:  %+v", got)
			t.Fatal("Re-enabling member didn't work")
		}
	})
}
