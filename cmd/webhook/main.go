package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

const (
	httpAddr     = "127.0.0.1:8081"
	maxBodyBytes = int64(65536)
	civicrmId    = "civicrm_id"

	customerCreatedEvent        = "customer.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"
)

type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(bytes []byte) error {
	var raw int64
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	t.Time = time.Unix(raw, 0)
	return nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /stripe_events", webhookHandler)

	s := http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	log.Printf("stripe.APIVersion: %s", stripe.APIVersion)
	log.Printf("Listening on %s", httpAddr)
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}

func webhookHandler(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, maxBodyBytes)
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := os.Getenv("STRIPE_ENDPOINT_SECRET")
	event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		log.Printf("Payload: %s", string(payload))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case customerCreatedEvent:
		var c stripe.Customer
		if err := json.Unmarshal(event.Data.Raw, &c); err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		id, ok := c.Metadata[civicrmId]
		if ok {
			log.Printf("New member: %s %s %s", c.Name, c.Email, id)
		} else {
			log.Printf("New member: %s %s", c.Name, c.Email)
		}

	case customerSubscriptionDeleted:
		var s stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &s); err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		id, ok := s.Metadata[civicrmId]
		if ok {
			log.Printf("Deleted sub: %s", id)
		} else {
			log.Print("Deleted sub: no data")
		}

	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}
