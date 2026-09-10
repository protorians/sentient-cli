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

// Client is a thin JSON-aware HTTP client used to talk to sentient-connect.
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
// JSON response into out (when out is not nil).
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
		var apiErr APIError
		if json.Unmarshal(data, &apiErr) == nil {
			apiErr.StatusCode = resp.StatusCode
			if apiErr.Message == "" {
				apiErr.Message = strings.TrimSpace(string(data))
			}
			return &apiErr
		}
		return fmt.Errorf("réponse HTTP %d : %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("décodage de la réponse impossible : %w", err)
		}
	}
	return nil
}

// APIError represents an error returned by the sentient-connect API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (%d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s (%d)", e.Message, e.StatusCode)
}

// IsNotFound reports whether the error is an HTTP 404.
func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.StatusCode == http.StatusNotFound
}
