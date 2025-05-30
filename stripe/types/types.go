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
	CustomerId string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

const (
	SubscriptionIncomplete        = "incomplete"
	SubscriptionIncompleteExpired = "incomplete_expired"
	SubscriptionTrialing          = "trialing"
	SubscriptionActive            = "active"
	SubscriptionPastDue           = "past_due"
	SubscriptionCanceled          = "canceled"
	SubscriptionUnpaid            = "unpaid"
	SubscriptionPaused            = "paused"
)

type Subscription struct {
	Status     string `json:"status"`
	Customer   string `json:"customer"`
	CancelAt   *int64 `json:"cancel_at"`
	CanceledAt *int64 `json:"canceled_at"`
}

const (
	MemberStatusActive    = "active"
	MemberStatusNotActive = "not_active"
)

type Member struct {
	MemberId   int64
	CustomerId string
	AccessId   *string
	Name       string
	Email      string
	Status     string
}
