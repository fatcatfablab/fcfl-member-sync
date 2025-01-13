package types

import mapset "github.com/deckarep/golang-set/v2"

type (
	MemberSet        = mapset.Set[ComparableMember]
	ComparableMember struct {
		Id        int32
		UAId      string
		FirstName string
		LastName  string
	}
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

func (cm ComparableMember) SetUAId(uaid string) ComparableMember {
	cm.UAId = uaid
	return cm
}
