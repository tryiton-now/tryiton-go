package tryiton

import (
	"encoding/json"
	"fmt"
)

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

// apiError maps a non-2xx response to *Error. The API envelope is
// {"ok": false, "error": {"name", "message"}}; older shapes put "error" and
// "message" strings at the top level. Falls back to "HTTP <status>".
func apiError(status int, data []byte) *Error {
	var body struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	_ = json.Unmarshal(data, &body)

	var name, msg string
	var nested struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body.Error, &nested) == nil {
		name, msg = nested.Name, nested.Message
	} else {
		_ = json.Unmarshal(body.Error, &name)
	}
	if msg == "" {
		msg = body.Message
	}
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	return &Error{Status: status, Name: name, Message: msg}
}
