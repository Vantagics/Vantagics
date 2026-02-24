package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackToZipAndUnpackRoundTrip_NoPassword(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.qap")
	original := []byte(`{"file_type":"Vantagics_QuickAnalysisPack","format_version":"1.0"}`)

	if err := PackToZip(original, zipPath, ""); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	got, err := UnpackFromZip(zipPath, "")
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	if string(got) != string(original) {
		t.Errorf("round-trip mismatch:\n  got:  %s\n  want: %s", got, original)
	}
}

func TestPackToZipAndUnpackRoundTrip_WithPassword(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test_enc.qap")
	original := []byte(`{"file_type":"Vantagics_QuickAnalysisPack","steps":[1,2,3]}`)
	password := "s3cret!"

	if err := PackToZip(original, zipPath, password); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	got, err := UnpackFromZip(zipPath, password)
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	if string(got) != string(original) {
		t.Errorf("round-trip mismatch:\n  got:  %s\n  want: %s", got, original)
	}
}

func TestUnpackFromZip_WrongPassword(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test_wrong.qap")
	original := []byte(`{"data":"secret"}`)

	if err := PackToZip(original, zipPath, "correct"); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	_, err := UnpackFromZip(zipPath, "wrong")
	if err != ErrWrongPassword {
		t.Errorf("expected ErrWrongPassword, got: %v", err)
	}
}

func TestUnpackFromZip_EncryptedNoPassword(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test_nopw.qap")
	original := []byte(`{"data":"secret"}`)

	if err := PackToZip(original, zipPath, "mypass"); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	_, err := UnpackFromZip(zipPath, "")
	if err != ErrPasswordRequired {
		t.Errorf("expected ErrPasswordRequired, got: %v", err)
	}
}

func TestIsEncrypted_Encrypted(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "enc.qap")

	if err := PackToZip([]byte(`{}`), zipPath, "pass"); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	encrypted, err := IsEncrypted(zipPath)
	if err != nil {
		t.Fatalf("IsEncrypted: %v", err)
	}
	if !encrypted {
		t.Error("expected IsEncrypted=true for encrypted file")
	}
}

func TestIsEncrypted_NotEncrypted(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "plain.qap")

	if err := PackToZip([]byte(`{}`), zipPath, ""); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	encrypted, err := IsEncrypted(zipPath)
	if err != nil {
		t.Fatalf("IsEncrypted: %v", err)
	}
	if encrypted {
		t.Error("expected IsEncrypted=false for unencrypted file")
	}
}

func TestIsEncrypted_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.qap")
	os.WriteFile(badPath, []byte("not a zip"), 0644)

	_, err := IsEncrypted(badPath)
	if err == nil {
		t.Error("expected error for invalid ZIP file")
	}
}

func TestUnpackFromZip_EmptyData(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.qap")
	original := []byte(``)

	if err := PackToZip(original, zipPath, ""); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	got, err := UnpackFromZip(zipPath, "")
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	if string(got) != string(original) {
		t.Errorf("expected empty data, got: %s", got)
	}
}

func TestPackToZipAndUnpackRoundTrip_LargeData(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "large.qap")

	// Generate ~100KB of data
	large := make([]byte, 100*1024)
	for i := range large {
		large[i] = byte('A' + (i % 26))
	}

	if err := PackToZip(large, zipPath, "bigpass"); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	got, err := UnpackFromZip(zipPath, "bigpass")
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	if len(got) != len(large) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(large))
	}
	for i := range large {
		if got[i] != large[i] {
			t.Fatalf("byte mismatch at index %d", i)
		}
	}
}

func TestUnpackFromZip_PasswordOnUnencryptedIsOK(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "plain.qap")
	original := []byte(`{"ok":true}`)

	if err := PackToZip(original, zipPath, ""); err != nil {
		t.Fatalf("PackToZip: %v", err)
	}

	// Providing a password for an unencrypted file should still work
	got, err := UnpackFromZip(zipPath, "unnecessary_password")
	if err != nil {
		t.Fatalf("UnpackFromZip: %v", err)
	}

	if string(got) != string(original) {
		t.Errorf("round-trip mismatch:\n  got:  %s\n  want: %s", got, original)
	}
}
