package main

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestQueue(t *testing.T) *PendingUsageQueue {
	t.Helper()
	dir := t.TempDir()
	return &PendingUsageQueue{
		filePath: filepath.Join(dir, "pending_usage_reports.json"),
		records:  []PendingUsageRecord{},
	}
}

func TestNewPendingUsageQueue(t *testing.T) {
	q, err := NewPendingUsageQueue()
	if err != nil {
		t.Fatalf("NewPendingUsageQueue() error: %v", err)
	}
	if q == nil {
		t.Fatal("NewPendingUsageQueue() returned nil")
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got Len()=%d", q.Len())
	}
}

func TestLoadFileNotExist(t *testing.T) {
	q := newTestQueue(t)
	err := q.Load()
	if err != nil {
		t.Fatalf("Load() on non-existent file should not error, got: %v", err)
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue after loading non-existent file, got Len()=%d", q.Len())
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	q := newTestQueue(t)
	// Write corrupted JSON
	dir := filepath.Dir(q.filePath)
	os.MkdirAll(dir, 0755)
	os.WriteFile(q.filePath, []byte("not valid json{{{"), 0644)

	err := q.Load()
	if err != nil {
		t.Fatalf("Load() on corrupted file should not error, got: %v", err)
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue after loading corrupted file, got Len()=%d", q.Len())
	}
}

func TestEnqueueAndGetAll(t *testing.T) {
	q := newTestQueue(t)
	rec := PendingUsageRecord{ListingID: 42, UsedAt: "2024-01-15T10:30:00Z"}

	if err := q.Enqueue(rec); err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("expected Len()=1, got %d", q.Len())
	}

	all := q.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 record, got %d", len(all))
	}
	if all[0].ListingID != 42 || all[0].UsedAt != "2024-01-15T10:30:00Z" {
		t.Errorf("unexpected record: %+v", all[0])
	}
}

func TestDequeue(t *testing.T) {
	q := newTestQueue(t)
	r1 := PendingUsageRecord{ListingID: 1, UsedAt: "2024-01-01T00:00:00Z"}
	r2 := PendingUsageRecord{ListingID: 2, UsedAt: "2024-01-02T00:00:00Z"}

	q.Enqueue(r1)
	q.Enqueue(r2)

	if q.Len() != 2 {
		t.Fatalf("expected Len()=2, got %d", q.Len())
	}

	if err := q.Dequeue(1, "2024-01-01T00:00:00Z"); err != nil {
		t.Fatalf("Dequeue() error: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("expected Len()=1 after dequeue, got %d", q.Len())
	}

	all := q.GetAll()
	if all[0].ListingID != 2 {
		t.Errorf("expected remaining record ListingID=2, got %d", all[0].ListingID)
	}
}

func TestDequeueNonExistent(t *testing.T) {
	q := newTestQueue(t)
	q.Enqueue(PendingUsageRecord{ListingID: 1, UsedAt: "2024-01-01T00:00:00Z"})

	// Dequeue a record that doesn't exist — should be a no-op
	if err := q.Dequeue(999, "2024-12-31T00:00:00Z"); err != nil {
		t.Fatalf("Dequeue() non-existent should not error, got: %v", err)
	}
	if q.Len() != 1 {
		t.Errorf("expected Len()=1 after no-op dequeue, got %d", q.Len())
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	q := newTestQueue(t)
	records := []PendingUsageRecord{
		{ListingID: 10, UsedAt: "2024-01-10T10:00:00Z"},
		{ListingID: 20, UsedAt: "2024-01-20T20:00:00Z"},
		{ListingID: 30, UsedAt: "2024-01-30T12:00:00Z"},
	}
	for _, r := range records {
		q.Enqueue(r)
	}

	// Create a new queue pointing to the same file and load
	q2 := &PendingUsageQueue{
		filePath: q.filePath,
		records:  []PendingUsageRecord{},
	}
	if err := q2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if q2.Len() != len(records) {
		t.Fatalf("expected Len()=%d after load, got %d", len(records), q2.Len())
	}

	all := q2.GetAll()
	for i, r := range records {
		if all[i].ListingID != r.ListingID || all[i].UsedAt != r.UsedAt {
			t.Errorf("record[%d] mismatch: expected %+v, got %+v", i, r, all[i])
		}
	}
}

func TestGetAllReturnsCopy(t *testing.T) {
	q := newTestQueue(t)
	q.Enqueue(PendingUsageRecord{ListingID: 1, UsedAt: "2024-01-01T00:00:00Z"})

	all := q.GetAll()
	all[0].ListingID = 999 // mutate the copy

	// Original should be unchanged
	original := q.GetAll()
	if original[0].ListingID != 1 {
		t.Errorf("GetAll() did not return a copy; original was mutated")
	}
}
