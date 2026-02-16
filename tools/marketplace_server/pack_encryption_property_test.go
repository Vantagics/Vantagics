package main

import (
	"bytes"
	"math/rand"
	"testing"
	"testing/quick"
	"time"
)

// Feature: paid-pack-encryption, Property 1: 加密解密往返一致性
// **Validates: Requirements 1.3, 6.3**
//
// For any valid byte slice (representing JSON data) and any non-empty password,
// encrypting with serverEncryptPackJSON and then decrypting with testDecryptData
// should yield the original data.
func TestProperty1_EncryptDecryptRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate random plaintext data (0 to 4096 bytes)
		dataLen := r.Intn(4097)
		plainData := make([]byte, dataLen)
		for i := range plainData {
			plainData[i] = byte(r.Intn(256))
		}

		// Generate a random non-empty password (1 to 64 ASCII printable chars)
		pwLen := r.Intn(64) + 1
		pwBytes := make([]byte, pwLen)
		for i := range pwBytes {
			pwBytes[i] = byte(r.Intn(94) + 33) // ASCII 33-126
		}
		password := string(pwBytes)

		// Encrypt
		encrypted, err := serverEncryptPackJSON(plainData, password)
		if err != nil {
			t.Logf("seed=%d: encrypt failed: %v", seed, err)
			return false
		}

		// Decrypt using the test helper (mirrors client-side decryption)
		decrypted, err := testDecryptData(encrypted, password)
		if err != nil {
			t.Logf("seed=%d: decrypt failed: %v", seed, err)
			return false
		}

		// Property: decrypted data must equal original plaintext
		if !bytes.Equal(plainData, decrypted) {
			t.Logf("seed=%d: round-trip mismatch: original len=%d, decrypted len=%d",
				seed, len(plainData), len(decrypted))
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 1 (加密解密往返一致性) failed: %v", err)
	}
}
