package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/protorians/sentient-cli/internal/pkg"
)

func TestSessionSaveLoadClear(t *testing.T) {
	store := NewStoreVolatile()
	sess := &Session{
		Store:        store,
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		User:         &User{ID: "u1", Email: "dev@example.com", Role: "Developer"},
	}
	exp := time.Now().Add(time.Hour)
	sess.ExpiresAt = &exp

	if err := sess.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadSession(store)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if loaded == nil || !loaded.IsAuthenticated() {
		t.Fatal("session chargée non authentifiée")
	}
	if loaded.AccessToken != "access-123" || loaded.RefreshToken != "refresh-456" {
		t.Errorf("tokens incorrects: %+v", loaded)
	}
	if loaded.User == nil || loaded.User.Email != "dev@example.com" {
		t.Errorf("user incorrect: %+v", loaded.User)
	}
	if loaded.IsExpired() {
		t.Error("session fraîche doit ne pas être expirée")
	}

	if err := sess.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	after, err := LoadSession(store)
	if err != nil {
		t.Fatalf("LoadSession après Clear: %v", err)
	}
	if after != nil && after.IsAuthenticated() {
		t.Error("les credentials doivent être supprimées après Clear")
	}
}

func TestSessionExpired(t *testing.T) {
	store := NewStoreVolatile()
	past := time.Now().Add(-time.Minute)
	sess := &Session{
		Store:       store,
		AccessToken: "token",
		ExpiresAt:   &past,
	}
	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, _ := LoadSession(store)
	if !loaded.IsExpired() {
		t.Error("token dans le passé doit être marqué expiré")
	}
}

func TestConnectorSignIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/sign-in" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"tok","refresh_token":"ref","expires_in":3600,"user":{"id":"u1","email":"dev@example.com","name":"Dev","role":"Developer"},"mfaRequired":false}`))
	}))
	defer server.Close()

	conn := &Connector{Client: pkg.NewClient(server.URL)}
	resp, err := conn.SignIn(t.Context(), SignInRequest{Email: "dev@example.com", Password: "secret"})
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if resp.AccessToken != "tok" {
		t.Errorf("AccessToken = %q, want tok", resp.AccessToken)
	}
	if resp.User.Email != "dev@example.com" {
		t.Errorf("User.Email = %q", resp.User.Email)
	}
	if resp.MFARequired {
		t.Error("MFARequired doit être false")
	}
}

func TestConnectorSignInAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"INVALID_CREDENTIALS","message":"Identifiants invalides"}`))
	}))
	defer server.Close()

	conn := &Connector{Client: pkg.NewClient(server.URL)}
	_, err := conn.SignIn(t.Context(), SignInRequest{Email: "x", Password: "y"})
	if err == nil {
		t.Fatal("SignIn doit échouer")
	}
	apiErr, ok := err.(*pkg.APIError)
	if !ok {
		t.Fatalf("erreur = %T, want *pkg.APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Message != "Identifiants invalides" {
		t.Errorf("Message = %q, want Identifiants invalides", apiErr.Message)
	}
}
