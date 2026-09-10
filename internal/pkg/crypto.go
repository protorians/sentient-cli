package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

// EncryptAESGCM encrypts plaintext with AES-256-GCM using a key derived from
// the given passphrase (SHA-256). Returns a random nonce prepended to the
// ciphertext.
func EncryptAESGCM(passphrase string, plaintext []byte) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("initialisation du chiffrement impossible : %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialisation du GCM impossible : %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("génération du nonce impossible : %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptAESGCM decrypts ciphertext produced by EncryptAESGCM.
func DecryptAESGCM(passphrase string, ciphertext []byte) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("initialisation du chiffrement impossible : %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialisation du GCM impossible : %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("données chiffrées invalides")
	}
	nonce, payload := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, errors.New("déchiffrement impossible (accès refusé)")
	}
	return plaintext, nil
}
