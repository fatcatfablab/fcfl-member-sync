package types

import mapset "github.com/deckarep/golang-set/v2"

type (
	MemberSet        = mapset.Set[ComparableMember]
	ComparableMember struct {
		Id        int32
		FirstName string
		LastName  string
	}
	MemberMap = map[string]ComparableMember
)

func ToIdMap(ms MemberSet) map[int32]ComparableMember {
	idmap := make(map[int32]ComparableMember)
	for m := range ms.Iter() {
		idmap[m.Id] = m
	}
	return idmap
}

func NewMemberSet(vals ...ComparableMember) MemberSet {
	return mapset.NewSet(vals...)
}

func Equal(a, b MemberMap) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if v != b[k] {
			return false
		}
	}

	return true
}
