package updater

import (
	"fmt"
	"log"
	"strconv"

	"github.com/fatcatfablab/fcfl-member-sync/client/types"
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
}

func New(uaClient *ua.Client) *UAUpdater {
	return &UAUpdater{uaClient: uaClient}
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
		log.Printf("Adding member %v", m)
		id := fmt.Sprintf("%d", m.Id)
		_, err = u.uaClient.CreateUser(schema.UserRequest{
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			EmployeeNumber: &id,
		})
		if err != nil {
			log.Printf("Skipping due to failure: %v", err)
			reterror = err
		}
	}

	return reterror
}

func (u *UAUpdater) Disable(members memberMap) error {
	var err, reterror error
	for id, m := range members {
		log.Printf("Disabling member %v", m)
		err = u.uaClient.UpdateUser(id, schema.UserRequest{
			FirstName: m.FirstName,
			LastName:  m.LastName,
			Status:    &deactivated,
		})
		if err != nil {
			log.Printf("error disabling member: %s", err)
			reterror = err
		}
	}

	return reterror
}

func (u *UAUpdater) Update(members memberMap) error {
	var err, reterror error
	for id, m := range members {
		log.Printf("Updating member %v", m)
		employeeNumber := fmt.Sprintf("%d", m.Id)
		err = u.uaClient.UpdateUser(id, schema.UserRequest{
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			EmployeeNumber: &employeeNumber,
			Status:         &active,
		})
		if err != nil {
			log.Printf("error updating member: %s", err)
			reterror = err
		}
	}

	return reterror
}
