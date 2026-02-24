package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"golang.org/x/crypto/scrypt"
)

// --- generateSecurePassword tests ---

func TestGenerateSecurePassword_Length(t *testing.T) {
	pw, err := generateSecurePassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != 64 {
		t.Errorf("expected 64-char hex string, got %d chars", len(pw))
	}
}

func TestGenerateSecurePassword_ValidHex(t *testing.T) {
	pw, err := generateSecurePassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := hex.DecodeString(pw); err != nil {
		t.Errorf("password is not valid hex: %v", err)
	}
}

func TestGenerateSecurePassword_Unique(t *testing.T) {
	pw1, _ := generateSecurePassword()
	pw2, _ := generateSecurePassword()
	if pw1 == pw2 {
		t.Error("two generated passwords should not be identical")
	}
}

// --- serverEncryptPackJSON tests ---

func TestServerEncryptPackJSON_MagicHeader(t *testing.T) {
	plain := []byte(`{"test": "data"}`)
	encrypted, err := serverEncryptPackJSON(plain, "testpassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(encrypted), serverEncryptionMagic) {
		t.Error("encrypted data should start with QAPENC magic header")
	}
}

func TestServerEncryptPackJSON_MinimumLength(t *testing.T) {
	plain := []byte(`{"test": "data"}`)
	encrypted, err := serverEncryptPackJSON(plain, "testpassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// QAPENC(6) + salt(32) + nonce(12) + ciphertext(>=len(plain)+16 for GCM tag)
	minLen := 6 + 32 + 12 + len(plain) + 16
	if len(encrypted) < minLen {
		t.Errorf("encrypted data too short: got %d, want at least %d", len(encrypted), minLen)
	}
}

func TestServerEncryptPackJSON_DecryptRoundTrip(t *testing.T) {
	plain := []byte(`{"key": "value", "number": 42}`)
	password := "my-secure-password"

	encrypted, err := serverEncryptPackJSON(plain, password)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}

	// Manually decrypt using the same format the client uses
	decrypted, err := testDecryptData(encrypted, password)
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}

	if !bytes.Equal(plain, decrypted) {
		t.Errorf("round-trip mismatch:\n  got:  %s\n  want: %s", decrypted, plain)
	}
}

func TestServerEncryptPackJSON_EmptyInput(t *testing.T) {
	encrypted, err := serverEncryptPackJSON([]byte{}, "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(encrypted), serverEncryptionMagic) {
		t.Error("encrypted empty data should still have QAPENC header")
	}
	decrypted, err := testDecryptData(encrypted, "password")
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty decrypted data, got %d bytes", len(decrypted))
	}
}

// testDecryptData decrypts data in the QAPENC format (mirrors client-side decryptData).
func testDecryptData(data []byte, password string) ([]byte, error) {
	magicLen := len(serverEncryptionMagic)
	salt := data[magicLen : magicLen+serverSaltLen]

	key, err := scrypt.Key([]byte(password), salt, serverScryptN, serverScryptR, serverScryptP, serverScryptKeyLen)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceStart := magicLen + serverSaltLen
	nonceEnd := nonceStart + gcm.NonceSize()
	nonce := data[nonceStart:nonceEnd]
	ciphertext := data[nonceEnd:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// --- repackZipWithEncryptedData tests ---

// makeTestZip creates a ZIP with pack.json and optionally metadata.json.
func makeTestZip(t *testing.T, packData []byte, metadataData []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	pw, err := zw.Create("pack.json")
	if err != nil {
		t.Fatalf("create pack.json: %v", err)
	}
	pw.Write(packData)

	if metadataData != nil {
		mw, err := zw.Create("metadata.json")
		if err != nil {
			t.Fatalf("create metadata.json: %v", err)
		}
		mw.Write(metadataData)
	}

	zw.Close()
	return buf.Bytes()
}

func TestRepackZipWithEncryptedData_PackJsonReplaced(t *testing.T) {
	originalPack := []byte(`{"original": true}`)
	metadata := []byte(`{"name": "test-pack"}`)
	originalZip := makeTestZip(t, originalPack, metadata)

	encryptedPack := []byte("QAPENC-fake-encrypted-data")
	result, err := repackZipWithEncryptedData(originalZip, encryptedPack)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read pack.json from result ZIP
	zr, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("open result zip: %v", err)
	}

	for _, f := range zr.File {
		if f.Name == "pack.json" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			if !bytes.Equal(data, encryptedPack) {
				t.Errorf("pack.json content mismatch:\n  got:  %s\n  want: %s", data, encryptedPack)
			}
			return
		}
	}
	t.Error("pack.json not found in result ZIP")
}

func TestRepackZipWithEncryptedData_MetadataPreserved(t *testing.T) {
	originalPack := []byte(`{"original": true}`)
	metadata := []byte(`{"name": "test-pack", "author": "tester"}`)
	originalZip := makeTestZip(t, originalPack, metadata)

	encryptedPack := []byte("QAPENC-encrypted")
	result, err := repackZipWithEncryptedData(originalZip, encryptedPack)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("open result zip: %v", err)
	}

	for _, f := range zr.File {
		if f.Name == "metadata.json" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			if !bytes.Equal(data, metadata) {
				t.Errorf("metadata.json content mismatch:\n  got:  %s\n  want: %s", data, metadata)
			}
			return
		}
	}
	t.Error("metadata.json not found in result ZIP")
}

func TestRepackZipWithEncryptedData_NoMetadata(t *testing.T) {
	originalPack := []byte(`{"original": true}`)
	originalZip := makeTestZip(t, originalPack, nil)

	encryptedPack := []byte("QAPENC-encrypted")
	result, err := repackZipWithEncryptedData(originalZip, encryptedPack)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("open result zip: %v", err)
	}

	hasPackJSON := false
	hasMetadata := false
	for _, f := range zr.File {
		if f.Name == "pack.json" {
			hasPackJSON = true
		}
		if f.Name == "metadata.json" {
			hasMetadata = true
		}
	}
	if !hasPackJSON {
		t.Error("pack.json should exist in result ZIP")
	}
	if hasMetadata {
		t.Error("metadata.json should not exist when original had none")
	}
}

func TestRepackZipWithEncryptedData_EndToEnd(t *testing.T) {
	// Create a realistic QAP with JSON pack data
	packJSON := []byte(`{"file_type":"Vantagics_QuickAnalysisPack","metadata":{"author":"test"}}`)
	metaJSON, _ := json.Marshal(map[string]string{"author": "test"})
	originalZip := makeTestZip(t, packJSON, metaJSON)

	// Encrypt the pack data
	password := "test-password-123"
	encrypted, err := serverEncryptPackJSON(packJSON, password)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}

	// Repack
	result, err := repackZipWithEncryptedData(originalZip, encrypted)
	if err != nil {
		t.Fatalf("repack error: %v", err)
	}

	// Verify the repacked ZIP's pack.json is encrypted and decryptable
	zr, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("open result zip: %v", err)
	}

	for _, f := range zr.File {
		if f.Name == "pack.json" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()

			if !strings.HasPrefix(string(data), serverEncryptionMagic) {
				t.Fatal("repacked pack.json should be encrypted (QAPENC header)")
			}

			decrypted, err := testDecryptData(data, password)
			if err != nil {
				t.Fatalf("decrypt repacked data: %v", err)
			}
			if !bytes.Equal(decrypted, packJSON) {
				t.Errorf("decrypted content mismatch:\n  got:  %s\n  want: %s", decrypted, packJSON)
			}
			return
		}
	}
	t.Error("pack.json not found in result ZIP")
}
