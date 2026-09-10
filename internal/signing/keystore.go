package signing

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/zalando/go-keyring"
)

// Keychain keys for signing key storage.
const (
	signingServiceName = "sentient-cli-signing"
	keyPublicKey       = "signing_public_key"
	keyPrivateKey      = "signing_private_key"
)

// KeyStore is a secure storage for Ed25519 signing keys.
type KeyStore interface {
	GetPublicKey() ([]byte, error)
	SetPublicKey(data []byte) error
	GetPrivateKey() ([]byte, error)
	SetPrivateKey(data []byte) error
	DeleteKeys() error
	HasKeys() bool
}

// keyringKeyStore uses the OS keychain via go-keyring.
type keyringKeyStore struct{}

// fallbackKeyStore persists keys in an AES-256-GCM encrypted file.
type fallbackKeyStore struct {
	path string
	key  string
	mu   sync.Mutex
}

// NewKeyStore returns the appropriate key store for the platform.
func NewKeyStore() KeyStore {
	return &keyringKeyStore{}
}

// NewKeyStoreVolatile returns a store forced to use the fallback backend.
func NewKeyStoreVolatile() KeyStore {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return &fallbackKeyStore{
		path: filepath.Join(home, ".sentient-cli", "signing.enc"),
		key:  "sentient-cli-signing-v1",
	}
}

// --- keyringKeyStore ---

func (s *keyringKeyStore) GetPublicKey() ([]byte, error) {
	v, err := keyring.Get(signingServiceName, keyPublicKey)
	if err != nil {
		return nil, fmt.Errorf("clé publique absente du keychain : %w", err)
	}
	return []byte(v), nil
}

func (s *keyringKeyStore) SetPublicKey(data []byte) error {
	return keyring.Set(signingServiceName, keyPublicKey, string(data))
}

func (s *keyringKeyStore) GetPrivateKey() ([]byte, error) {
	v, err := keyring.Get(signingServiceName, keyPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("clé privée absente du keychain : %w", err)
	}
	return []byte(v), nil
}

func (s *keyringKeyStore) SetPrivateKey(data []byte) error {
	return keyring.Set(signingServiceName, keyPrivateKey, string(data))
}

func (s *keyringKeyStore) DeleteKeys() error {
	if err := keyring.Delete(signingServiceName, keyPublicKey); err != nil && !isKeyringNotExist(err) {
		return err
	}
	if err := keyring.Delete(signingServiceName, keyPrivateKey); err != nil && !isKeyringNotExist(err) {
		return err
	}
	return nil
}

func (s *keyringKeyStore) HasKeys() bool {
	_, err := keyring.Get(signingServiceName, keyPublicKey)
	return err == nil
}

// --- fallbackKeyStore ---

func (s *fallbackKeyStore) GetPublicKey() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	v, ok := data[keyPublicKey]
	if !ok {
		return nil, fmt.Errorf("clé publique absente")
	}
	return decodeValue(v)
}

func (s *fallbackKeyStore) SetPublicKey(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, err := s.load()
	if err != nil {
		return err
	}
	d[keyPublicKey] = encodeValue(data)
	return s.save(d)
}

func (s *fallbackKeyStore) GetPrivateKey() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	v, ok := data[keyPrivateKey]
	if !ok {
		return nil, fmt.Errorf("clé privée absente")
	}
	return decodeValue(v)
}

func (s *fallbackKeyStore) SetPrivateKey(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, err := s.load()
	if err != nil {
		return err
	}
	d[keyPrivateKey] = encodeValue(data)
	return s.save(d)
}

func (s *fallbackKeyStore) DeleteKeys() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("suppression du fichier de clés impossible : %w", err)
	}
	return nil
}

func (s *fallbackKeyStore) HasKeys() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return false
	}
	_, ok := data[keyPublicKey]
	return ok
}

func (s *fallbackKeyStore) load() (map[string]string, error) {
	if !pkg.FileExists(s.path) {
		return map[string]string{}, nil
	}
	ciphertext, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("lecture du fichier de clés impossible : %w", err)
	}
	plain, err := pkg.DecryptAESGCM(s.key, ciphertext)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, fmt.Errorf("décodage du fichier de clés impossible : %w", err)
	}
	return out, nil
}

func (s *fallbackKeyStore) save(data map[string]string) error {
	plain, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sérialisation des clés impossible : %w", err)
	}
	ciphertext, err := pkg.EncryptAESGCM(s.key, plain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("création du dossier de clés impossible : %w", err)
	}
	return os.WriteFile(s.path, ciphertext, 0o600)
}

// encodeValue base64-encodes binary key material so it survives a JSON round
// trip (JSON would otherwise corrupt non-UTF-8 bytes).
func encodeValue(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// decodeValue base64-decodes a value previously stored via encodeValue.
func decodeValue(v string) ([]byte, error) {
	out, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("décodage de la clé stockée impossible : %w", err)
	}
	return out, nil
}

func isKeyringNotExist(err error) bool {
	return err == keyring.ErrNotFound
}
