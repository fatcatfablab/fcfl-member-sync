package types

import "encoding/json"

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
