package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/scrypt"
)

const (
	packJSONFileName     = "pack.json"
	metadataJSONFileName = "metadata.json"
	encryptionMagic      = "QAPENC" // 6-byte magic header indicating encrypted content
	scryptN              = 32768
	scryptR              = 8
	scryptP              = 1
	scryptKeyLen         = 32 // AES-256
	saltLen              = 32
)

var (
	ErrInvalidZip       = errors.New("invalid ZIP file or missing pack.json")
	ErrWrongPassword    = errors.New("incorrect password")
	ErrPasswordRequired = errors.New("file is encrypted, password required")
)

// deriveKey derives a 32-byte AES-256 key from a password and salt using scrypt.
func deriveKey(password string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, scryptKeyLen)
}

// PackToZip writes jsonData into a ZIP file at outputPath with an internal entry named "pack.json".
// If password is non-empty, the data is encrypted using AES-256-GCM before being stored.
// It also extracts and stores metadata unencrypted as "metadata.json" for display in lists.
func PackToZip(jsonData []byte, outputPath string, password string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// 1. Create pack.json (encrypted if password set)
	w, err := zw.Create(packJSONFileName)
	if err != nil {
		zw.Close()
		return fmt.Errorf("create zip entry: %w", err)
	}

	var payload []byte
	if password != "" {
		payload, err = encryptData(jsonData, password)
		if err != nil {
			zw.Close()
			return fmt.Errorf("encrypt data: %w", err)
		}
	} else {
		payload = jsonData
	}

	if _, err := w.Write(payload); err != nil {
		zw.Close()
		return fmt.Errorf("write zip entry: %w", err)
	}

	// 2. Create metadata.json (always unencrypted)
	var meta struct {
		Metadata PackMetadata `json:"metadata"`
	}
	if err := json.Unmarshal(jsonData, &meta); err == nil {
		mw, err := zw.Create(metadataJSONFileName)
		if err == nil {
			metaBytes, _ := json.MarshalIndent(meta.Metadata, "", "  ")
			if _, err := mw.Write(metaBytes); err != nil {
				zw.Close()
				return fmt.Errorf("write metadata entry: %w", err)
			}
		}
	}

	// Explicitly close zip writer to flush central directory — defer would swallow errors
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finalize zip: %w", err)
	}
	return nil
}

// UnpackFromZip reads the "pack.json" entry from the ZIP file at zipPath.
// If the entry is encrypted, password is used to decrypt it.
// Returns the raw JSON bytes.
func UnpackFromZip(zipPath string, password string) ([]byte, error) {
	data, err := readPackEntry(zipPath)
	if err != nil {
		return nil, err
	}

	if isDataEncrypted(data) {
		if password == "" {
			return nil, ErrPasswordRequired
		}
		decrypted, err := decryptData(data, password)
		if err != nil {
			return nil, err
		}
		return decrypted, nil
	}

	// Not encrypted — if a password was provided, that's fine, just ignore it.
	return data, nil
}

// ReadMetadataFromZip attempts to read metadata from the ZIP file at zipPath.
// It first tries "metadata.json" (unencrypted), and falls back to "pack.json"
// if it's not encrypted.
func ReadMetadataFromZip(zipPath string) (PackMetadata, bool, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return PackMetadata{}, false, fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	var metadata PackMetadata
	foundMetadata := false

	// Try metadata.json first
	for _, f := range zr.File {
		if f.Name == metadataJSONFileName {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err == nil {
				if err := json.Unmarshal(data, &metadata); err == nil {
					foundMetadata = true
					break
				}
			}
		}
	}

	// Fallback to pack.json if metadata.json not found or failed
	isEncrypted := false
	if !foundMetadata {
		for _, f := range zr.File {
			if f.Name == packJSONFileName {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err == nil {
					if isDataEncrypted(data) {
						isEncrypted = true
					} else {
						var pack struct {
							Metadata PackMetadata `json:"metadata"`
						}
						if err := json.Unmarshal(data, &pack); err == nil {
							metadata = pack.Metadata
							foundMetadata = true
						}
					}
				}
				break
			}
		}
	} else {
		// If we found metadata.json, we still need to know if pack.json is encrypted
		for _, f := range zr.File {
			if f.Name == packJSONFileName {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				header := make([]byte, len(encryptionMagic))
				io.ReadFull(rc, header)
				rc.Close()
				if string(header) == encryptionMagic {
					isEncrypted = true
				}
				break
			}
		}
	}

	if !foundMetadata {
		return PackMetadata{}, isEncrypted, fmt.Errorf("metadata not found or encrypted")
	}

	return metadata, isEncrypted, nil
}

// IsEncrypted checks whether the ZIP file at zipPath contains encrypted data.
func IsEncrypted(zipPath string) (bool, error) {
	data, err := readPackEntry(zipPath)
	if err != nil {
		return false, err
	}
	return isDataEncrypted(data), nil
}

// readPackEntry opens the ZIP file and reads the raw bytes of the "pack.json" entry.
func readPackEntry(zipPath string) ([]byte, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == packJSONFileName {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open zip entry: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, ErrInvalidZip
}

// isDataEncrypted checks if raw data starts with the encryption magic header.
func isDataEncrypted(data []byte) bool {
	return len(data) >= len(encryptionMagic) && string(data[:len(encryptionMagic)]) == encryptionMagic
}

// encryptData encrypts plaintext using AES-256-GCM with a key derived from password via scrypt.
// Format: QAPENC (6 bytes) | salt (32 bytes) | nonce (12 bytes) | ciphertext
func encryptData(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	var buf bytes.Buffer
	buf.WriteString(encryptionMagic)
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

// decryptData decrypts data that was encrypted by encryptData.
// Returns ErrWrongPassword if the password is incorrect.
func decryptData(data []byte, password string) ([]byte, error) {
	magicLen := len(encryptionMagic)
	minLen := magicLen + saltLen + 12 // magic + salt + minimum nonce size
	if len(data) < minLen {
		return nil, ErrInvalidZip
	}

	salt := data[magicLen : magicLen+saltLen]

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceStart := magicLen + saltLen
	nonceEnd := nonceStart + gcm.NonceSize()
	if len(data) < nonceEnd {
		return nil, ErrInvalidZip
	}

	nonce := data[nonceStart:nonceEnd]
	ciphertext := data[nonceEnd:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrWrongPassword
	}
	return plaintext, nil
}
