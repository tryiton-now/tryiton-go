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
//
// The rest are optional overrides. DetailImages are up to 3 extra views (URLs
// or data URLs) that only sharpen the generated description. Description (at
// most 200 characters) is your own description of the item, used inside the
// generated prompt. Instructions (at most 1000 characters) is your own prompt,
// used verbatim instead of the generated one. EnhancePrompt lets the stream
// model enhance the prompt (true when nil); set it with Bool(false) to turn
// that off.
type PrepareLiveGarmentParams struct {
	GarmentImage  string   `json:"garment_image"`
	Category      string   `json:"category,omitempty"`
	Subcategory   string   `json:"subcategory,omitempty"`
	DetailImages  []string `json:"detail_images,omitempty"`
	Description   string   `json:"description,omitempty"`
	Instructions  string   `json:"instructions,omitempty"`
	EnhancePrompt *bool    `json:"enhance_prompt,omitempty"`
}

// PrepareLiveOutfitParams are the inputs for PrepareLiveOutfit. GarmentIDs are
// 2 or 3 distinct prepared garment IDs (e.g. a top, trousers and shoes), in any
// order. Instructions (at most 1000 characters) replaces the generated outfit
// prompt; the items' own Instructions don't carry over. EnhancePrompt is true
// when nil; set it with Bool(false) to turn it off.
type PrepareLiveOutfitParams struct {
	GarmentIDs    []string `json:"garment_ids"`
	Instructions  string   `json:"instructions,omitempty"`
	EnhancePrompt *bool    `json:"enhance_prompt,omitempty"`
}

// Bool returns a pointer to v, for optional boolean params such as
// PrepareLiveGarmentParams.EnhancePrompt.
func Bool(v bool) *bool {
	return &v
}

// LiveGarment is a garment or outfit prepared for live try-on. Pass ID to
// StartLiveSession (or to the browser's garment switch). Category is "outfit"
// for an outfit. Prompt is the stream prompt and EnhancePrompt whether the
// stream model enhances it. Items is an outfit's garment IDs, top to bottom,
// and nil for a single garment.
type LiveGarment struct {
	ID            string   `json:"id"`
	Category      string   `json:"category"`
	Subcategory   *string  `json:"subcategory"`
	Description   *string  `json:"description"`
	Prompt        string   `json:"prompt"`
	EnhancePrompt bool     `json:"enhancePrompt"`
	Items         []string `json:"items"`
	CreatedAt     string   `json:"createdAt"`
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

// PrepareLiveOutfit combines 2 or 3 prepared garments into one outfit the
// stream applies at once. The outfit is a LiveGarment (Category "outfit"):
// pass its ID wherever a garment ID goes (StartLiveSession, the browser's
// garment switch). It's cached per set of garments and overrides, in any order.
func (c *Client) PrepareLiveOutfit(ctx context.Context, p PrepareLiveOutfitParams) (*LiveGarment, error) {
	var out struct {
		Outfit LiveGarment `json:"outfit"`
	}
	if err := c.do(ctx, http.MethodPost, "/live/outfits", p, &out); err != nil {
		return nil, err
	}
	return &out.Outfit, nil
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
