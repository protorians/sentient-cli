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

// API paths (global `/api` prefix, Raiton envelope responses).
const (
	APISignInPage   = "/api/auth/sign-in"
	APILogoutPage   = "/api/auth/logout"
	APIRefreshPage  = "/api/auth/sessions/refresh"
	APIChallengePage = "/api/mfa/challenge"
	APITOTPVerify   = "/api/mfa/totp/verify"
	APIRecoveryVerify = "/api/mfa/recovery/verify"
)

// Role is a role carried by the authenticated user (UserVm.roles).
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// User is the authenticated developer (UserVm).
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Status   string `json:"status,omitempty"`
	Roles    []Role `json:"roles,omitempty"`
	// Name and Role are legacy/derived fields kept for backward-compatible
	// display: they fall back to Username and the first role name.
	Name string `json:"name,omitempty"`
	Role string `json:"role,omitempty"`
}

// normalized returns a copy of the user with `Name`/`Role` filled from the
// modern `username`/`roles` fields when the legacy fields are empty.
func (u User) normalized() User {
	if u.Name == "" {
		u.Name = u.Username
	}
	if u.Role == "" && len(u.Roles) > 0 {
		u.Role = u.Roles[0].Name
	}
	return u
}

// Device identifies the connected device session.
type Device struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// SignInRequest is the payload for POST /api/auth/sign-in.
type SignInRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
	OTP      string `json:"otp,omitempty"`
}

// SignInResponse is the `data` returned by POST /api/auth/sign-in
// (SignInVm: `{user, token, device}`).
type SignInResponse struct {
	User          User           `json:"user"`
	Token         string         `json:"token"`
	Device        Device         `json:"device"`
	Organizations []Organization `json:"organizations,omitempty"`
}

// Organization is a compact organization entry exposed by sign-in.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// RefreshSessionRequest is the payload for POST /api/auth/sessions/refresh.
type RefreshSessionRequest struct {
	ClientID string `json:"clientId,omitempty"`
}

// RefreshSessionResponse is the `data` returned by POST /api/auth/sessions/refresh.
type RefreshSessionResponse struct {
	Token string `json:"token"`
}

// LogoutRequest is the payload for POST /api/auth/logout.
type LogoutRequest struct {
	ClientID string `json:"clientId,omitempty"`
}

// MFAFactor is a factor exposed by POST /api/mfa/challenge (MfaFactorVm).
type MFAFactor struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	Enabled    bool   `json:"enabled"`
	VerifiedAt string `json:"verifiedAt,omitempty"`
}

// ChallengeResponse is the `data` returned by POST /api/mfa/challenge
// (MfaChallengeVm: `{mfaRequired, challenge?, factors}`).
type ChallengeResponse struct {
	MFARequired bool        `json:"mfaRequired"`
	Challenge   string      `json:"challenge,omitempty"`
	Factors     []MFAFactor `json:"factors"`
}

// VerifyRequest is the payload for MFA verification endpoints.
type VerifyRequest struct {
	Code string `json:"code"`
}

// VerifyResponse is the `data` returned by POST /api/mfa/totp/verify and
// POST /api/mfa/recovery/verify (MfaVerifyVm: `{mfaVerified, mfaToken?}`).
type VerifyResponse struct {
	MFAVerified bool   `json:"mfaVerified"`
	MFAToken    string `json:"mfaToken,omitempty"`
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

// SignIn authenticates with email + password and returns the single session
// token (SignInVm `{user, token, device}`).
func (c *Connector) SignIn(ctx context.Context, req SignInRequest) (*SignInResponse, error) {
	var out SignInResponse
	if err := c.Client.Do(ctx, "POST", APISignInPage, req, &out); err != nil {
		return nil, err
	}
	out.User = out.User.normalized()
	return &out, nil
}

// SignOut invalidates the current token server-side.
func (c *Connector) SignOut(ctx context.Context, clientID string) error {
	return c.Client.Do(ctx, "POST", APILogoutPage, LogoutRequest{ClientID: clientID}, nil)
}

// RefreshSession rotates the current session token via POST /api/auth/sessions/refresh.
func (c *Connector) RefreshSession(ctx context.Context, clientID string) (*RefreshSessionResponse, error) {
	var out RefreshSessionResponse
	if err := c.Client.Do(ctx, "POST", APIRefreshPage, RefreshSessionRequest{ClientID: clientID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChallengeMFA requests the available MFA factors for the current session.
func (c *Connector) ChallengeMFA(ctx context.Context) (*ChallengeResponse, error) {
	var out ChallengeResponse
	if err := c.Client.Do(ctx, "POST", APIChallengePage, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyTOTP verifies a 6-digit TOTP code.
func (c *Connector) VerifyTOTP(ctx context.Context, code string) (*VerifyResponse, error) {
	var out VerifyResponse
	if err := c.Client.Do(ctx, "POST", APITOTPVerify, VerifyRequest{Code: code}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyRecovery verifies a backup/recovery code.
func (c *Connector) VerifyRecovery(ctx context.Context, code string) (*VerifyResponse, error) {
	var out VerifyResponse
	if err := c.Client.Do(ctx, "POST", APIRecoveryVerify, VerifyRequest{Code: code}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}