package sync

import (
	"log"
	"maps"
	"slices"

	"github.com/fatcatfablab/fcfl-member-sync/client/types"
)

type (
	MemberMap = types.MemberMap
	MemberSet = types.MemberSet
	Member    = types.ComparableMember
)

type updater interface {
	Add(MemberSet) error
	Disable(MemberMap) error
	Update(MemberMap) error
}

func Reconcile(remote MemberSet, localMap MemberMap, u updater) error {
	var err error

	// This allows for quick extraction of the UniFi Access ID given the Member id
	idMapping := make(map[int32]string)
	for k, v := range localMap {
		idMapping[v.Id] = k
	}

	local := types.NewMemberSet(slices.Collect(maps.Values(localMap))...)
	if local.Equal(remote) {
		log.Print("Nothing to do")
		return nil
	}

	localIds := types.ToIdMap(local)
	add := types.NewMemberSet()
	update := make(MemberMap)

	for m := range remote.Difference(local).Iter() {
		log.Printf("diff: %v", m)
		// We need to check for the id's in order to know if a member is missing
		// or if it just needs updating.
		_, present := localIds[m.Id]
		if present {
			update[idMapping[m.Id]] = m
		} else {
			add.Add(m)
		}
		localIds[m.Id] = m
	}

	if len(update) > 0 {
		log.Printf("Members to update: %d", len(update))
		if err = u.Update(update); err != nil {
			log.Printf("error updating members: %s", err)
		}
	}

	if !add.IsEmpty() {
		log.Printf("Members to add: %d", add.Cardinality())
		if err = u.Add(add); err != nil {
			log.Printf("error adding members: %s", err)
		}
	}

	local = types.NewMemberSet(slices.Collect(maps.Values(localIds))...)
	extra := local.Difference(remote)
	if !extra.IsEmpty() {
		disable := make(MemberMap)
		for e := range extra.Iter() {
			if e.Status == types.StatusActive {
				disable[idMapping[e.Id]] = e
			}
		}
		if len(disable) > 0 {
			log.Printf("Members to disable: %d", extra.Cardinality())
			if err = u.Disable(disable); err != nil {
				log.Printf("error disabling members: %s", err)
			}
		}
	}

	return err
}
