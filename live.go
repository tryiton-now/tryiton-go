package tryiton

import (
	"context"
	"net/http"
	"net/url"
)

// Live (real-time webcam) try-on. These calls run on your server with the API
// key. The end user's browser streams with the `tryiton-live` npm package,
// using the response of StartLiveSession (session-scoped credentials, never
// your API key).

// LiveConfig is live try-on availability and pricing for your account.
// SecondsPerCredit is the seconds of streaming one credit buys, MaxSeconds the
// longest session allowed, MinCredits the credits needed to start a session and
// MaxConcurrentSessions the parallel sessions allowed per account.
type LiveConfig struct {
	Enabled               bool `json:"enabled"`
	SecondsPerCredit      int  `json:"secondsPerCredit"`
	MaxSeconds            int  `json:"maxSeconds"`
	MinCredits            int  `json:"minCredits"`
	MaxConcurrentSessions int  `json:"maxConcurrentSessions"`
}

// PrepareLiveGarmentParams are the inputs for PrepareLiveGarment. GarmentImage
// is a public URL or base64 data URL (a product photo on a plain background
// works best). Category is recommended (same values as FashionParams); omit it
// to let it be detected. Subcategory is e.g. "tops" for clothing.
type PrepareLiveGarmentParams struct {
	GarmentImage string `json:"garment_image"`
	Category     string `json:"category,omitempty"`
	Subcategory  string `json:"subcategory,omitempty"`
}

// LiveGarment is a garment prepared for live try-on. Pass ID to
// StartLiveSession (or to the browser's garment switch).
type LiveGarment struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Subcategory *string `json:"subcategory"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
}

// StartLiveSessionParams are the optional inputs for StartLiveSession.
// GarmentID is a prepared garment to start with. MaxSeconds caps the session
// (10 to LiveConfig.MaxSeconds). EndUserID is your id for the end user: one
// stream per end user, and a restart ends their previous one. AllowedOrigin
// locks the media token to your site, e.g. "https://shop.example.com".
type StartLiveSessionParams struct {
	GarmentID     string `json:"garment_id,omitempty"`
	MaxSeconds    int    `json:"max_seconds,omitempty"`
	EndUserID     string `json:"end_user_id,omitempty"`
	AllowedOrigin string `json:"allowed_origin,omitempty"`
}

// LiveSession is a live session's status and usage. Status is "active" or
// "ended"; EndReason is e.g. "user", "time_limit", "out_of_credits",
// "superseded" or "error". Seconds is the streamed seconds once ended and the
// billed seconds while running.
type LiveSession struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	EndReason      *string `json:"endReason"`
	Seconds        int     `json:"seconds"`
	CreditsCharged int     `json:"creditsCharged"`
	EndUserID      *string `json:"endUserId"`
	CreatedAt      string  `json:"createdAt"`
	EndedAt        *string `json:"endedAt"`
}

// LiveConnection is the media connection the browser opens.
type LiveConnection struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Token    string `json:"token"`
}

// LiveGarmentInput is a garment's stream input.
type LiveGarmentInput struct {
	Prompt         string `json:"prompt"`
	EnhancePrompt  bool   `json:"enhancePrompt"`
	ReferenceImage string `json:"referenceImage"`
}

// LiveSessionGarment is the first garment of a session, with its stream input.
type LiveSessionGarment struct {
	LiveGarment
	Input *LiveGarmentInput `json:"input,omitempty"`
}

// LiveSessionStart is the response of StartLiveSession. Its JSON tags match the
// API response, so json.Marshal(start) reproduces the body to send to the
// browser, which passes it to connectLiveTryOn from `tryiton-live`.
// ClientSecret is the session-scoped secret the browser uses for billing
// heartbeats. Garment is nil unless StartLiveSessionParams.GarmentID was set.
type LiveSessionStart struct {
	OK               bool                `json:"ok"`
	Session          LiveSession         `json:"session"`
	ClientSecret     string              `json:"clientSecret"`
	Connection       LiveConnection      `json:"connection"`
	SecondsPerCredit int                 `json:"secondsPerCredit"`
	MaxSeconds       int                 `json:"maxSeconds"`
	SecondsLeft      int                 `json:"secondsLeft"`
	Garment          *LiveSessionGarment `json:"garment"`
}

// GetLiveConfig fetches live try-on availability and pricing for your account.
func (c *Client) GetLiveConfig(ctx context.Context) (*LiveConfig, error) {
	var cfg LiveConfig
	if err := c.do(ctx, http.MethodGet, "/live/config", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// PrepareLiveGarment prepares a garment for live try-on. Do it once per product
// ahead of time (e.g. at catalog sync): it's cached per image and category, so
// starting a session or switching garments later is instant.
func (c *Client) PrepareLiveGarment(ctx context.Context, p PrepareLiveGarmentParams) (*LiveGarment, error) {
	var out struct {
		Garment LiveGarment `json:"garment"`
	}
	if err := c.do(ctx, http.MethodPost, "/live/garments", p, &out); err != nil {
		return nil, err
	}
	return &out.Garment, nil
}

// StartLiveSession starts a live session for one end user and holds the first
// credit. Send the result to the browser unchanged (json.Marshal reproduces the
// API body); it streams with connectLiveTryOn from the `tryiton-live` package.
func (c *Client) StartLiveSession(ctx context.Context, p StartLiveSessionParams) (*LiveSessionStart, error) {
	var start LiveSessionStart
	if err := c.do(ctx, http.MethodPost, "/live/sessions", p, &start); err != nil {
		return nil, err
	}
	return &start, nil
}

// GetLiveSession fetches a live session's status and usage.
func (c *Client) GetLiveSession(ctx context.Context, sessionID string) (*LiveSession, error) {
	return c.liveSession(ctx, http.MethodGet, "/live/sessions/"+url.PathEscape(sessionID), nil)
}

// StopLiveSession ends a live session from your server (e.g. the user left).
// It is idempotent.
func (c *Client) StopLiveSession(ctx context.Context, sessionID string) (*LiveSession, error) {
	return c.liveSession(ctx, http.MethodPost, "/live/sessions/"+url.PathEscape(sessionID)+"/stop", struct{}{})
}

func (c *Client) liveSession(ctx context.Context, method, path string, body any) (*LiveSession, error) {
	var out struct {
		Session LiveSession `json:"session"`
	}
	if err := c.do(ctx, method, path, body, &out); err != nil {
		return nil, err
	}
	return &out.Session, nil
}
