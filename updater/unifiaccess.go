package updater

import (
	"fmt"
	"log"
	"strconv"

	"github.com/fatcatfablab/fcfl-member-sync/types"
	ua "github.com/miquelruiz/go-unifi-access-api"
	"github.com/miquelruiz/go-unifi-access-api/schema"
)

var (
	deactivated = "DEACTIVATED"
	active      = "ACTIVE"
)

type (
	memberSet = types.MemberSet
	member    = types.ComparableMember
	memberMap = map[string]member
)

type UAUpdater struct {
	uaClient *ua.Client
	dryRun   bool
}

func New(uaClient *ua.Client, dryRun bool) *UAUpdater {
	return &UAUpdater{uaClient: uaClient, dryRun: dryRun}
}

func (u *UAUpdater) List() (memberMap, error) {
	users, err := u.uaClient.ListUsers()
	if err != nil {
		return nil, err
	}

	members := make(map[string]member)
	for _, user := range users {
		id, err := strconv.ParseInt(user.EmployeeNumber, 0, 32)
		if err != nil {
			log.Printf("skipping user without EmployeeNumber: %s", user.FullName)
			continue
		}
		members[user.Id] = member{
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Id:        int32(id),
			Status:    user.Status,
		}
	}

	return members, nil
}

func (u *UAUpdater) Add(members memberSet) error {
	var err, reterror error
	for m := range members.Iter() {
		_, err = u.AddMember(m)
		if err != nil {
			log.Printf("Skipping due to failure: %v", err)
			reterror = err
		}
	}

	return reterror
}

func (u *UAUpdater) AddMember(m member) (string, error) {
	var err error
	var accessId string

	msg := "Adding member %v"
	if u.dryRun {
		msg = "[DRY-RUN] " + msg
	}
	log.Printf(msg, m)

	id := fmt.Sprintf("%d", m.Id)
	if !u.dryRun {
		r, err := u.uaClient.CreateUser(schema.UserRequest{
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			EmployeeNumber: &id,
		})
		accessId = r.Id
		if err != nil {
			return "", err
		}
	}

	return accessId, err
}

func (u *UAUpdater) Disable(members memberMap) error {
	var err, reterror error
	for id, m := range members {
		err = u.DisableMember(id, m)
		if err != nil {
			log.Printf("error disabling member: %s", err)
			reterror = err
		}
	}

	return reterror
}

func (u *UAUpdater) DisableMember(id string, m member) error {
	var err error

	msg := "Disabling member %v"
	if u.dryRun {
		msg = "[DRY-RUN] " + msg
	}
	log.Printf(msg, m)

	if !u.dryRun {
		err = u.uaClient.UpdateUser(id, schema.UserRequest{
			FirstName: m.FirstName,
			LastName:  m.LastName,
			Status:    &deactivated,
		})
	}

	return err
}

func (u *UAUpdater) Update(members memberMap) error {
	var err, reterror error
	for id, m := range members {
		if err = u.UpdateMember(id, m); err != nil {
			log.Printf("error updating member: %s", err)
			reterror = err
		}
	}

	return reterror
}

func (u *UAUpdater) UpdateMember(id string, m member) error {
	var err error

	msg := "Updating member %v"
	if u.dryRun {
		msg = "[DRY-RUN] " + msg
	}
	log.Printf(msg, m)

	employeeNumber := fmt.Sprintf("%d", m.Id)
	if !u.dryRun {
		err = u.uaClient.UpdateUser(id, schema.UserRequest{
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			EmployeeNumber: &employeeNumber,
			Status:         &active,
		})
	}

	return err
}
