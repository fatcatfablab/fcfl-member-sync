package listener

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/db"
	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
)

const (
	customerUpdatedEvent        = "customer.updated"
	customerCreatedEvent        = "customer.created"
	customerSubscriptionCreated = "customer.subscription.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"

	maxBodyBytes          = int64(65536)
	stripeSignatureHeader = "Stripe-Signature"
)

type Listener struct {
	secret     string
	listenAddr string
	endpoint   string
	d          *db.DB
}

func New(secret, listeAddr, endpoint string, d *db.DB) *Listener {
	return &Listener{
		secret:     secret,
		listenAddr: listeAddr,
		endpoint:   endpoint,
		d:          d,
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
	log.Printf("Request received")
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

	log.Printf("%s event: %+v", eventType, c)
	return l.d.CreateCustomer(c)
}

func (l *Listener) handleSubscriptionCreated(rawEvent json.RawMessage) error {
	var s types.Subscription
	if err := json.Unmarshal(rawEvent, &s); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	log.Printf("%s event: %+v", customerSubscriptionCreated, s)
	_, err := l.d.CreateMember(s.Customer)

	// TODO:
	// 1- pull customer from db
	// 2- pull from stripe if it doesn't exist -> create customer in db
	// 3- create Access user
	// 4- update member with access id
	return err
}

func (l *Listener) handleSubscriptionDeleted(rawEvent json.RawMessage) error {
	var s types.Subscription
	if err := json.Unmarshal(rawEvent, &s); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	log.Printf("%s event: %+v", customerSubscriptionDeleted, s)
	// TODO deactivate in Access
	return l.d.RemoveMember(s.Customer)
}
