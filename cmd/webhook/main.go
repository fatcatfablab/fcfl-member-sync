package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const (
	httpAddr     = "127.0.0.1:8081"
	maxBodyBytes = int64(65536)
	civicrmId    = "civicrm_id"

	customerCreatedEvent        = "customer.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"
)

type Event struct {
	Type string    `json:"type"`
	Data EventData `json:"data"`
}

type EventData struct {
	Raw json.RawMessage `json:"object"`
}

type Customer struct {
	Name     string            `json:"name"`
	Email    string            `json:"email"`
	Metadata map[string]string `json:"metadata"`
}

type Subscription struct {
	Metadata map[string]string `json:"metadata"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /stripe_events", webhookHandler)

	s := http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	log.Printf("Listening on %s", httpAddr)
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}

func webhookHandler(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, maxBodyBytes)
	j := json.NewDecoder(req.Body)

	var event Event
	if err := j.Decode(&event); err != nil {
		log.Printf("Error decoding event: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case customerCreatedEvent:
		var c Customer
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
		var s Subscription
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
