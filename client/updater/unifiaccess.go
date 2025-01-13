package updater

import (
	"fmt"
	"log"
	"strconv"

	"github.com/miquelruiz/fcfl-member-sync/client/types"
	ua "github.com/miquelruiz/go-unifi-access-api"
	"github.com/miquelruiz/go-unifi-access-api/schema"
)

var (
	deactivated = "DEACTIVATED"
	active      = "ACTIVE"
)

type memberSet = types.MemberSet

type UAUpdater struct {
	uaClient *ua.Client
}

func New(uaClient *ua.Client) *UAUpdater {
	return &UAUpdater{uaClient: uaClient}
}

func (u *UAUpdater) List() (memberSet, error) {
	users, err := u.uaClient.ListUsers()
	if err != nil {
		return nil, err
	}

	members := types.NewMemberSet()
	for _, user := range users {
		id, err := strconv.ParseInt(user.EmployeeNumber, 0, 32)
		if err != nil {
			log.Printf("skipping local user without EmployeeNumber: %s", user.FullName)
			continue
		}
		members.Add(types.ComparableMember{
			FirstName: user.FirstName,
			LastName:  user.LastName,
			UAId:      user.Id,
			Id:        int32(id),
		})
	}

	return members, nil
}

func (u *UAUpdater) Add(members memberSet) error {
	for m := range members.Iter() {
		log.Printf("Adding member %v", m)
		id := fmt.Sprintf("%d", m.Id)
		_, err := u.uaClient.CreateUser(schema.UserRequest{
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			EmployeeNumber: &id,
		})
		if err != nil {
			log.Printf("Skipping due to failure: %v", err)
		}
	}

	return nil
}

func (u *UAUpdater) Disable(members memberSet) error {
	for m := range members.Iter() {
		log.Printf("Disabling member %v", m)
		u.uaClient.UpdateUser(m.UAId, schema.UserRequest{
			FirstName: m.FirstName,
			LastName:  m.LastName,
			Status:    &deactivated,
		})
	}

	return nil
}

func (u *UAUpdater) Update(members memberSet) error {
	for m := range members.Iter() {
		log.Printf("Updating member %v", m)
		employeeNumber := fmt.Sprintf("%d", m.Id)
		u.uaClient.UpdateUser(m.UAId, schema.UserRequest{
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			EmployeeNumber: &employeeNumber,
			Status:         &active,
		})
	}

	return nil
}
