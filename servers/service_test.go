package servers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

type updateSpy struct {
	IStore
	fields bson.M
}

func (s *updateSpy) Update(_ context.Context, _ string, fields bson.M) error {
	s.fields = fields
	return nil
}

func (s *updateSpy) GetByID(context.Context, string) (Server, error) {
	return Server{}, nil
}

func updateWith(t *testing.T, input UpdateInput) bson.M {
	t.Helper()

	spy := &updateSpy{}

	if _, err := (&service{store: spy}).Update(context.Background(), "srv1", input); err != nil {
		t.Fatalf("update: %v", err)
	}

	return spy.fields
}

func ptr[T any](v T) *T { return &v }

func TestUpdateKeepsExistingTokenWhenBlank(t *testing.T) {
	for name, token := range map[string]string{"empty": "", "blank": "   "} {
		fields := updateWith(t, UpdateInput{AuthToken: ptr(token)})

		if _, present := fields["auth_token"]; present {
			t.Errorf("%s token: auth_token = %v, want the field to be left untouched", name, fields["auth_token"])
		}
	}
}

func TestUpdateStoresTrimmedToken(t *testing.T) {
	fields := updateWith(t, UpdateInput{AuthToken: ptr("  new-token  ")})

	if got := fields["auth_token"]; got != "new-token" {
		t.Errorf("auth_token = %v, want %q", got, "new-token")
	}
}

func TestUpdateIgnoresOmittedFields(t *testing.T) {
	fields := updateWith(t, UpdateInput{Location: ptr("Berlin")})

	if len(fields) != 1 || fields["location"] != "Berlin" {
		t.Errorf("fields = %v, want only {location: Berlin}", fields)
	}
}

func callTestPanel(t *testing.T, handler http.HandlerFunc) error {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	server := Server{ID: "srv1", ApiUrl: srv.URL, AuthToken: "token"}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/panel/api/clients/add", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	authorize(req, server)

	return callPanel(srv.Client(), req, server)
}

func TestCallPanelFailsOnUnsuccessfulEnvelope(t *testing.T) {
	err := callTestPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":false,"msg":"duplicate email"}`))
	})

	if err == nil {
		t.Fatal("success=false in a 200 response was treated as success")
	}

	if !strings.Contains(err.Error(), "duplicate email") {
		t.Errorf("error %q does not carry the panel message", err)
	}
}

func TestCallPanelAcceptsSuccessfulEnvelope(t *testing.T) {
	err := callTestPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":null}`))
	})

	if err != nil {
		t.Errorf("successful response rejected: %v", err)
	}
}

func TestCallPanelReportsRejectedToken(t *testing.T) {
	err := callTestPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	if err == nil || !strings.Contains(err.Error(), "rejected the auth token") {
		t.Errorf("err = %v, want a rejected-token error", err)
	}
}

func TestAuthorizeSetsBearerAndAjaxHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://panel.example", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	authorize(req, Server{AuthToken: "secret-token"})

	if got := req.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer secret-token")
	}

	if got := req.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
		t.Errorf("X-Requested-With = %q, want XMLHttpRequest", got)
	}
}

func validInput() CreateInput {
	return CreateInput{
		Location:  "Amsterdam",
		AuthToken: "3x-ui=session-value",
		Ip:        "203.0.113.10",
		ApiUrl:    "http://203.0.113.10:2053",
		Port:      443,
		InBoundID: 1,
	}
}

func TestValidateCreateInputAcceptsValidServer(t *testing.T) {
	if err := validateCreateInput(validInput()); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
}

func TestValidateCreateInputRejectsBadValues(t *testing.T) {
	cases := map[string]func(*CreateInput){
		"empty location":   func(in *CreateInput) { in.Location = "  " },
		"empty ip":         func(in *CreateInput) { in.Ip = "" },
		"empty api url":    func(in *CreateInput) { in.ApiUrl = "" },
		"empty auth token": func(in *CreateInput) { in.AuthToken = "" },
		"blank auth token": func(in *CreateInput) { in.AuthToken = "   " },
		"zero port":        func(in *CreateInput) { in.Port = 0 },
		"port too large":   func(in *CreateInput) { in.Port = 70000 },
		"negative limit":   func(in *CreateInput) { in.MaxClients = -1 },
		"negative port":    func(in *CreateInput) { in.Port = -1 },
	}

	for name, mutate := range cases {
		input := validInput()
		mutate(&input)

		err := validateCreateInput(input)
		if err == nil {
			t.Errorf("%s: expected an error, got nil", name)
			continue
		}

		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error %v does not wrap ErrInvalidInput", name, err)
		}
	}
}
