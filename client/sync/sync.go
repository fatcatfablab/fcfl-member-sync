package sync

import (
	"log"
	"maps"
	"slices"

	"github.com/miquelruiz/fcfl-member-sync/client/types"
	"github.com/samber/lo"
)

type (
	MemberSet = types.MemberSet
	Member    = types.ComparableMember
)

type updater interface {
	Add(MemberSet) error
	Disable(MemberSet) error
	Update(MemberSet) error
}

func Reconcile(remote, local MemberSet, u updater) error {
	var err error

	// Extract the UniFi Access ID's from the local member set so it can be
	// compared with the remote member set, which doesn't have those ID's
	idMapping := make(map[int32]string)
	localCopy := types.NewMemberSet()
	for l := range local.Iter() {
		idMapping[l.Id] = l.UAId
		localCopy.Add(l.SetUAId(""))
	}
	local = localCopy

	if local.Equal(remote) {
		log.Print("Nothing to do")
		return nil
	}

	localIds := types.ToIdMap(local)
	add := types.NewMemberSet()
	update := types.NewMemberSet()

	for m := range remote.Difference(local).Iter() {
		// We need to check for the id's in order to know if a member is missing
		// or if it just needs updating.
		_, present := localIds[m.Id]
		if present {
			update.Add(m.SetUAId(idMapping[m.Id]))
		} else {
			add.Add(m)
		}
		localIds[m.Id] = m
	}

	if !update.IsEmpty() {
		log.Printf("Members to update: %d", update.Cardinality())
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
		log.Printf("Members to disable: %d", extra.Cardinality())
		newSet := types.NewMemberSet(lo.Map(extra.ToSlice(), func(item Member, _ int) Member {
			return item.SetUAId(idMapping[item.Id])
		})...)
		if err = u.Disable(newSet); err != nil {
			log.Printf("error disabling members: %s", err)
		}
	}

	return err
}
