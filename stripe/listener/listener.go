//go:generate mockgen --destination mock_listener_test.go --package listener . memberDb,uaUpdater

package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
	uaTypes "github.com/fatcatfablab/fcfl-member-sync/types"
)

const (
	customerUpdatedEvent        = "customer.updated"
	customerCreatedEvent        = "customer.created"
	customerSubscriptionCreated = "customer.subscription.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"

	maxBodyBytes          = int64(65536)
	stripeSignatureHeader = "Stripe-Signature"
)

type memberDb interface {
	CreateMember(c types.Customer) error
	ActivateMember(customerId string) error
	UpdateMemberAccess(customerId string, accessId string) error
	DeactivateMember(customerId string) error
	FindMemberByCustomerId(customerId string) (*types.Member, error)
}

type uaUpdater interface {
	AddMember(uaTypes.ComparableMember) (string, error)
	DisableMember(string, uaTypes.ComparableMember) error
}

type Listener struct {
	secret     string
	listenAddr string
	endpoint   string
	db         memberDb
	ua         uaUpdater
}

func New(secret, listenAddr, endpoint string, d memberDb, u uaUpdater) *Listener {
	return &Listener{
		secret:     secret,
		listenAddr: listenAddr,
		endpoint:   endpoint,
		db:         d,
		ua:         u,
	}
}

// Start does not return until the listener exits
func (l *Listener) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("POST %s", l.endpoint), l.webhookHandler)

	s := http.Server{
		Addr:    l.listenAddr,
		Handler: mux,
	}

	log.Printf("Listening on %s", l.listenAddr)
	return s.ListenAndServe()
}

func (l *Listener) webhookHandler(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, maxBodyBytes)
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error copying request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	signature := req.Header.Get(stripeSignatureHeader)
	if err := verifySignature(payload, signature, l.secret); err != nil {
		log.Printf("Error verifying signature: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var event types.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Error decoding event: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case customerCreatedEvent, customerUpdatedEvent:
		err = l.handleCustomerEvent(event.Data.Raw, event.Type)

	case customerSubscriptionCreated:
		err = l.handleSubscriptionCreated(event.Data.Raw)

	case customerSubscriptionDeleted:
		err = l.handleSubscriptionDeleted(event.Data.Raw)

	default:
		log.Printf("Unhandled event type: %s", event.Type)
		// log.Printf("Payload: %s", string(payload))
	}

	if err != nil {
		log.Printf("error handling request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (l *Listener) handleCustomerEvent(rawEvent json.RawMessage, eventType string) error {
	var c types.Customer
	if err := json.Unmarshal(rawEvent, &c); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	if c.CustomerId == "" && c.Email == "" && c.Name == "" {
		return fmt.Errorf("no relevant data in event: %s", string(rawEvent))
	}

	log.Printf("%s event: %+v", eventType, c)
	return l.db.CreateMember(c)
}

func (l *Listener) handleSubscriptionCreated(rawEvent json.RawMessage) error {
	var s types.Subscription
	if err := json.Unmarshal(rawEvent, &s); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	if s.Customer == "" {
		return errors.New("no customer id in subscription event")
	}

	log.Printf("%s event: %+v", customerSubscriptionCreated, s)
	m, err := l.db.FindMemberByCustomerId(s.Customer)
	if err != nil {
		// TODO: pull from stripe if it doesn't exist
		return fmt.Errorf("error querying member %q: %w", s.Customer, err)
	}

	log.Printf("activating member %q", m.Name)
	if err := l.db.ActivateMember(s.Customer); err != nil {
		return fmt.Errorf("error activating member %q: %w", s.Customer, err)
	}
	log.Printf("member id for %q: %d", m.Name, m.MemberId)

	accessId, err := l.ua.AddMember(memberToComparableMember(*m))
	if err != nil {
		return fmt.Errorf("failed to add member %+v to UA: %w", m, err)
	}
	log.Printf("access id for %q: %s", m.Name, accessId)

	return l.db.UpdateMemberAccess(s.Customer, accessId)
}

func (l *Listener) handleSubscriptionDeleted(rawEvent json.RawMessage) error {
	var s types.Subscription
	if err := json.Unmarshal(rawEvent, &s); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	if s.Customer == "" {
		return errors.New("no customer found in event")
	}

	log.Printf("%s event: %+v", customerSubscriptionDeleted, s)
	m, err := l.db.FindMemberByCustomerId(s.Customer)
	if err != nil {
		return fmt.Errorf("error finding membmer %q: %w", s.Customer, err)
	}

	if m.AccessId != nil {
		err = l.ua.DisableMember(*m.AccessId, memberToComparableMember(*m))
		if err != nil {
			return fmt.Errorf(
				"error disabing member %q in UA: %w",
				s.Customer,
				err,
			)
		}
	} else {
		log.Printf("member didn't have an access_id: %s", s.Customer)
	}

	return l.db.DeactivateMember(s.Customer)
}

func memberToComparableMember(m types.Member) uaTypes.ComparableMember {
	firstName, lastNAme, _ := strings.Cut(m.Name, " ")
	return uaTypes.ComparableMember{
		Id:        int32(m.MemberId),
		FirstName: firstName,
		LastName:  lastNAme,
	}
}
