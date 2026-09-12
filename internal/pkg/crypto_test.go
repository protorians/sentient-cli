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

func TestDeriveKey(t *testing.T) {
	salt := []byte("sel-de-test")
	k1, err := DeriveKey("passphrase", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	k2, err := DeriveKey("passphrase", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	if len(k1) != 32 {
		t.Fatalf("clé dérivée de %d octets, attendu 32", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatal("la dérivation doit être déterministe à sel égal")
		}
	}
	// Une passphrase différente doit produire une clé différente.
	k3, _ := DeriveKey("autre", salt)
	if bytes.Equal(k1, k3) {
		t.Error("deux passphrases différentes ne doivent pas produire la même clé")
	}
	if _, err := DeriveKey("x", nil); err == nil {
		t.Error("DeriveKey sans sel doit échouer")
	}
}

func TestEncryptVaultRoundTrip(t *testing.T) {
	secret, err := NewRandomKey()
	if err != nil {
		t.Fatalf("NewRandomKey: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("clé aléatoire de %d octets, attendu 32", len(secret))
	}

	plain := []byte("credentials sensibles")
	vault, err := EncryptVault(secret, plain)
	if err != nil {
		t.Fatalf("EncryptVault: %v", err)
	}
	if bytes.Equal(vault, plain) {
		t.Error("le coffre ne doit pas être identique au clair")
	}

	decoded, err := DecryptVault(secret, vault)
	if err != nil {
		t.Fatalf("DecryptVault: %v", err)
	}
	if !bytes.Equal(decoded, plain) {
		t.Errorf("round-trip: got %q want %q", decoded, plain)
	}
}

func TestDecryptVaultWrongSecret(t *testing.T) {
	secret, _ := NewRandomKey()
	other, _ := NewRandomKey()
	vault, err := EncryptVault(secret, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptVault(other, vault); err == nil {
		t.Error("déchiffrement du coffre avec un autre secret doit échouer")
	}
	if _, err := DecryptVault(secret, vault[:3]); err == nil {
		t.Error("coffre tronqué doit échouer")
	}
}

func TestEncryptVaultUniqueSalt(t *testing.T) {
	secret, _ := NewRandomKey()
	a, _ := EncryptVault(secret, []byte("même clair"))
	b, _ := EncryptVault(secret, []byte("même clair"))
	if bytes.Equal(a, b) {
		t.Error("deux chiffrements du même clair doivent différer (sel aléatoire)")
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
