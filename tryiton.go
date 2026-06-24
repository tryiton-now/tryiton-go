// Package tryiton is the official Go SDK for the TryItOn virtual try-on API.
//
// See https://docs.tryiton.now for the full API reference.
package tryiton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API base URL.
const DefaultBaseURL = "https://tryiton.now/api/v1"

// Haircuts lists the supported `haircut` values for hairstyle try-on.
var Haircuts = []string{
	"Afro", "BobCut", "BowlCut", "BoxBraids", "BuzzCut", "Chignon", "CombOver",
	"CornrowBraids", "CurlyBob", "CurlyShag", "DoubleBun", "Dreadlocks", "FauxHawk",
	"FishtailBraid", "LongCurly", "LongHairTiedUp", "LongHimeCut", "LongStraight",
	"LongTwintails", "LongWavy", "LongWavyCurtainBangs", "ManBun", "MessyTousled",
	"PixieCut", "Pompadour", "Ponytail", "ShortCurlyPixie", "ShortTwintails",
	"ShoulderLengthHair", "Spiky", "TexturedFringe", "TwinBraids", "Updo", "WavyShag",
}

// Error is returned for API-level errors and runtime (job) failures.
type Error struct {
	// Status is the HTTP status code, or 0 for a runtime job failure.
	Status int
	// Name is the API error name, e.g. "OutOfCredits" or "ProcessingError".
	Name string
	// Message is the human-readable message.
	Message string
}

func (e *Error) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("tryiton: %s (%s)", e.Message, e.Name)
	}
	return "tryiton: " + e.Message
}

// Client is a TryItOn API client. Create one with New.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithHTTPClient supplies a custom *http.Client (e.g. with a timeout).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// New creates a Client with the given API key.
func New(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, &Error{Name: "ConfigError", Message: "an apiKey is required"}
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// ClothesParams are the inputs for a clothing/accessory try-on. Image fields
// accept a public URL or a base64 data URL. Category is one of: auto, clothing,
// eyewear, footwear, headwear, jewelry, accessories, others. clothing, jewelry,
// and accessories require a Subcategory.
// NumSamples is the number of output images (1-4); charged per image.
// OutputFormat is "png" or "jpeg" (defaults to "png"). ModerationLevel is one of
// "conservative", "permissive", or "none".
type ClothesParams struct {
	ModelImage      string `json:"model_image"`
	GarmentImage    string `json:"garment_image"`
	Category        string `json:"category,omitempty"`
	Subcategory     string `json:"subcategory,omitempty"`
	Mode            string `json:"mode,omitempty"`
	NumSamples      int    `json:"num_samples,omitempty"`
	OutputFormat    string `json:"output_format,omitempty"`
	ModerationLevel string `json:"moderation_level,omitempty"`
}

// HairstyleParams are the inputs for a hairstyle try-on. NumSamples is the
// number of output images (1-4); OutputFormat is "png" or "jpeg".
type HairstyleParams struct {
	FaceImage    string `json:"face_image"`
	Haircut      string `json:"haircut"`
	HairColor    string `json:"hair_color,omitempty"`
	NumSamples   int    `json:"num_samples,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// TattooParams are the inputs for a tattoo try-on. NumSamples is the number of
// output images (1-4); OutputFormat is "png" or "jpeg".
type TattooParams struct {
	BodyImage    string `json:"body_image"`
	DesignImage  string `json:"design_image"`
	Placement    string `json:"placement,omitempty"`
	NumSamples   int    `json:"num_samples,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// Status is a job's status snapshot.
type Status struct {
	Status string     `json:"status"` // "processing" | "completed" | "failed"
	Output []string   `json:"output"`
	Error  *JobError  `json:"error"`
}

// JobError describes a runtime job failure.
type JobError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Credits is your credit balance.
type Credits struct {
	OnDemand     int `json:"on_demand"`
	Subscription int `json:"subscription"`
	Purchased    int `json:"purchased"`
	Reserved     int `json:"reserved"`
}

// TryOnClothes puts a garment or accessory on a person and returns the job id.
func (c *Client) TryOnClothes(ctx context.Context, p ClothesParams) (string, error) {
	return c.submit(ctx, "/tryon/clothes", p)
}

// TryOnHairstyle restyles a person's hair and returns the job id.
func (c *Client) TryOnHairstyle(ctx context.Context, p HairstyleParams) (string, error) {
	return c.submit(ctx, "/tryon/hairstyle", p)
}

// TryOnTattoo inks a design onto skin and returns the job id.
func (c *Client) TryOnTattoo(ctx context.Context, p TattooParams) (string, error) {
	return c.submit(ctx, "/tryon/tattoo", p)
}

func (c *Client) submit(ctx context.Context, path string, body any) (string, error) {
	var out struct {
		JobID string `json:"jobId"`
	}
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return "", err
	}
	return out.JobID, nil
}

// GetStatus fetches the current status of a job.
func (c *Client) GetStatus(ctx context.Context, jobID string) (*Status, error) {
	var s Status
	if err := c.do(ctx, http.MethodGet, "/status/"+url.PathEscape(jobID), nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetCredits fetches your current credit balance.
func (c *Client) GetCredits(ctx context.Context) (*Credits, error) {
	var out struct {
		Credits Credits `json:"credits"`
	}
	if err := c.do(ctx, http.MethodGet, "/credits", nil, &out); err != nil {
		return nil, err
	}
	return &out.Credits, nil
}

// WaitForResult polls a job until it completes, then returns the output image
// URLs. It returns an *Error if the job fails or the context is cancelled.
func (c *Client) WaitForResult(ctx context.Context, jobID string, pollInterval time.Duration) ([]string, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	for {
		s, err := c.GetStatus(ctx, jobID)
		if err != nil {
			return nil, err
		}
		switch s.Status {
		case "completed":
			return s.Output, nil
		case "failed":
			name, msg := "ProcessingError", "Try-on failed."
			if s.Error != nil {
				name, msg = s.Error.Name, s.Error.Message
			}
			return nil, &Error{Name: name, Message: msg}
		}
		select {
		case <-ctx.Done():
			return nil, &Error{Name: "Timeout", Message: ctx.Err().Error()}
		case <-time.After(pollInterval):
		}
	}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	hasBody := body != nil
	if hasBody {
		b, err := json.Marshal(body)
		if err != nil {
			return &Error{Name: "EncodeError", Message: err.Error()}
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return &Error{Name: "RequestError", Message: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Error{Name: "NetworkError", Message: err.Error()}
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &apiErr)
		msg := apiErr.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &Error{Status: resp.StatusCode, Name: apiErr.Error, Message: msg}
	}

	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return &Error{Name: "DecodeError", Message: err.Error()}
		}
	}
	return nil
}
