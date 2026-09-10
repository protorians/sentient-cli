package pkg

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	secret := "passphrase-secrete"
	plain := []byte("données sensibles à chiffrer")

	ciphertext, err := EncryptAESGCM(secret, plain)
	if err != nil {
		t.Fatalf("EncryptAESGCM: %v", err)
	}
	if bytes.Equal(ciphertext, plain) {
		t.Error("le chiffré ne doit pas être identique au clair")
	}

	decoded, err := DecryptAESGCM(secret, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAESGCM: %v", err)
	}
	if !bytes.Equal(decoded, plain) {
		t.Errorf("round-trip: got %q want %q", decoded, plain)
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	ciphertext, err := EncryptAESGCM("bonne-passe", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptAESGCM("mauvaise-passe", ciphertext); err == nil {
		t.Error("déchiffrement avec un mauvais mot de passe doit échouer")
	}
}

func TestNewUUID(t *testing.T) {
	a := NewUUID()
	b := NewUUID()
	if a == b {
		t.Error("deux UUID générés doivent différer")
	}
	if !IsUUID(a) || !IsUUID(b) {
		t.Error("UUID générés doivent être valides")
	}
}
