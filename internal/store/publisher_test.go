package store

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/protorians/sentient-cli/internal/auth"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
)

func TestListModules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store/modules" || r.Method != http.MethodGet {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"token":"t1","name":"Mod A","description":"desc","version":"0.1.0","status":"published"},{"token":"t2","name":"Mod B","description":"","version":"0.2.0","status":"draft"}]`))
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	client.SetToken("test-token")

	modules, err := client.ListModules(t.Context())
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(modules) != 2 {
		t.Fatalf("attendu 2 modules, reçu %d", len(modules))
	}
	if modules[0].Token != "t1" || modules[0].Name != "Mod A" {
		t.Errorf("premier module incorrect: %+v", modules[0])
	}
	if modules[1].Version != "0.2.0" {
		t.Errorf("deuxième module version = %q, want 0.2.0", modules[1].Version)
	}
}

func TestListModulesAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"UNAUTHORIZED","message":"Non authentifié"}`))
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	_, err := client.ListModules(t.Context())
	if err == nil {
		t.Error("ListModules doit échouer sans token valide")
	}
}

func TestGetModule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store/modules/tok-abc" || r.Method != http.MethodGet {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"tok-abc","name":"Blog","description":"A blog","version":"1.0.0","status":"published","publisher":{"id":"dev1","name":"Dev"}}`))
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	mod, err := client.GetModule(t.Context(), "tok-abc")
	if err != nil {
		t.Fatalf("GetModule: %v", err)
	}
	if mod.Token != "tok-abc" {
		t.Errorf("Token = %q, want tok-abc", mod.Token)
	}
	if mod.Publisher.ID != "dev1" {
		t.Errorf("Publisher.ID = %q, want dev1", mod.Publisher.ID)
	}
}

func TestGetModuleNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"Module introuvable"}`))
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	_, err := client.GetModule(t.Context(), "missing")
	if err == nil {
		t.Error("GetModule doit échouer pour un module absent")
	}
}

func TestPublish(t *testing.T) {
	// Set up a temporary archive file
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test-module-0.1.0.smp")
	if err := os.WriteFile(archivePath, []byte("fake-zip-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store/modules/publish" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// Verify authorization header
		if r.Header.Get("Authorization") != "Bearer test-pub-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Verify content type is multipart
		ct := r.Header.Get("Content-Type")
		if ct == "" || len(ct) < 19 || ct[:19] != "multipart/form-data" {
			http.Error(w, "bad content type: "+ct, http.StatusBadRequest)
			return
		}
		// Parse multipart
		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "no multipart: "+err.Error(), http.StatusBadRequest)
			return
		}
		part, err := reader.NextPart()
		if err != nil {
			http.Error(w, "no first part: "+err.Error(), http.StatusBadRequest)
			return
		}
		if part.FormName() != "archive" {
			http.Error(w, "first part name = "+part.FormName()+", want archive", http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(part)
		if string(data) != "fake-zip-content" {
			http.Error(w, "archive content mismatch", http.StatusBadRequest)
			return
		}
		// Check manifest part
		part2, err := reader.NextPart()
		if err != nil {
			http.Error(w, "no manifest part", http.StatusBadRequest)
			return
		}
		if part2.FormName() != "manifest" {
			http.Error(w, "second part name = "+part2.FormName()+", want manifest", http.StatusBadRequest)
			return
		}
		manifestData, _ := io.ReadAll(part2)
		var m module.Manifest
		if json.Unmarshal(manifestData, &m) != nil {
			http.Error(w, "invalid manifest JSON", http.StatusBadRequest)
			return
		}
		if m.ID != "test-module" {
			http.Error(w, "manifest id = "+m.ID, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"tok-new","version":"0.1.0","url":"https://store.sentient.dev/modules/test-module"}`))
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	client.SetToken("test-pub-token")

	manifest := module.NewManifest("test-module", "A test")
	resp, err := client.Publish(t.Context(), archivePath, &manifest)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if resp.Token != "tok-new" {
		t.Errorf("Token = %q, want tok-new", resp.Token)
	}
	if resp.URL != "https://store.sentient.dev/modules/test-module" {
		t.Errorf("URL = %q, want store URL", resp.URL)
	}
}

func TestPublishMissingArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not reach", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	manifest := module.NewManifest("mod", "")
	_, err := client.Publish(t.Context(), "/nonexistent/archive.smp", &manifest)
	if err == nil {
		t.Error("Publish doit échouer avec un fichier inexistant")
	}
}

func TestPublishServerError(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "mod.smp")
	if err := os.WriteFile(archivePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"VERSION_CONFLICT","message":"Version déjà publiée"}`))
	}))
	defer server.Close()

	client := &Client{Connector: &auth.Connector{Client: pkg.NewClient(server.URL)}}
	client.SetToken("tok")
	manifest := module.NewManifest("mod", "")
	_, err := client.Publish(t.Context(), archivePath, &manifest)
	if err == nil {
		t.Error("Publish doit échouer sur conflit de version")
	}
	apiErr, ok := err.(*pkg.APIError)
	if !ok {
		t.Fatalf("erreur = %T, want *pkg.APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %d, want 409", apiErr.StatusCode)
	}
}

func TestSetToken(t *testing.T) {
	client := NewClient()
	client.SetToken("my-token")
	if client.Connector.Client.Token != "my-token" {
		t.Errorf("Token = %q, want my-token", client.Connector.Client.Token)
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.Connector == nil {
		t.Error("Connector doit être initialisé")
	}
}
