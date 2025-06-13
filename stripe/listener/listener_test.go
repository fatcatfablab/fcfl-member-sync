package listener

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/types"
	"go.uber.org/mock/gomock"
)

const (
	secret = "somesecret"
)

func TestHandleCustomerEvent(t *testing.T) {
	for _, tt := range []struct {
		name       string
		input      json.RawMessage
		shouldFail bool
		mockSetup  func(*MockmemberDb, *MockuaUpdater)
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
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().CreateMember(gomock.Eq(types.Customer{
					CustomerId: "abc",
					Name:       "name",
					Email:      "email",
				})).Times(1)

				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc"}, nil).
					Times(1)

				ua.EXPECT().UpdateMember(gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name:  "Update existing member",
			input: []byte(`{"id":"abc","name":"name","email":"email"}`),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				accessId := "access-id"
				mdb.EXPECT().CreateMember(gomock.Eq(types.Customer{
					CustomerId: "abc",
					Name:       "name",
					Email:      "email",
				})).Times(1)

				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc", AccessId: &accessId}, nil).
					Times(1)

				ua.EXPECT().UpdateMember(gomock.Eq(accessId), gomock.Any()).Times(1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)
			ua := NewMockuaUpdater(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb, ua)
			}

			l := New("", "", "", mdb, ua)
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
		mockSetup  func(mdb *MockmemberDb, ua *MockuaUpdater)
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
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{}, sql.ErrNoRows).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Times(0)

				ua.EXPECT().
					AddMember(gomock.Any()).
					Times(0)
			},
		},
		{
			name:       "Failed activation",
			input:      []byte(`{"status":"active","customer":"abc"}`),
			shouldFail: true,
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc"}, nil).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Return(errors.New("")).
					Times(1)

				ua.EXPECT().
					AddMember(gomock.Any()).
					Times(0)
			},
		},
		{
			name:  "Regular subscription",
			input: []byte(`{"status":"active","customer":"abc"}`),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc"}, nil).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Times(1)

				ua.EXPECT().
					AddMember(gomock.Any()).
					Return("access-id", nil).
					Times(1)

				mdb.EXPECT().
					UpdateMemberAccess(gomock.Eq("abc"), gomock.Eq("access-id")).
					Times(1)
			},
		},
		{
			name:  "Duplicated subscription",
			input: []byte(`{"status":"active","customer":"abc"}`),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				accessId := "abcdef"
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc", Status: types.MemberStatusActive, AccessId: &accessId}, nil).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Any()).
					Times(0)

				ua.EXPECT().
					AddMember(gomock.Any()).
					Times(0)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)
			ua := NewMockuaUpdater(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb, ua)
			}

			l := New("", "", "", mdb, ua)
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
		mockSetup  func(mdb *MockmemberDb, ua *MockuaUpdater)
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
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc"}, nil).
					Times(1)

				ua.EXPECT().
					DisableMember(gomock.Any(), gomock.Any()).
					Times(0)

				mdb.EXPECT().
					DeactivateMember(gomock.Eq("abc")).
					Return(errors.New("")).
					Times(1)
			},
		},
		{
			name:  "Regular event",
			input: []byte(`{"status":"canceled","customer":"abc"}`),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				accessId := "zxcv"
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc", AccessId: &accessId}, nil).
					Times(1)

				ua.EXPECT().
					DisableMember(gomock.Eq(accessId), gomock.Any()).
					Times(1)

				mdb.EXPECT().DeactivateMember(gomock.Eq("abc")).Times(1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)
			ua := NewMockuaUpdater(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb, ua)
			}

			l := New("", "", "", mdb, ua)
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

func buildStripeRequest(t *testing.T, payload string) *http.Request {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	signature, err := sign([]byte(payload), ts, secret)
	if err != nil {
		t.Fatalf("Failed to sign test payload: %s", err)
	}

	hexSignature := make([]byte, hex.EncodedLen(len(signature)))
	hex.Encode(hexSignature, signature)
	header := fmt.Sprintf("t=%s,v1=%s", ts, string(hexSignature))

	return &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/"},
		Body:   io.NopCloser(strings.NewReader(payload)),
		Header: http.Header{
			stripeSignatureHeader: {header},
		},
	}
}

func buildInvalidSignatureRequest(t *testing.T) *http.Request {
	r := buildStripeRequest(t, "")
	r.Header[stripeSignatureHeader] = []string{"some-gibberish"}
	return r
}

func TestWebhookHandler(t *testing.T) {
	for _, tt := range []struct {
		name           string
		request        *http.Request
		mockSetup      func(mdb *MockmemberDb, ua *MockuaUpdater)
		wantStatusCode int
	}{
		{
			name:           "Signature verification error",
			request:        buildInvalidSignatureRequest(t),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Invalid event",
			request:        buildStripeRequest(t, ""),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "Customer event",
			request: buildStripeRequest(
				t,
				`{
					"type":"customer.created",
					"data":{
						"object":{
							"id":"abc",
							"name":"name",
							"email":"email"
						}
					}
				}`,
			),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().
					CreateMember(gomock.Eq(types.Customer{
						CustomerId: "abc",
						Name:       "name",
						Email:      "email",
					})).
					Times(1)

				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Times(1)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "Subscription created",
			request: buildStripeRequest(
				t,
				`{
					"type":"customer.subscription.created",
					"data":{
						"object":{
							"customer":"abc"
						}
					}
				}`,
			),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{}, nil).
					Times(1)

				mdb.EXPECT().
					ActivateMember(gomock.Eq("abc")).
					Times(1)

				ua.EXPECT().
					AddMember(gomock.Any()).
					Return("access-id", nil).
					Times(1)

				mdb.EXPECT().
					UpdateMemberAccess(gomock.Eq("abc"), gomock.Eq("access-id")).
					Times(1)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "Subscription deleted",
			request: buildStripeRequest(
				t,
				`{
					"type":"customer.subscription.deleted",
					"data":{
						"object":{
							"customer":"abc"
						}
					}
				}`,
			),
			mockSetup: func(mdb *MockmemberDb, ua *MockuaUpdater) {
				accessId := "zxcv"
				mdb.EXPECT().
					FindMemberByCustomerId(gomock.Eq("abc")).
					Return(&types.Member{MemberId: 123, CustomerId: "abc", AccessId: &accessId}, nil).
					Times(1)

				ua.EXPECT().
					DisableMember(gomock.Eq(accessId), gomock.Any()).
					Times(1)

				mdb.EXPECT().
					DeactivateMember(gomock.Eq("abc")).
					Times(1)
			},
			wantStatusCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mdb := NewMockmemberDb(ctrl)
			ua := NewMockuaUpdater(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mdb, ua)
			}
			l := New(secret, "", "", mdb, ua)

			mux := http.NewServeMux()
			mux.HandleFunc("POST /", l.webhookHandler)

			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, tt.request)

			if resp.Result().StatusCode != tt.wantStatusCode {
				t.Errorf(
					"Unexpected StatusCode. Wanted: %d, Got: %d",
					tt.wantStatusCode,
					resp.Result().StatusCode,
				)
			}
		})
	}
}
