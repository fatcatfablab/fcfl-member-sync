package sync

import (
	"testing"

	"github.com/fatcatfablab/fcfl-member-sync/client/types"
)

var (
	m1 = Member{
		FirstName: "m1",
		Id:        1,
	}
	m2 = Member{
		FirstName: "m2",
		Id:        2,
	}
	m3 = Member{
		FirstName: "m3",
		Id:        3,
	}
	m4 = Member{
		FirstName: "m4",
		Id:        4,
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
		u.t.Errorf("member map to update does not match")
	}
	return nil
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
			remote:     types.NewMemberSet([]Member{m1, m2, {Id: 3, FirstName: "xx"}}...),
			local:      MemberMap{"uaid1": m1, "uaid2": m2, "uaid3": m3},
			wantUpdate: MemberMap{"uaid3": {Id: 3, FirstName: "xx"}},
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
			local:       MemberMap{"uaid1": m1, "uaid2": {Id: 2, FirstName: "xx"}, "uaid3": m3},
			wantAdd:     types.NewMemberSet([]Member{m4}...),
			wantDisable: MemberMap{"uaid3": m3},
			wantUpdate:  MemberMap{"uaid2": m2},
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
