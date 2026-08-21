package domain

import "fmt"

// UpstreamError reports a non-2xx response from an upstream API, carrying the
// HTTP status code so delivery handlers can map client errors (4xx) distinctly
// from upstream failures (5xx).
type UpstreamError struct {
	Status int
	Msg    string
	// RetryAfter carries the upstream rate-limit hint for 429 responses.
	RetryAfter string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream status %d: %s", e.Status, e.Msg)
}
