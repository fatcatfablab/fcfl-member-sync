package sync

import (
	"testing"

	"github.com/miquelruiz/fcfl-member-sync/client/types"
)

var (
	m1 = types.ComparableMember{
		FirstName: "m1",
		Id:        1,
	}
	m2 = types.ComparableMember{
		FirstName: "m2",
		Id:        2,
	}
	m3 = types.ComparableMember{
		FirstName: "m3",
		Id:        3,
	}
	m4 = types.ComparableMember{
		FirstName: "m4",
		Id:        4,
	}
)

type mockUpdater struct {
	t       *testing.T
	add     MemberSet
	disable MemberSet
	update  MemberSet
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

func (u *mockUpdater) Disable(m MemberSet) error {
	if u.disable == nil && m != nil {
		u.t.Errorf("unexpected disable: %v", m)
	}
	if m != nil && u.disable != nil && !m.Equal(u.disable) {
		u.t.Errorf("member set to disable does not match")
	}
	return nil
}

func (u *mockUpdater) Update(m MemberSet) error {
	if u.update == nil && m != nil {
		u.t.Errorf("unexpected update: %v", m)
	}
	if m != nil && u.update != nil && !m.Equal(u.update) {
		u.t.Errorf("member set to update does not match")
	}
	return nil
}

func TestReconcile(t *testing.T) {
	for _, tt := range []struct {
		name        string
		remote      MemberSet
		local       MemberSet
		wantAdd     MemberSet
		wantDisable MemberSet
		wantUpdate  MemberSet
	}{
		{
			name:   "Nothing to do",
			remote: types.NewMemberSet([]types.ComparableMember{m1}...),
			local:  types.NewMemberSet([]types.ComparableMember{m1}...),
		},
		{
			name:    "Single member gets added",
			remote:  types.NewMemberSet([]types.ComparableMember{m1}...),
			local:   types.NewMemberSet(),
			wantAdd: types.NewMemberSet([]types.ComparableMember{m1}...),
		},
		{
			name:    "Multiple members get added",
			remote:  types.NewMemberSet([]types.ComparableMember{m1, m2, m3}...),
			local:   types.NewMemberSet([]types.ComparableMember{m1}...),
			wantAdd: types.NewMemberSet([]types.ComparableMember{m2, m3}...),
		},
		{
			name:       "Member gets updated",
			remote:     types.NewMemberSet([]types.ComparableMember{m1, m2, {Id: 3, FirstName: "xx"}}...),
			local:      types.NewMemberSet([]types.ComparableMember{m1, m2, m3}...),
			wantUpdate: types.NewMemberSet([]types.ComparableMember{{Id: 3, FirstName: "xx"}}...),
		},
		{
			name:        "Member gets disabled",
			remote:      types.NewMemberSet([]types.ComparableMember{m1, m2, m3}...),
			local:       types.NewMemberSet([]types.ComparableMember{m1, m2, m3, m4}...),
			wantDisable: types.NewMemberSet([]types.ComparableMember{m4}...),
		},
		{
			name:        "Multiple operations",
			remote:      types.NewMemberSet([]types.ComparableMember{m1, m2, m4}...),
			local:       types.NewMemberSet([]types.ComparableMember{m1, {Id: 2, FirstName: "xx"}, m3}...),
			wantAdd:     types.NewMemberSet([]types.ComparableMember{m4}...),
			wantDisable: types.NewMemberSet([]types.ComparableMember{m3}...),
			wantUpdate:  types.NewMemberSet([]types.ComparableMember{m2}...),
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
