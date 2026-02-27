package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PendingUsageRecord 待上报的使用记录
type PendingUsageRecord struct {
	ListingID int64  `json:"listing_id"`
	UsedAt    string `json:"used_at"` // RFC3339 时间�?
}

// pendingUsageFileData is the JSON file structure for persisting pending usage records.
type pendingUsageFileData struct {
	Records []PendingUsageRecord `json:"records"`
}

// PendingUsageQueue 管理待上报使用记录的持久化队�?
type PendingUsageQueue struct {
	mu       sync.Mutex
	filePath string
	records  []PendingUsageRecord
}

// NewPendingUsageQueue creates a new PendingUsageQueue with the default file path
// (~/.vantagics/pending_usage_reports.json).
func NewPendingUsageQueue() (*PendingUsageQueue, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	filePath := filepath.Join(home, ".vantagics", "pending_usage_reports.json")
	return &PendingUsageQueue{
		filePath: filePath,
		records:  []PendingUsageRecord{},
	}, nil
}

// Load reads the pending usage queue from the JSON file on disk.
// If the file does not exist, the queue remains empty (no error).
// If the file is corrupted, a warning is logged and the queue is reset to empty.
func (q *PendingUsageQueue) Load() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := os.ReadFile(q.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			q.records = []PendingUsageRecord{}
			return nil
		}
		return fmt.Errorf("failed to read pending usage file: %w", err)
	}

	var fileData pendingUsageFileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		// Log detailed error for debugging, but don't fail - reset queue instead
		fmt.Printf("[PendingUsageQueue] ERROR: corrupted pending usage file %s (size: %d bytes), resetting queue: %v\n", q.filePath, len(data), err)
		q.records = []PendingUsageRecord{}
		// Backup corrupted file for investigation
		backupPath := q.filePath + ".corrupted." + fmt.Sprintf("%d", time.Now().Unix())
		if backupErr := os.WriteFile(backupPath, data, 0600); backupErr == nil {
			fmt.Printf("[PendingUsageQueue] Backed up corrupted file to: %s\n", backupPath)
		}
		return nil
	}

	q.records = fileData.Records
	if q.records == nil {
		q.records = []PendingUsageRecord{}
	}
	return nil
}

// saveLocked writes the current queue to disk atomically. Caller must hold q.mu.
// Uses write-to-temp-then-rename pattern to prevent corruption on crash.
// File permissions set to 0600 (owner read/write only) for security.
func (q *PendingUsageQueue) saveLocked() error {
	dir := filepath.Dir(q.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	fileData := pendingUsageFileData{
		Records: q.records,
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pending usage data: %w", err)
	}

	// Write to a temporary file first, then rename for atomicity
	tmpPath := q.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write pending usage temp file: %w", err)
	}
	if err := os.Rename(tmpPath, q.filePath); err != nil {
		os.Remove(tmpPath) // clean up on rename failure
		return fmt.Errorf("failed to rename pending usage file: %w", err)
	}
	return nil
}

// Save writes the current pending usage queue to the JSON file on disk.
// It creates the parent directory if it does not exist.
func (q *PendingUsageQueue) Save() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.saveLocked()
}

// Enqueue adds a pending usage record to the queue and persists to disk.
func (q *PendingUsageQueue) Enqueue(record PendingUsageRecord) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.records = append(q.records, record)
	return q.saveLocked()
}

// Dequeue removes a pending usage record matching the given listingID and usedAt.
// If no matching record is found, this is a no-op (no disk write).
func (q *PendingUsageQueue) Dequeue(listingID int64, usedAt string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, r := range q.records {
		if r.ListingID == listingID && r.UsedAt == usedAt {
			q.records = append(q.records[:i], q.records[i+1:]...)
			return q.saveLocked()
		}
	}
	// No matching record found �?skip unnecessary disk write
	return nil
}

// GetAll returns a copy of all pending usage records.
func (q *PendingUsageQueue) GetAll() []PendingUsageRecord {
	q.mu.Lock()
	defer q.mu.Unlock()

	result := make([]PendingUsageRecord, len(q.records))
	copy(result, q.records)
	return result
}

// Len returns the number of pending usage records in the queue.
func (q *PendingUsageQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.records)
}
