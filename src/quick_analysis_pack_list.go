package main

import (
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

		metadata, isEncrypted, err := ReadMetadataFromZip(filePath)
		if err != nil {
			// If we can't even read metadata, it might be an old corrupted file or something else
			// We still try to show it as encrypted if it's indeed encrypted but metadata is missing
			packs = append(packs, LocalPackInfo{
				FileName:    entry.Name(),
				FilePath:    filePath,
				PackName:    entry.Name(),
				IsEncrypted: isEncrypted,
			})
			continue
		}

		packName := metadata.PackName
		if packName == "" {
			packName = metadata.SourceName
		}

		packs = append(packs, LocalPackInfo{
			FileName:    entry.Name(),
			FilePath:    filePath,
			PackName:    packName,
			Description: metadata.Description,
			SourceName:  metadata.SourceName,
			Author:      metadata.Author,
			CreatedAt:   metadata.CreatedAt,
			IsEncrypted: isEncrypted,
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
