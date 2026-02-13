package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalPackInfo represents a locally available quick analysis pack (.qap file in QAP directory).
type LocalPackInfo struct {
	FileName    string `json:"file_name"`
	FilePath    string `json:"file_path"`
	PackName    string `json:"pack_name"`
	Description string `json:"description"`
	SourceName  string `json:"source_name"`
	Author      string `json:"author"`
	CreatedAt   string `json:"created_at"`
	IsEncrypted bool   `json:"is_encrypted"`
}

// ListLocalQuickAnalysisPacks returns all local quick analysis packs by scanning
// .qap files in the {DataCacheDir}/qap/ directory.
func (a *App) ListLocalQuickAnalysisPacks() ([]LocalPackInfo, error) {
	cfg, err := a.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	qapDir := filepath.Join(cfg.DataCacheDir, "qap")

	entries, err := os.ReadDir(qapDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LocalPackInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read qap directory: %w", err)
	}

	var packs []LocalPackInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".qap") {
			continue
		}

		filePath := filepath.Join(qapDir, entry.Name())

		// Try to unpack with empty password
		jsonData, err := UnpackFromZip(filePath, "")
		if err != nil {
			if errors.Is(err, ErrPasswordRequired) {
				// Encrypted file — add with IsEncrypted flag, minimal info
				packs = append(packs, LocalPackInfo{
					FileName:    entry.Name(),
					FilePath:    filePath,
					PackName:    entry.Name(),
					IsEncrypted: true,
				})
			}
			// Corrupted or other error — skip
			continue
		}

		var pack QuickAnalysisPack
		if err := json.Unmarshal(jsonData, &pack); err != nil {
			// Invalid JSON — skip
			continue
		}

		packs = append(packs, LocalPackInfo{
			FileName:    entry.Name(),
			FilePath:    filePath,
			PackName:    pack.Metadata.SourceName,
			Description: pack.Metadata.Description,
			SourceName:  pack.Metadata.SourceName,
			Author:      pack.Metadata.Author,
			CreatedAt:   pack.Metadata.CreatedAt,
			IsEncrypted: false,
		})
	}

	// Sort by CreatedAt descending
	sort.Slice(packs, func(i, j int) bool {
		return packs[i].CreatedAt > packs[j].CreatedAt
	})

	if packs == nil {
		packs = []LocalPackInfo{}
	}
	return packs, nil
}
