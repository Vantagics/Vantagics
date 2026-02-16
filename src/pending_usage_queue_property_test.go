package main

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"
)

// Feature: pack-usage-tracking, Property 1: PendingUsageQueue 持久化往返一致性
// For any set of PendingUsageRecord records, enqueuing them, saving to disk,
// and loading from disk should produce a queue with content equivalent to the original.
// Validates: Requirements 2.4
func TestProperty1_PendingUsageQueuePersistenceRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		dir := t.TempDir()
		q := &PendingUsageQueue{
			filePath: filepath.Join(dir, "pending_usage_reports.json"),
			records:  []PendingUsageRecord{},
		}

		// Generate a random number of records (0 to 20)
		numRecords := rng.Intn(21)
		originalRecords := make([]PendingUsageRecord, numRecords)
		for j := 0; j < numRecords; j++ {
			listingID := int64(rng.Intn(100000) + 1)
			ts := time.Now().Add(-time.Duration(rng.Intn(8760)) * time.Hour).UTC()
			originalRecords[j] = PendingUsageRecord{
				ListingID: listingID,
				UsedAt:    ts.Format(time.RFC3339),
			}
		}

		// Enqueue all records
		for _, rec := range originalRecords {
			if err := q.Enqueue(rec); err != nil {
				t.Fatalf("iteration %d: Enqueue() error: %v", i, err)
			}
		}

		// Save explicitly (Enqueue already saves, but be explicit)
		if err := q.Save(); err != nil {
			t.Fatalf("iteration %d: Save() error: %v", i, err)
		}

		// Create a new queue pointing to the same file and load
		q2 := &PendingUsageQueue{
			filePath: q.filePath,
			records:  []PendingUsageRecord{},
		}
		if err := q2.Load(); err != nil {
			t.Fatalf("iteration %d: Load() error: %v", i, err)
		}

		// Verify length matches
		if q2.Len() != numRecords {
			t.Errorf("iteration %d: expected Len()=%d after load, got %d", i, numRecords, q2.Len())
			continue
		}

		// Verify each record matches in order
		loaded := q2.GetAll()
		for j, orig := range originalRecords {
			if loaded[j].ListingID != orig.ListingID {
				t.Errorf("iteration %d, record %d: ListingID mismatch: expected %d, got %d",
					i, j, orig.ListingID, loaded[j].ListingID)
			}
			if loaded[j].UsedAt != orig.UsedAt {
				t.Errorf("iteration %d, record %d: UsedAt mismatch: expected %s, got %s",
					i, j, orig.UsedAt, loaded[j].UsedAt)
			}
		}
	}
}

// Feature: pack-usage-tracking, Property 6: 队列入队与出队一致性
// For any PendingUsageRecord, enqueue increases Len() by 1; dequeue decreases Len() by 1
// and the record no longer exists in GetAll().
// Validates: Requirements 2.1, 2.3, 2.5
func TestProperty6_QueueEnqueueDequeueConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		dir := t.TempDir()
		q := &PendingUsageQueue{
			filePath: filepath.Join(dir, "pending_usage_reports.json"),
			records:  []PendingUsageRecord{},
		}

		// Pre-populate with some random records (0 to 10)
		preCount := rng.Intn(11)
		for j := 0; j < preCount; j++ {
			rec := PendingUsageRecord{
				ListingID: int64(rng.Intn(100000) + 1),
				UsedAt:    fmt.Sprintf("2024-%02d-%02dT%02d:00:00Z", rng.Intn(12)+1, rng.Intn(28)+1, rng.Intn(24)),
			}
			if err := q.Enqueue(rec); err != nil {
				t.Fatalf("iteration %d: pre-populate Enqueue() error: %v", i, err)
			}
		}

		lenBefore := q.Len()

		// Generate a unique record to enqueue
		newRec := PendingUsageRecord{
			ListingID: int64(rng.Intn(100000) + 1),
			UsedAt:    time.Now().Add(time.Duration(i) * time.Second).UTC().Format(time.RFC3339),
		}

		// Enqueue and verify Len() increases by 1
		if err := q.Enqueue(newRec); err != nil {
			t.Fatalf("iteration %d: Enqueue() error: %v", i, err)
		}
		lenAfterEnqueue := q.Len()
		if lenAfterEnqueue != lenBefore+1 {
			t.Errorf("iteration %d: after enqueue expected Len()=%d, got %d", i, lenBefore+1, lenAfterEnqueue)
		}

		// Dequeue the same record and verify Len() decreases by 1
		if err := q.Dequeue(newRec.ListingID, newRec.UsedAt); err != nil {
			t.Fatalf("iteration %d: Dequeue() error: %v", i, err)
		}
		lenAfterDequeue := q.Len()
		if lenAfterDequeue != lenBefore {
			t.Errorf("iteration %d: after dequeue expected Len()=%d, got %d", i, lenBefore, lenAfterDequeue)
		}

		// Verify the dequeued record is no longer in GetAll()
		all := q.GetAll()
		for _, r := range all {
			if r.ListingID == newRec.ListingID && r.UsedAt == newRec.UsedAt {
				t.Errorf("iteration %d: dequeued record still found in GetAll(): %+v", i, newRec)
				break
			}
		}
	}
}
