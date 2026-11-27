package tryiton

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// The contract fixture is shared by every TryItOn SDK. testdata/sdk-cases.json
// is a generated copy: edit tryiton/backend/api-contract/sdk-cases.json and
// re-sync instead. The fixture types below have no JSON tags because
// encoding/json matches field names case-insensitively ("baseUrl" -> BaseURL).

type contractFixture struct {
	BaseURL     string
	APIKey      string
	AuthHeader  string
	UserAgent   userAgentSpec
	MethodCases []methodCase
	ErrorCases  []errorCase
}

type userAgentSpec struct {
	Pattern string
}

type methodCase struct {
	Name                       string
	Method                     string
	Params                     map[string]any
	Responses                  []scriptedResponse
	ExpectRequests             []expectedRequest
	ExpectResult               json.RawMessage
	ExpectResultIsResponseBody bool
	ExpectError                *expectedError
}

type errorCase struct {
	Name        string
	Response    scriptedResponse
	ExpectError expectedError
}

// scriptedResponse is a JSON Body, or a Raw body when Raw is set.
type scriptedResponse struct {
	Status int
	Body   json.RawMessage
	Raw    *string
}

type expectedRequest struct {
	Method string
	Path   string
	Body   json.RawMessage
}

// expectedError uses pointers because the fixture writes null for "none".
type expectedError struct {
	Status       *int
	ErrorName    *string
	Message      *string
	MessageAnyOf []string
}

func loadFixture(t *testing.T) contractFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/sdk-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var f contractFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(f.MethodCases) == 0 || len(f.ErrorCases) == 0 {
		t.Fatal("fixture has no methodCases or errorCases")
	}
	return f
}

type recordedCall struct {
	method      string
	path        string
	auth        string
	userAgent   string
	contentType string
	body        []byte
}

// contractServer replies with its scripted responses in order and records
// every request, with the path made relative to the fixture's base path.
type contractServer struct {
	t         *testing.T
	basePath  string
	responses []scriptedResponse

	mu    sync.Mutex
	calls []recordedCall
}

// newContractClient starts a contractServer and returns a Client pointed at it.
func newContractClient(t *testing.T, f contractFixture, responses []scriptedResponse) (*Client, *contractServer) {
	t.Helper()
	base, err := url.Parse(f.BaseURL)
	if err != nil {
		t.Fatalf("fixture baseUrl: %v", err)
	}
	srv := &contractServer{t: t, basePath: strings.TrimRight(base.Path, "/"), responses: responses}
	hs := httptest.NewServer(srv)
	t.Cleanup(hs.Close)

	c, err := New(f.APIKey, WithBaseURL(hs.URL+srv.basePath))
	if err != nil {
		t.Fatal(err)
	}
	return c, srv
}

func (s *contractServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	path := r.URL.EscapedPath()
	if !strings.HasPrefix(path, s.basePath+"/") {
		s.t.Errorf("request path %q is outside the base path %q", path, s.basePath)
	}
	call := recordedCall{
		method:      r.Method,
		path:        strings.TrimPrefix(path, s.basePath),
		auth:        r.Header.Get("Authorization"),
		userAgent:   r.Header.Get("User-Agent"),
		contentType: r.Header.Get("Content-Type"),
		body:        body,
	}

	s.mu.Lock()
	n := len(s.calls)
	s.calls = append(s.calls, call)
	s.mu.Unlock()

	if n >= len(s.responses) {
		s.t.Errorf("unexpected request #%d: %s %s", n+1, call.method, call.path)
		http.Error(w, "no response scripted", http.StatusInternalServerError)
		return
	}
	resp := s.responses[n]
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	if resp.Raw != nil {
		_, _ = io.WriteString(w, *resp.Raw)
		return
	}
	_, _ = w.Write(resp.Body)
}

func (s *contractServer) recorded() []recordedCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedCall(nil), s.calls...)
}

func TestContractMethodCases(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.MethodCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			c, srv := newContractClient(t, f, tc.Responses)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := callContractMethod(ctx, t, c, tc.Method, tc.Params)
			assertRequests(t, f, srv.recorded(), tc.ExpectRequests)

			switch {
			case tc.ExpectError != nil:
				assertContractError(t, err, *tc.ExpectError)
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.ExpectResultIsResponseBody:
				assertResultJSON(t, result, tc.Responses[len(tc.Responses)-1].Body)
			case len(tc.ExpectResult) > 0:
				assertResultJSON(t, result, tc.ExpectResult)
			default:
				t.Fatal("case has no expectation")
			}
		})
	}
}

