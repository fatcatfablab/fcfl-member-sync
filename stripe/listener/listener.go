package listener

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
)

const (
	customerCreatedEvent        = "customer.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"

	maxBodyBytes          = int64(65536)
	stripeSignatureHeader = "Stripe-Signature"
)

type DeletionSaver interface {
	Save(string, time.Time) error
}

type Listener struct {
	secret     string
	listenAddr string
	endpoint   string
	ds         DeletionSaver
}

func New(secret, listeAddr, endpoint string, d DeletionSaver) *Listener {
	return &Listener{
		secret:     secret,
		listenAddr: listeAddr,
		endpoint:   endpoint,
		ds:         d,
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
	case customerCreatedEvent:
		err = handleCustomerCreated(event.Data.Raw)

	case customerSubscriptionDeleted:
		err = handleSubscriptionDeleted(event.Data.Raw, l.ds)

	default:
		log.Printf("Unhandled event type: %s", event.Type)
		// log.Printf("Payload: %s", string(payload))
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleCustomerCreated(rawEvent json.RawMessage) error {
	var c types.Customer
	if err := json.Unmarshal(rawEvent, &c); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	// TODO create the user in Access
	log.Printf("Would create user: %v", c)

	return nil
}

func handleSubscriptionDeleted(rawEvent json.RawMessage, ds DeletionSaver) error {
	var s types.Subscription
	if err := json.Unmarshal(rawEvent, &s); err != nil {
		return fmt.Errorf("error unmarshalling json: %w", err)
	}

	log.Printf("Marking user for deletion: %v", s)

	var t time.Time
	if s.CancelAt != nil {
		t = time.Unix(*s.CancelAt, 0)
	} else {
		t = time.Now()
	}

	return ds.Save(s.Customer, t)
}
