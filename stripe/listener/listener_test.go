package listener

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
	"go.uber.org/mock/gomock"
)

func TestHandleCustomerEvent(t *testing.T) {
	for _, tt := range []struct {
		name       string
		input      json.RawMessage
		shouldFail bool
		mockSetup  func(*MockmemberDb)
	}{
		{
			name:       "Empty input",
			input:      []byte(""),
			shouldFail: true,
		},
		{
			name:       "Empty json object",
			input:      []byte("{}"),
			shouldFail: true,
		},
		{
			name:  "Regular event",
			input: []byte(`{"id":"abc","name":"name","email":"email"}`),
			mockSetup: func(mdb *MockmemberDb) {
				mdb.EXPECT().CreateMember(gomock.Eq(types.Customer{
					CustomerId: "abc",
					Name:       "name",
					Email:      "email",
				})).Times(1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb)
			}

			l := New("", "", "", mdb)
			err := l.handleCustomerEvent(tt.input, customerCreatedEvent)
			failed := err != nil

			if tt.shouldFail != failed {
				if tt.shouldFail {
					t.Error("test should've failed")
				} else {
					t.Errorf("unexpected failure: %s", err)
				}
			}
		})
	}
}

func TestHandleSubscriptionCreated(t *testing.T) {
	for _, tt := range []struct {
		name       string
		input      []byte
		shouldFail bool
		mockSetup  func(mdb *MockmemberDb)
	}{
		{
			name:       "Empty input",
			input:      []byte(""),
			shouldFail: true,
		},
		{
			name:       "Empty json object",
			input:      []byte("{}"),
			shouldFail: true,
		},
		{
			name:       "Unexistent member",
			input:      []byte(`{"status":"active","customer":"abc"}`),
			shouldFail: true,
			mockSetup: func(mdb *MockmemberDb) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{}, sql.ErrNoRows).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Times(0)
			},
		},
		{
			name:       "Failed activation",
			input:      []byte(`{"status":"active","customer":"abc"}`),
			shouldFail: true,
			mockSetup: func(mdb *MockmemberDb) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc"}, nil).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Return(errors.New("")).
					Times(1)
			},
		},
		{
			name:  "Regular subscription",
			input: []byte(`{"status":"active","customer":"abc"}`),
			mockSetup: func(mdb *MockmemberDb) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc"}, nil).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Times(1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb)
			}

			l := New("", "", "", mdb)
			err := l.handleSubscriptionCreated(tt.input)
			failed := err != nil

			if tt.shouldFail != failed {
				if tt.shouldFail {
					t.Error("test should've failed")
				} else {
					t.Errorf("unexpected failure: %s", err)
				}
			}
		})
	}
}

func TestHandleSubscriptionDeleted(t *testing.T) {
	for _, tt := range []struct {
		name       string
		input      json.RawMessage
		shouldFail bool
		mockSetup  func(*MockmemberDb)
	}{
		{
			name:       "Empty input",
			input:      []byte(""),
			shouldFail: true,
		},
		{
			name:       "Empty json object",
			input:      []byte("{}"),
			shouldFail: true,
		},
		{
			name:       "Failed deactivation",
			input:      []byte(`{"status":"canceled","customer":"abc"}`),
			shouldFail: true,
			mockSetup: func(mdb *MockmemberDb) {
				mdb.EXPECT().
					DeactivateMember(gomock.Eq("abc")).
					Return(errors.New("")).
					Times(1)
			},
		},
		{
			name:  "Regular event",
			input: []byte(`{"status":"canceled","customer":"abc"}`),
			mockSetup: func(mdb *MockmemberDb) {
				mdb.EXPECT().DeactivateMember(gomock.Eq("abc")).Times(1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb)
			}

			l := New("", "", "", mdb)
			err := l.handleSubscriptionDeleted(tt.input)
			failed := err != nil

			if tt.shouldFail != failed {
				if tt.shouldFail {
					t.Error("test should've failed")
				} else {
					t.Errorf("unexpected failure: %s", err)
				}
			}
		})
	}
}
