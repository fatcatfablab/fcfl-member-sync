package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	httpAddr     = "127.0.0.1:8081"
	maxBodyBytes = int64(65536)
	civicrmId    = "civicrm_id"

	stripeSignatureHeader = "Stripe-Signature"

	customerCreatedEvent        = "customer.created"
	customerSubscriptionDeleted = "customer.subscription.deleted"
)

var stripeEndpointSecret = ""

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

func init() {
	flag.StringVar(&stripeEndpointSecret, "endpoint-secret", os.Getenv("STRIPE_ENDPOINT_SECRET"), "Stripe endpoint secret")
}

func main() {
	flag.Parse()
	if stripeEndpointSecret == "" {
		panic("No stripe endpoint secret given")
	}

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
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error copying request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	signature := req.Header.Get(stripeSignatureHeader)
	if !verifySignature(payload, signature, stripeEndpointSecret) {
		log.Printf("Error verifying signature")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
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
		log.Printf("Payload: %s", string(payload))
	}

	w.WriteHeader(http.StatusOK)
}

func verifySignature(payload []byte, signature string, secret string) bool {
	elemMap := make(map[string]string)
	elemSlice := strings.Split(signature, ",")
	for _, e := range elemSlice {
		kv := strings.SplitN(e, "=", 2)
		elemMap[kv[0]] = kv[1]
	}

	signedPayload := fmt.Appendf(nil, "%s.%s", elemMap["t"], payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write(signedPayload)
	if err != nil {
		log.Printf("error writing to hmac: %v", err)
		return false
	}
	expectedMAC := mac.Sum(nil)

	v1Header := []byte(elemMap["v1"])
	actualMAC := make([]byte, hex.DecodedLen(len(v1Header)))
	_, err = hex.Decode(actualMAC, v1Header)
	if err != nil {
		log.Printf("error decoding header: %v", err)
		return false
	}

	return hmac.Equal(expectedMAC, actualMAC)
}
