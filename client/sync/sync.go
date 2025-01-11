package sync

import (
	"log"
	"maps"
	"slices"

	"github.com/miquelruiz/fcfl-member-sync/client/types"
)

type MemberSet = types.MemberSet

type updater interface {
	Add(MemberSet) error
	Disable(MemberSet) error
	Update(MemberSet) error
}

func Reconcile(remote, local MemberSet, u updater) error {
	if local.Equal(remote) {
		log.Print("Nothing to do")
		return nil
	}

	localIds := types.ToIdMap(local)

	// We need to check for the id's in order to know if a member is missing,
	// or if it just needs updating.
	add := types.NewMemberSet()
	update := types.NewMemberSet()

	for m := range remote.Difference(local).Iter() {
		_, present := localIds[m.Id]
		if present {
			update.Add(m)
		} else {
			add.Add(m)
		}
		localIds[m.Id] = m
	}

	if !update.IsEmpty() {
		log.Printf("Members to update: %d", update.Cardinality())
		if err := u.Update(update); err != nil {
			log.Printf("error updating members: %s", err)
		}
	}

	if !add.IsEmpty() {
		log.Printf("Members to add: %d", add.Cardinality())
		if err := u.Add(add); err != nil {
			log.Printf("error adding members: %s", err)
		}
	}

	local = types.NewMemberSet(slices.Collect(maps.Values(localIds))...)
	extra := local.Difference(remote)
	if !extra.IsEmpty() {
		log.Printf("Members to disable: %d", extra.Cardinality())
		if err := u.Disable(extra); err != nil {
			log.Printf("error disabling members: %s", err)
		}
	}

	return nil
}
