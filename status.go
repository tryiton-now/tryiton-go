package tryiton

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// processingErrorName is the error name of a failed try-on job.
const processingErrorName = "ProcessingError"

// Status is a job's status snapshot.
type Status struct {
	Status string    `json:"status"` // "processing" | "completed" | "failed"
	Output []string  `json:"output"`
	Error  *JobError `json:"error"`
}

// JobError describes a runtime job failure. GET /status returns the failure
// reason as a plain string, which decodes to Name "ProcessingError" with the
// reason as Message. A {"name", "message"} object is accepted too.
type JobError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// jobErrorObject is JobError without its UnmarshalJSON, so it can use the
// default object decoding without recursing.
type jobErrorObject JobError

// UnmarshalJSON decodes a failure reason string or a {"name", "message"}
// object.
func (e *JobError) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var reason string
		if err := json.Unmarshal(data, &reason); err != nil {
			return err
		}
		*e = JobError{Name: processingErrorName, Message: reason}
		return nil
	}
	return json.Unmarshal(data, (*jobErrorObject)(e))
}

// Credits is your credit balance.
type Credits struct {
	OnDemand     int `json:"on_demand"`
	Subscription int `json:"subscription"`
	Purchased    int `json:"purchased"`
	Reserved     int `json:"reserved"`
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
// URLs. It returns an *Error if the job fails or the context is cancelled. A
// failed job's *Error has Status 0, Name "ProcessingError" (unless the API
// names it otherwise) and the job's failure reason as Message.
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
			return nil, jobFailure(s.Error)
		}
		select {
		case <-ctx.Done():
			return nil, &Error{Name: "Timeout", Message: ctx.Err().Error()}
		case <-time.After(pollInterval):
		}
	}
}

// jobFailure maps a failed job's error (nil when the API gave none) to *Error.
func jobFailure(je *JobError) *Error {
	e := &Error{Name: processingErrorName, Message: "Try-on failed."}
	if je != nil && je.Name != "" {
		e.Name = je.Name
	}
	if je != nil && je.Message != "" {
		e.Message = je.Message
	}
	return e
}
