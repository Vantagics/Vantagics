package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

// Encryption constants matching the client-side format in quick_analysis_pack_zip.go.
const (
	serverEncryptionMagic = "QAPENC" // 6-byte magic header
	serverScryptN         = 32768
	serverScryptR         = 8
	serverScryptP         = 1
	serverScryptKeyLen    = 32 // AES-256
	serverSaltLen         = 32
)

// generateSecurePassword generates a cryptographically secure random password.
// It produces 32 random bytes and returns them as a 64-character hex string.
func generateSecurePassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// serverEncryptPackJSON encrypts plainJSON using AES-256-GCM with a key derived
// from password via scrypt. The output format matches the client-side QAPENC format:
//
//	QAPENC (6 bytes) | salt (32 bytes) | nonce (12 bytes) | ciphertext
func serverEncryptPackJSON(plainJSON []byte, password string) ([]byte, error) {
	salt := make([]byte, serverSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	key, err := scrypt.Key([]byte(password), salt, serverScryptN, serverScryptR, serverScryptP, serverScryptKeyLen)
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

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes for GCM
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plainJSON, nil)

	var buf bytes.Buffer
	buf.WriteString(serverEncryptionMagic)
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

// repackZipWithEncryptedData creates a new ZIP archive containing the encrypted
// pack.json data and the original metadata.json from the source ZIP.
func repackZipWithEncryptedData(originalZip []byte, encryptedPackJSON []byte) ([]byte, error) {
	// Read metadata.json from the original ZIP
	zr, err := zip.NewReader(bytes.NewReader(originalZip), int64(len(originalZip)))
	if err != nil {
		return nil, fmt.Errorf("open original zip: %w", err)
	}

	var metadataJSON []byte
	for _, f := range zr.File {
		if f.Name == "metadata.json" {
			rc, openErr := f.Open()
			if openErr != nil {
				return nil, fmt.Errorf("open metadata.json: %w", openErr)
			}
			metadataJSON, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("read metadata.json: %w", err)
			}
			break
		}
	}

	// Create new ZIP with encrypted pack.json + original metadata.json
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	pw, err := zw.Create("pack.json")
	if err != nil {
		return nil, fmt.Errorf("create pack.json entry: %w", err)
	}
	if _, err := pw.Write(encryptedPackJSON); err != nil {
		return nil, fmt.Errorf("write pack.json: %w", err)
	}

	if metadataJSON != nil {
		mw, err := zw.Create("metadata.json")
		if err != nil {
			return nil, fmt.Errorf("create metadata.json entry: %w", err)
		}
		if _, err := mw.Write(metadataJSON); err != nil {
			return nil, fmt.Errorf("write metadata.json: %w", err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}

// injectListingIDIntoQAP injects listing_id into the pack.json and metadata.json
// entries of a .qap ZIP file. The pack.json must NOT be encrypted (call this before
// encryption). Returns the modified ZIP bytes.
func injectListingIDIntoQAP(zipData []byte, listingID int64) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	// Read all entries
	var packJSONBytes []byte
	var metadataJSONBytes []byte
	var packEntryName string // "pack.json" or "analysis_pack.json"

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		switch f.Name {
		case "pack.json", "analysis_pack.json":
			packJSONBytes = data
			packEntryName = f.Name
		case "metadata.json":
			metadataJSONBytes = data
		}
	}

	if packJSONBytes == nil {
		return nil, fmt.Errorf("pack.json not found in ZIP")
	}

	// Check not encrypted
	if len(packJSONBytes) >= 6 && string(packJSONBytes[:6]) == serverEncryptionMagic {
		return nil, fmt.Errorf("pack.json is encrypted, cannot inject listing_id")
	}

	// Inject listing_id into pack.json using generic map to preserve all fields
	var packMap map[string]interface{}
	if err := json.Unmarshal(packJSONBytes, &packMap); err != nil {
		return nil, fmt.Errorf("parse pack.json: %w", err)
	}
	if metadata, ok := packMap["metadata"].(map[string]interface{}); ok {
		metadata["listing_id"] = listingID
	} else {
		// No metadata field — create one with just listing_id
		packMap["metadata"] = map[string]interface{}{"listing_id": listingID}
	}
	newPackJSON, err := json.Marshal(packMap)
	if err != nil {
		return nil, fmt.Errorf("marshal pack.json: %w", err)
	}

	// Inject listing_id into metadata.json if present
	var newMetadataJSON []byte
	if metadataJSONBytes != nil {
		var metaMap map[string]interface{}
		if err := json.Unmarshal(metadataJSONBytes, &metaMap); err == nil {
			metaMap["listing_id"] = listingID
			if b, err := json.MarshalIndent(metaMap, "", "  "); err == nil {
				newMetadataJSON = b
			}
		}
	}

	// Rebuild ZIP
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	pw, err := zw.Create(packEntryName)
	if err != nil {
		return nil, fmt.Errorf("create %s entry: %w", packEntryName, err)
	}
	if _, err := pw.Write(newPackJSON); err != nil {
		return nil, fmt.Errorf("write %s: %w", packEntryName, err)
	}

	if newMetadataJSON != nil {
		mw, err := zw.Create("metadata.json")
		if err != nil {
			return nil, fmt.Errorf("create metadata.json entry: %w", err)
		}
		if _, err := mw.Write(newMetadataJSON); err != nil {
			return nil, fmt.Errorf("write metadata.json: %w", err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}
