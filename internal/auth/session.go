package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// Session represents a stored authenticated session.
type Session struct {
	Store        Store
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	User         *User
}

// Combined auth+session flow for interactive connectors keeps the two terms
// distinct: sign-in performs the API exchange; the session then persists it.

// LoadSession reads credentials from the store, if present.
func LoadSession(store Store) (*Session, error) {
	s := &Session{Store: store}

	access, err := store.Get(KeyAccessToken)
	if err != nil {
		return nil, nil // not authenticated
	}
	s.AccessToken = access

	s.RefreshToken, _ = store.Get(KeyRefreshToken)

	if raw, err := store.Get(KeyExpiresAt); err == nil {
		if ts, perr := strconv.ParseInt(raw, 10, 64); perr == nil {
			t := time.Unix(ts, 0)
			s.ExpiresAt = &t
		}
	}

	email, _ := store.Get(KeyUserEmail)
	id, _ := store.Get(KeyUserID)
	s.User = &User{Email: email, ID: id}

	return s, nil
}

// IsAuthenticated reports whether an access token is present.
func (s *Session) IsAuthenticated() bool {
	return s != nil && s.AccessToken != ""
}

// IsExpired reports whether the access token has passed its expiry.
func (s *Session) IsExpired() bool {
	return s != nil && s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

// Save persists the tokens and user info into the store.
func (s *Session) Save() error {
	if s.User == nil {
		s.User = &User{}
	}
	calls := []struct {
		key   string
		value string
	}{
		{KeyAccessToken, s.AccessToken},
		{KeyRefreshToken, s.RefreshToken},
		{KeyUserEmail, s.User.Email},
		{KeyUserID, s.User.ID},
	}
	if s.ExpiresAt != nil {
		calls = append(calls, struct {
			key   string
			value string
		}{KeyExpiresAt, strconv.FormatInt(s.ExpiresAt.Unix(), 10)})
	}
	for _, c := range calls {
		if err := s.Store.Set(c.key, c.value); err != nil {
			return err
		}
	}
	return nil
}

// Refresh renews the access token using the stored refresh token.
func (s *Session) Refresh(ctx context.Context, connector *Connector) error {
	if s.RefreshToken == "" {
		return fmt.Errorf("aucun refresh token disponible — exécutez 'sentient connect'")
	}
	resp, err := connector.Refresh(ctx, s.RefreshToken)
	if err != nil {
		return fmt.Errorf("rafraîchissement du token impossible : %w", err)
	}
	s.AccessToken = resp.AccessToken
	s.RefreshToken = resp.RefreshToken
	if resp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
		s.ExpiresAt = &t
	}
	if resp.User.ID != "" {
		s.User = &resp.User
	}
	return s.Save()
}

// Clear removes every stored credential.
func (s *Session) Clear() error {
	return s.Store.DeleteAll()
}

// ValidToken returns a non-expired access token, refreshing when needed.
func (s *Session) ValidToken(ctx context.Context, connector *Connector) (string, error) {
	if !s.IsAuthenticated() {
		return "", fmt.Errorf("non authentifié — exécutez 'sentient connect'")
	}
	if s.IsExpired() {
		if err := s.Refresh(ctx, connector); err != nil {
			return "", err
		}
	}
	return s.AccessToken, nil
}