func TestContractErrorCases(t *testing.T) {
	f := loadFixture(t)
	calls := []struct {
		name string
		path string
		call func(ctx context.Context, c *Client) error
	}{
		{"getCredits", "/credits", func(ctx context.Context, c *Client) error {
			_, err := c.GetCredits(ctx)
			return err
		}},
		{"getLiveSession", "/live/sessions/s1", func(ctx context.Context, c *Client) error {
			_, err := c.GetLiveSession(ctx, "s1")
			return err
		}},
	}
	for _, ec := range f.ErrorCases {
		for _, m := range calls {
			ec, m := ec, m
			t.Run(ec.Name+"/"+m.name, func(t *testing.T) {
				c, srv := newContractClient(t, f, []scriptedResponse{ec.Response})
				err := m.call(context.Background(), c)
				assertRequests(t, f, srv.recorded(), []expectedRequest{{Method: http.MethodGet, Path: m.path}})
				assertContractError(t, err, ec.ExpectError)
			})
		}
	}
}

func TestVersionMatchesUserAgentPattern(t *testing.T) {
	f := loadFixture(t)
	if !regexp.MustCompile(f.UserAgent.Pattern).MatchString(userAgent) {
		t.Fatalf("User-Agent %q does not match %q", userAgent, f.UserAgent.Pattern)
	}
	if userAgent != "tryiton-go/"+Version {
		t.Fatalf("User-Agent %q is not built from Version %q", userAgent, Version)
	}
}

// callContractMethod maps a canonical fixture method name and its camelCase
// params to the Go method.
func callContractMethod(ctx context.Context, t *testing.T, c *Client, method string, p map[string]any) (any, error) {
	t.Helper()
	switch method {
	case "tryOnFashion":
		return c.TryOnFashion(ctx, FashionParams{
			ModelImage:      paramString(p, "modelImage"),
			GarmentImage:    paramString(p, "garmentImage"),
			Category:        paramString(p, "category"),
			Subcategory:     paramString(p, "subcategory"),
			Mode:            paramString(p, "mode"),
			NumSamples:      paramInt(p, "numSamples"),
			OutputFormat:    paramString(p, "outputFormat"),
			ModerationLevel: paramString(p, "moderationLevel"),
		})
	case "tryOnHairstyle":
		return c.TryOnHairstyle(ctx, HairstyleParams{
			FaceImage:    paramString(p, "faceImage"),
			Haircut:      paramString(p, "haircut"),
			HairColor:    paramString(p, "hairColor"),
			NumSamples:   paramInt(p, "numSamples"),
			OutputFormat: paramString(p, "outputFormat"),
		})
	case "tryOnTattoo":
		return c.TryOnTattoo(ctx, TattooParams{
			BodyImage:    paramString(p, "bodyImage"),
			DesignImage:  paramString(p, "designImage"),
			Placement:    paramString(p, "placement"),
			Region:       paramRegion(p, "region"),
			NumSamples:   paramInt(p, "numSamples"),
			OutputFormat: paramString(p, "outputFormat"),
		})
	case "getStatus":
		return c.GetStatus(ctx, paramString(p, "jobId"))
	case "getCredits":
		return c.GetCredits(ctx)
	case "waitForResult":
		// The smallest interval WaitForResult accepts (<= 0 means the default).
		return c.WaitForResult(ctx, paramString(p, "jobId"), time.Nanosecond)
	case "getLiveConfig":
		return c.GetLiveConfig(ctx)
	case "prepareLiveGarment":
		return c.PrepareLiveGarment(ctx, PrepareLiveGarmentParams{
			GarmentImage:  paramString(p, "garmentImage"),
			Category:      paramString(p, "category"),
			Subcategory:   paramString(p, "subcategory"),
			DetailImages:  paramStrings(p, "detailImages"),
			Description:   paramString(p, "description"),
			Instructions:  paramString(p, "instructions"),
			EnhancePrompt: paramBool(p, "enhancePrompt"),
		})
	case "prepareLiveOutfit":
		return c.PrepareLiveOutfit(ctx, PrepareLiveOutfitParams{
			GarmentIDs:    paramStrings(p, "garmentIds"),
			Instructions:  paramString(p, "instructions"),
			EnhancePrompt: paramBool(p, "enhancePrompt"),
		})
	case "startLiveSession":
		return c.StartLiveSession(ctx, StartLiveSessionParams{
			GarmentID:     paramString(p, "garmentId"),
			MaxSeconds:    paramInt(p, "maxSeconds"),
			EndUserID:     paramString(p, "endUserId"),
			AllowedOrigin: paramString(p, "allowedOrigin"),
		})
	case "getLiveSession":
		return c.GetLiveSession(ctx, paramString(p, "sessionId"))
	case "stopLiveSession":
		return c.StopLiveSession(ctx, paramString(p, "sessionId"))
	}
	t.Fatalf("no Go mapping for fixture method %q", method)
	return nil, nil
}

