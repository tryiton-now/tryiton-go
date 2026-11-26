package tryiton

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// Unit tests for behavior the shared contract fixture (contract_test.go)
// doesn't cover.

var unitFixture = contractFixture{BaseURL: "https://api.test/api/v1", APIKey: "test_key", AuthHeader: "Bearer test_key"}

func jsonResponse(status int, body string) scriptedResponse {
	return scriptedResponse{Status: status, Body: json.RawMessage(body)}
}

func TestNewRequiresAPIKey(t *testing.T) {
	_, err := New("")
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Name != "ConfigError" {
		t.Fatalf("got %v, want a ConfigError", err)
	}
}

func TestPrepareLiveGarmentOmitsUnsetOptions(t *testing.T) {
	resp := jsonResponse(201, `{"ok":true,"garment":{"id":"g_1","category":"auto","subcategory":null,"description":null,"createdAt":"x"}}`)
	c, srv := newContractClient(t, unitFixture, []scriptedResponse{resp})
	g, err := c.PrepareLiveGarment(context.Background(), PrepareLiveGarmentParams{GarmentImage: "data:image/png;base64,AAAA"})
	if err != nil {
		t.Fatal(err)
	}
	assertRequests(t, unitFixture, srv.recorded(), []expectedRequest{
		{Method: "POST", Path: "/live/garments", Body: json.RawMessage(`{"garment_image":"data:image/png;base64,AAAA"}`)},
	})
	if g.ID != "g_1" || g.Subcategory != nil || g.Description != nil {
		t.Fatalf("unexpected garment %+v", g)
	}
}

func TestJobErrorJSON(t *testing.T) {
	cases := []struct {
		name, body string
		want       *JobError
	}{
		{"reason string", `{"status":"failed","output":[],"error":"No person found."}`, &JobError{Name: "ProcessingError", Message: "No person found."}},
		{"object", `{"status":"failed","output":[],"error":{"name":"Moderation","message":"Blocked."}}`, &JobError{Name: "Moderation", Message: "Blocked."}},
		{"null", `{"status":"completed","output":["https://cdn.test/a.png"],"error":null}`, nil},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var s Status
			if err := json.Unmarshal([]byte(tc.body), &s); err != nil {
				t.Fatal(err)
			}
			if !jobErrorsEqual(s.Error, tc.want) {
				t.Fatalf("got %+v, want %+v", s.Error, tc.want)
			}
		})
	}
}

func jobErrorsEqual(a, b *JobError) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestWaitForResultFailureFallbacks(t *testing.T) {
	cases := []struct {
		name, errorJSON   string
		wantName, wantMsg string
	}{
		{"no error given", `null`, "ProcessingError", "Try-on failed."},
		{"object without a name", `{"message":"Bad input."}`, "ProcessingError", "Bad input."},
		{"named object", `{"name":"Moderation","message":"Blocked."}`, "Moderation", "Blocked."},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := jsonResponse(200, `{"ok":true,"status":"failed","output":[],"error":`+tc.errorJSON+`}`)
			c, _ := newContractClient(t, unitFixture, []scriptedResponse{resp})
			_, err := c.WaitForResult(context.Background(), "job_1", 0)
			var apiErr *Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("got %T %v, want *Error", err, err)
			}
			if apiErr.Status != 0 || apiErr.Name != tc.wantName || apiErr.Message != tc.wantMsg {
				t.Fatalf("got {%d %q %q}, want {0 %q %q}", apiErr.Status, apiErr.Name, apiErr.Message, tc.wantName, tc.wantMsg)
			}
		})
	}
}
