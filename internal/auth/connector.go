package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/protorians/sentient-cli/internal/pkg"
)

// DefaultConnectAPI is the production base URL of the sentient-connect API.
const DefaultConnectAPI = "https://connect.sentient.protorians.com"

// EnvAPIBase overrides the default API base URL.
const EnvAPIBase = "SENTIENT_CONNECT_API"

// User is the authenticated developer.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// SignInRequest is the payload for POST /auth/sign-in.
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignInResponse is the payload returned by POST /auth/sign-in.
type SignInResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
	MFARequired  bool   `json:"mfaRequired"`
}

// ChallengeRequest is the payload for POST /mfa/challenge.
type ChallengeRequest struct {
	Email string `json:"email"`
}

// ChallengeResponse is returned by POST /mfa/challenge.
type ChallengeResponse struct {
	Factors []string `json:"factors"`
}

// VerifyRequest is the payload for MFA verification endpoints.
type VerifyRequest struct {
	Code string `json:"code"`
}

// TokenResponse is returned by MFA verification and /auth/refresh.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

// Connector talks to the sentient-connect API.
type Connector struct {
	Client *pkg.Client
}

// apiBaseURL resolves the sentient-connect base URL. Resolution order:
//  1. `SENTIENT_CONNECT_API` environment variable,
//  2. `app.config.json` (sentient-connect.api.baseUrl) when present in the
//     current working directory,
//  3. the production default.
func apiBaseURL() string {
	if v := os.Getenv(EnvAPIBase); v != "" {
		return v
	}
	if base, ok := appConfigBaseURL(); ok {
		return base
	}
	return DefaultConnectAPI
}

// appConfigBaseURL reads the workspace `app.config.json` when available and
// returns the sentient-connect `api.baseUrl`.
func appConfigBaseURL() (string, bool) {
	path := filepath.Join(".", "app.config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var cfg struct {
		Applications map[string]struct {
			API struct {
				BaseURL string `json:"baseUrl"`
			} `json:"api"`
		} `json:"applications"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", false
	}
	app, ok := cfg.Applications["sentient-connect"]
	if !ok || app.API.BaseURL == "" {
		return "", false
	}
	return app.API.BaseURL, true
}

// NewConnector builds a connector against the resolved API base URL.
func NewConnector() *Connector {
	return &Connector{
		Client: pkg.NewClient(apiBaseURL()),
	}
}

// SignIn authenticates with email + password.
func (c *Connector) SignIn(ctx context.Context, req SignInRequest) (*SignInResponse, error) {
	var out SignInResponse
	if err := c.Client.Do(ctx, "POST", "/auth/sign-in", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SignOut invalidates the current token server-side.
func (c *Connector) SignOut(ctx context.Context) error {
	return c.Client.Do(ctx, "POST", "/auth/sign-out", nil, nil)
}

// Refresh exchanges a refresh token for a new access token.
func (c *Connector) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	body := map[string]string{"refresh_token": refreshToken}
	var out TokenResponse
	if err := c.Client.Do(ctx, "POST", "/auth/refresh", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChallengeMFA requests the available MFA factors.
func (c *Connector) ChallengeMFA(ctx context.Context, email string) (*ChallengeResponse, error) {
	var out ChallengeResponse
	if err := c.Client.Do(ctx, "POST", "/mfa/challenge", ChallengeRequest{Email: email}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyTOTP verifies a 6-digit TOTP code.
func (c *Connector) VerifyTOTP(ctx context.Context, code string) (*TokenResponse, error) {
	var out TokenResponse
	if err := c.Client.Do(ctx, "POST", "/mfa/totp/verify", VerifyRequest{Code: code}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyRecovery verifies a backup/recovery code.
func (c *Connector) VerifyRecovery(ctx context.Context, code string) (*TokenResponse, error) {
	var out TokenResponse
	if err := c.Client.Do(ctx, "POST", "/mfa/recovery/verify", VerifyRequest{Code: code}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
