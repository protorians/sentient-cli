package signing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// GenerateKeyPair generates a new Ed25519 key pair.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("génération de la paire de clés impossible : %w", err)
	}
	return pub, priv, nil
}

// Fingerprint returns the SHA-256 fingerprint of a public key (hex-encoded).
func Fingerprint(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return hex.EncodeToString(hash[:])
}

// SignArchive signs an .smp archive file and writes the signature to a .sig file.
func SignArchive(archivePath string, privKey ed25519.PrivateKey) (string, error) {
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return "", fmt.Errorf("lecture de l'archive impossible : %w", err)
	}

	sig := ed25519.Sign(privKey, data)

	sigPath := archivePath + ".sig"
	if err := os.WriteFile(sigPath, sig, 0o600); err != nil {
		return "", fmt.Errorf("écriture de la signature impossible : %w", err)
	}

	return sigPath, nil
}

// VerifySignature verifies a .sig file against an .smp archive.
func VerifySignature(archivePath, sigPath string, pubKey ed25519.PublicKey) (bool, error) {
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		return false, fmt.Errorf("lecture de l'archive impossible : %w", err)
	}

	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return false, fmt.Errorf("lecture de la signature impossible : %w", err)
	}

	return ed25519.Verify(pubKey, archiveData, sigData), nil
}

// LoadKeyPair loads a public and private key from the key store.
func LoadKeyPair(store KeyStore) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privBytes, err := store.GetPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("clé privée introuvable : %w", err)
	}
	pubBytes, err := store.GetPublicKey()
	if err != nil {
		return nil, nil, fmt.Errorf("clé publique introuvable : %w", err)
	}

	privKey := ed25519.PrivateKey(privBytes)
	pubKey := ed25519.PublicKey(pubBytes)

	if !pubKey.Equal(privKey.Public()) {
		return nil, nil, fmt.Errorf("incohérence entre la clé publique et la clé privée")
	}

	return pubKey, privKey, nil
}

// SaveKeyPair saves a public and private key to the key store.
func SaveKeyPair(store KeyStore, pub ed25519.PublicKey, priv ed25519.PrivateKey) error {
	if err := store.SetPublicKey(pub); err != nil {
		return fmt.Errorf("sauvegarde de la clé publique impossible : %w", err)
	}
	if err := store.SetPrivateKey(priv); err != nil {
		return fmt.Errorf("sauvegarde de la clé privée impossible : %w", err)
	}
	return nil
}

// FindArchive finds the .smp archive for a module in .sentients/build/.
func FindArchive(root, moduleName, version string) (string, error) {
	buildDir := config.BuildDir(root)
	path := filepath.Join(buildDir, moduleName+"-"+version+".smp")
	if !pkg.FileExists(path) {
		return "", fmt.Errorf("archive %s introuvable dans %s", moduleName+"-"+version+".smp", buildDir)
	}
	return path, nil
}
