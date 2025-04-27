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
	civicrmId = "civicrm_id"

	customerCreatedEvent        = "customer.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"

	maxBodyBytes          = int64(65536)
	stripeSignatureHeader = "Stripe-Signature"
)

type Listener struct {
	secret     string
	listenAddr string
	endpoint   string
	db         *db.DB
}

func New(secret, listeAddr, endpoint string, d *db.DB) *Listener {
	return &Listener{
		secret:     secret,
		listenAddr: listeAddr,
		endpoint:   endpoint,
		db:         d,
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
	if !verifySignature(payload, signature, l.secret) {
		log.Printf("Error verifying signature")
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
	case customerCreatedEvent:
		if err := l.handleCustomerCreated(w, event.Data.Raw); err != nil {
			log.Printf("Error decoding sub-event: %v", err)
			return
		}

	case customerSubscriptionDeleted:
		if err := l.handleSubscriptionDeleted(w, event.Data.Raw); err != nil {
			return
		}

	default:
		log.Printf("Unhandled event type: %s", event.Type)
		log.Printf("Payload: %s", string(payload))
	}

	w.WriteHeader(http.StatusOK)
}

func (l *Listener) handleCustomerCreated(w http.ResponseWriter, rawEvent json.RawMessage) error {
	c, err := parseRawEvent[types.Customer](w, rawEvent)
	if err != nil {
		return err
	}

	id, ok := c.Metadata[civicrmId]
	if ok {
		log.Printf("New member: %s %s %s", c.Name, c.Email, id)
	} else {
		log.Printf("New member: %s %s", c.Name, c.Email)
	}

	// TODO create the user in Access

	return nil
}

func (l *Listener) handleSubscriptionDeleted(w http.ResponseWriter, rawEvent json.RawMessage) error {
	s, err := parseRawEvent[types.Subscription](w, rawEvent)
	if err != nil {
		return err
	}

	id, ok := s.Metadata[civicrmId]
	if ok {
		log.Printf("Deleted sub: %s", id)
	} else {
		log.Print("Deleted sub: no data")
	}

	// TODO store the event in db

	return nil
}

func parseRawEvent[T any](w http.ResponseWriter, e json.RawMessage) (*T, error) {
	var r T
	if err := json.Unmarshal(e, &r); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &r, nil
}