func paramString(p map[string]any, key string) string {
	s, _ := p[key].(string)
	return s
}

func paramInt(p map[string]any, key string) int {
	n, _ := p[key].(float64)
	return int(n)
}

func paramStrings(p map[string]any, key string) []string {
	list, _ := p[key].([]any)
	var out []string
	for _, v := range list {
		s, _ := v.(string)
		out = append(out, s)
	}
	return out
}

// paramBool is nil when the case leaves the param out, so false stays distinct.
func paramBool(p map[string]any, key string) *bool {
	b, ok := p[key].(bool)
	if !ok {
		return nil
	}
	return &b
}

func paramRegion(p map[string]any, key string) *TattooRegion {
	m, ok := p[key].(map[string]any)
	if !ok {
		return nil
	}
	side := func(k string) float64 {
		v, _ := m[k].(float64)
		return v
	}
	return &TattooRegion{X: side("x"), Y: side("y"), W: side("w"), H: side("h")}
}

func assertRequests(t *testing.T, f contractFixture, got []recordedCall, want []expectedRequest) {
	t.Helper()
	wantUA := "tryiton-go/" + Version
	if len(got) != len(want) {
		t.Errorf("got %d requests, want %d", len(got), len(want))
	}
	for i := 0; i < len(got) && i < len(want); i++ {
		g, w := got[i], want[i]
		if g.method != w.Method || g.path != w.Path {
			t.Errorf("request %d: got %s %s, want %s %s", i, g.method, g.path, w.Method, w.Path)
		}
		if g.auth != f.AuthHeader {
			t.Errorf("request %d: Authorization %q, want %q", i, g.auth, f.AuthHeader)
		}
		if g.userAgent != wantUA {
			t.Errorf("request %d: User-Agent %q, want %q", i, g.userAgent, wantUA)
		}
		if len(w.Body) == 0 {
			if len(g.body) != 0 || g.contentType != "" {
				t.Errorf("request %d: want no body, got %q (Content-Type %q)", i, g.body, g.contentType)
			}
			continue
		}
		if g.contentType != "application/json" {
			t.Errorf("request %d: Content-Type %q, want application/json", i, g.contentType)
		}
		assertSameJSON(t, "request body", g.body, w.Body)
	}
}

// assertResultJSON marshals a Go result and compares it with the expected JSON.
func assertResultJSON(t *testing.T, result any, want json.RawMessage) {
	t.Helper()
	// The API sends a failed job's error as a plain string; Status keeps it as
	// JobError{Message}. Compare in the wire shape.
	if s, ok := result.(*Status); ok && s != nil {
		var reason any
		if s.Error != nil {
			reason = s.Error.Message
		}
		result = map[string]any{"status": s.Status, "output": s.Output, "error": reason}
	}
	got, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	assertSameJSON(t, "result", got, want)
}

// assertSameJSON compares two JSON documents semantically.
func assertSameJSON(t *testing.T, label string, got, want []byte) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Errorf("%s: invalid JSON %q: %v", label, got, err)
		return
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("%s: invalid expected JSON %q: %v", label, want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("%s mismatch\n got: %s\nwant: %s", label, got, want)
	}
}

// assertContractError checks a *Error against the fixture. A null status means
// no HTTP status (a failed job), which the Go SDK reports as Status 0; a null
// errorName is an empty Name.
func assertContractError(t *testing.T, err error, want expectedError) {
	t.Helper()
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("got error %T %v, want *Error", err, err)
	}
	wantStatus, wantName := 0, ""
	if want.Status != nil {
		wantStatus = *want.Status
	}
	if want.ErrorName != nil {
		wantName = *want.ErrorName
	}
	if apiErr.Status != wantStatus || apiErr.Name != wantName {
		t.Errorf("got Status %d Name %q, want Status %d Name %q", apiErr.Status, apiErr.Name, wantStatus, wantName)
	}
	switch {
	case want.Message != nil:
		if apiErr.Message != *want.Message {
			t.Errorf("got Message %q, want %q", apiErr.Message, *want.Message)
		}
	case len(want.MessageAnyOf) > 0:
		if !containsString(want.MessageAnyOf, apiErr.Message) {
			t.Errorf("got Message %q, want one of %q", apiErr.Message, want.MessageAnyOf)
		}
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
