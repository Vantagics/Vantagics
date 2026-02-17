package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeleteLocalPack deletes a .qap file from the QAP directory.
// It validates the path is within the QAP directory to prevent path traversal attacks.
func (a *App) DeleteLocalPack(filePath string) error {
	qapDir, err := a.getQAPDir()
	if err != nil {
		return fmt.Errorf("get QAP directory: %w", err)
	}

	if err := validatePathInQAPDir(filePath, qapDir); err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

// UpdatePackMetadata updates the name, description and author fields in a .qap file's metadata.
// Encrypted files cannot be edited and will return an error.
func (a *App) UpdatePackMetadata(filePath, packName, description, author string) error {
	qapDir, err := a.getQAPDir()
	if err != nil {
		return fmt.Errorf("get QAP directory: %w", err)
	}

	if err := validatePathInQAPDir(filePath, qapDir); err != nil {
		return err
	}

	// Try to unpack with empty password
	jsonData, err := UnpackFromZip(filePath, "")
	if err != nil {
		if errors.Is(err, ErrPasswordRequired) {
			return fmt.Errorf("encrypted files cannot be edited")
		}
		return fmt.Errorf("read pack file: %w", err)
	}

	var pack QuickAnalysisPack
	if err := json.Unmarshal(jsonData, &pack); err != nil {
		return fmt.Errorf("parse pack JSON: %w", err)
	}

	pack.Metadata.PackName = packName
	pack.Metadata.Description = description
	pack.Metadata.Author = author

	updatedJSON, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated pack: %w", err)
	}

	if err := PackToZip(updatedJSON, filePath, ""); err != nil {
		return fmt.Errorf("write pack file: %w", err)
	}

	return nil
}

// getQAPDir returns the absolute path to the QAP directory.
func (a *App) getQAPDir() (string, error) {
	cfg, err := a.GetConfig()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	qapDir := filepath.Join(cfg.DataCacheDir, qapSubDir)
	absDir, err := filepath.Abs(qapDir)
	if err != nil {
		return "", fmt.Errorf("resolve QAP directory: %w", err)
	}
	return absDir, nil
}

// validatePathInQAPDir checks that filePath is within qapDir to prevent path traversal.
// qapDir is expected to already be an absolute path from getQAPDir().
func validatePathInQAPDir(filePath, qapDir string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolve file path: %w", err)
	}
	// Ensure the path starts with the QAP directory (with trailing separator to avoid prefix attacks)
	if !strings.HasPrefix(absPath, qapDir+string(filepath.Separator)) {
		return fmt.Errorf("path is outside QAP directory")
	}
	return nil
}
