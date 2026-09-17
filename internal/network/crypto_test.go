package network

import (
	"bytes"
	"testing"
)

func TestCryptoEncryptDecrypt(t *testing.T) {
	secret := "光明顶禁地密道"
	key := DeriveRoomKey(secret)
	plaintext := []byte("乾坤大挪移心法第七层")

	// 1. Normal encrypt and decrypt
	ciphertext, err := EncryptPayload(key, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := DecryptPayload(key, ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted does not match plaintext: got %s, want %s", string(decrypted), string(plaintext))
	}

	// 2. Wrong key decryption should fail
	wrongKey := DeriveRoomKey("黑木崖后花园")
	_, err = DecryptPayload(wrongKey, ciphertext)
	if err == nil {
		t.Fatalf("expected error when decrypting with wrong key, but got nil")
	}

	// 3. No key (plaintext mode)
	plainEnc, err := EncryptPayload(nil, plaintext)
	if err != nil {
		t.Fatalf("nil key encrypt failed: %v", err)
	}
	plainDec, err := DecryptPayload(nil, plainEnc)
	if err != nil {
		t.Fatalf("nil key decrypt failed: %v", err)
	}
	if !bytes.Equal(plaintext, plainDec) {
		t.Fatalf("plain mode mismatch")
	}
}

func TestAdminTokenHash(t *testing.T) {
	h1 := DeriveAdminTokenHash("master-secret-123")
	h2 := DeriveAdminTokenHash("master-secret-123")
	h3 := DeriveAdminTokenHash("wrong-secret")

	if h1 != h2 {
		t.Fatalf("identical secrets should yield identical hashes")
	}
	if h1 == h3 {
		t.Fatalf("different secrets should yield different hashes")
	}
}
