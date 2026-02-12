package main

import "fmt"

// LocalPackInfo represents a locally available quick analysis pack (from replay sessions).
type LocalPackInfo struct {
	ThreadID    string `json:"thread_id"`
	PackName    string `json:"pack_name"`
	Description string `json:"description"`
	SourceName  string `json:"source_name"`
	Author      string `json:"author"`
	QapFilePath string `json:"qap_file_path"`
	CreatedAt   string `json:"created_at"`
}

// ListLocalQuickAnalysisPacks returns all local quick analysis packs by scanning
// chat threads that are replay sessions with pack metadata.
func (a *App) ListLocalQuickAnalysisPacks() ([]LocalPackInfo, error) {
	if a.chatService == nil {
		return nil, fmt.Errorf("chat service not initialized")
	}

	threads, err := a.chatService.LoadThreads()
	if err != nil {
		return nil, fmt.Errorf("failed to load threads: %w", err)
	}

	var packs []LocalPackInfo
	for _, t := range threads {
		if !t.IsReplaySession || t.PackMetadata == nil {
			continue
		}
		packs = append(packs, LocalPackInfo{
			ThreadID:    t.ID,
			PackName:    t.Title,
			Description: t.PackMetadata.Description,
			SourceName:  t.PackMetadata.SourceName,
			Author:      t.PackMetadata.Author,
			QapFilePath: t.QapFilePath,
			CreatedAt:   t.PackMetadata.CreatedAt,
		})
	}

	if packs == nil {
		packs = []LocalPackInfo{}
	}
	return packs, nil
}
