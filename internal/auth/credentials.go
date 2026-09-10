package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/zalando/go-keyring"
)

// Keychain keys storing credentials.
const (
	ServiceName = "sentient-cli"
	prefix      = "sentient-cli."

	KeyAccessToken  = prefix + "access_token"
	KeyRefreshToken = prefix + "refresh_token"
	KeyExpiresAt    = prefix + "expires_at"
	KeyUserID       = prefix + "user_id"
	KeyUserEmail    = prefix + "user_email"
	KeyMFASecret    = prefix + "mfa_secret"
)

// AllKeys lists every credential key used by the CLI.
var AllKeys = []string{
	KeyAccessToken,
	KeyRefreshToken,
	KeyExpiresAt,
	KeyUserID,
	KeyUserEmail,
	KeyMFASecret,
}

// Store is a secure credential storage.
type Store interface {
	// Get returns the value stored under key.
	Get(key string) (string, error)
	// Set stores value under key.
	Set(key string, value string) error
	// Delete removes a single key.
	Delete(kind string) error
	// DeleteAll removes every sentient-cli credential.
	DeleteAll() error
}

// keyringStore uses the operating system keychain via go-keyring.
type keyringStore struct{}

// fallbackStore persists credentials in an AES-256-GCM encrypted file, used
// when no system keychain is available (see risk R-002).
type fallbackStore struct {
	path string
	key  string
	mu   sync.Mutex
}

// NewStore returns the appropriate credential store for the platform.
// When the system keychain is unavailable, it falls back to an encrypted
// file under the user home directory.
func NewStore() Store {
	return &keyringStore{}
}

// NewStoreVolatile returns a store forced to use the in-memory/fallback
// backend regardless of keychain availability (useful when running without
// a system keyring, e.g. headless or tests).
func NewStoreVolatile() Store {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return &fallbackStore{
		path: filepath.Join(home, ".sentient-cli", "credentials.enc"),
		key:  "sentient-cli-fallback-v1",
	}
}

func (s *keyringStore) Get(key string) (string, error) {
	v, err := keyring.Get(ServiceName, key)
	if err != nil {
		return "", fmt.Errorf("lecture de la credential %q impossible : %w", key, err)
	}
	return v, nil
}

func (s *keyringStore) Set(key, value string) error {
	if value == "" {
		return nil
	}
	if err := keyring.Set(ServiceName, key, value); err != nil {
		return fmt.Errorf("stockage de la credential %q impossible : %w", key, err)
	}
	return nil
}

func (s *keyringStore) Delete(kind string) error {
	if err := keyring.Delete(ServiceName, kind); err != nil {
		if isKeyringNotExist(err) {
			return nil
		}
		return fmt.Errorf("suppression de la credential %q impossible : %w", kind, err)
	}
	return nil
}

func (s *keyringStore) DeleteAll() error {
	for _, k := range AllKeys {
		if err := s.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

func (s *fallbackStore) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return "", err
	}
	if v, ok := data[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("credential %q absente", key)
}

func (s *fallbackStore) Set(key, value string) error {
	if value == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return err
	}
	data[key] = value
	return s.save(data)
}

func (s *fallbackStore) Delete(kind string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return err
	}
	delete(data, kind)
	return s.save(data)
}

func (s *fallbackStore) DeleteAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("suppression du fichier de credentials impossible : %w", err)
	}
	return nil
}

func (s *fallbackStore) load() (map[string]string, error) {
	if !pkg.FileExists(s.path) {
		return map[string]string{}, nil
	}
	ciphertext, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("lecture du fichier de credentials impossible : %w", err)
	}
	plain, err := pkg.DecryptAESGCM(s.key, ciphertext)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, fmt.Errorf("décodage du fichier de credentials impossible : %w", err)
	}
	return out, nil
}

func (s *fallbackStore) save(data map[string]string) error {
	plain, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sérialisation des credentials impossible : %w", err)
	}
	ciphertext, err := pkg.EncryptAESGCM(s.key, plain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("création du dossier de credentials impossible : %w", err)
	}
	return os.WriteFile(s.path, ciphertext, 0o600)
}

func isKeyringNotExist(err error) bool {
	// go-keyring returns ErrNotFound for missing entries.
	return err == keyring.ErrNotFound
}
