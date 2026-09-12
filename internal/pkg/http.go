package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultHTTPTimeout is the default timeout for API calls (see app.config.json).
const DefaultHTTPTimeout = 30 * time.Second

// RaitonResponse is the standard Raiton API envelope `{message, data, statusCode}`
// returned by every endpoint of `sentient-api-core` / `sentient-api-connect`
// (see `RaitonResponses(message, data, statusCode)` in the Raiton framework).
type RaitonResponse struct {
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
	StatusCode int             `json:"statusCode"`
}

// Client is a thin JSON-aware HTTP client used to talk to the sentient APIs.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Token   string
}

// NewClient builds a client for the given base URL with a 30s timeout.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: DefaultHTTPTimeout},
	}
}

// Do performs a request with the given method, path, body and decodes the
// `data` field of the Raiton envelope into out (when out is not nil).
func (c *Client) Do(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("sérialisation du corps de requête impossible : %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("construction de la requête impossible : %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sentient-cli")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("erreur réseau : %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("lecture de la réponse impossible : %w", err)
	}

	if resp.StatusCode >= 400 {
		if apiErr := parseAPIError(resp.StatusCode, data); apiErr != nil {
			return apiErr
		}
		return fmt.Errorf("réponse HTTP %d : %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if out != nil && len(data) > 0 {
		var envelope RaitonResponse
		if err := json.Unmarshal(data, &envelope); err != nil {
			return fmt.Errorf("décodage de la réponse impossible : %w", err)
		}
		if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return nil
		}
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("décodage de la réponse impossible : %w", err)
		}
	}
	return nil
}

// parseAPIError extracts an APIError from an HTTP error body. Errors follow
// the Raiton envelope (`{message, statusCode, data}`); legacy flat bodies
// `{code, message}` are still accepted.
func parseAPIError(statusCode int, data []byte) *APIError {
	var envelope RaitonResponse
	if json.Unmarshal(data, &envelope) == nil && envelope.Message != "" {
		code := ""
		var legacy struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &legacy) == nil {
			code = legacy.Code
		}
		return &APIError{StatusCode: statusCode, Code: code, Message: envelope.Message}
	}
	var apiErr APIError
	if json.Unmarshal(data, &apiErr) == nil && apiErr.Message != "" {
		apiErr.StatusCode = statusCode
		return &apiErr
	}
	return nil
}

// APIError represents an error returned by the sentient API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (code=%s, HTTP %d)", e.Message, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Message, e.StatusCode)
}

// IsNotFound reports whether the error is an HTTP 404.
func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.StatusCode == http.StatusNotFound
}
