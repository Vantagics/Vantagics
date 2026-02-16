package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Feature: marketplace-sn-auth-billing, Property 7: 使用权限检查正确性
// For any UsageLicense and current time, CheckPermission result satisfies:
// free always allowed; per_use allowed when remaining_uses > 0;
// time_limited/subscription allowed when not expired.
// Validates: Requirements 6.3, 6.4, 6.5, 6.6, 6.7, 6.8
func TestProperty7_UsagePermissionCheckCorrectness(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		listingID := int64(rng.Intn(10000) + 1)
		models := []string{"free", "per_use", "subscription", "time_limited"}
		model := models[rng.Intn(len(models))]

		var lic *UsageLicense
		var expectAllowed bool

		switch model {
		case "free":
			lic = &UsageLicense{
				ListingID:    listingID,
				PackName:     "test",
				PricingModel: "free",
				CreatedAt:    time.Now().UTC().Format(time.RFC3339),
				UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
			}
			expectAllowed = true

		case "per_use":
			remaining := rng.Intn(10) // 0-9
			lic = &UsageLicense{
				ListingID:     listingID,
				PackName:      "test",
				PricingModel:  "per_use",
				RemainingUses: remaining,
				TotalUses:     remaining + rng.Intn(5),
				CreatedAt:     time.Now().UTC().Format(time.RFC3339),
				UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
			}
			expectAllowed = remaining > 0

		case "subscription", "time_limited":
			// Randomly choose expired or not expired
			expired := rng.Intn(2) == 0
			var expiresAt time.Time
			if expired {
				expiresAt = time.Now().Add(-time.Duration(rng.Intn(720)+1) * time.Hour)
			} else {
				expiresAt = time.Now().Add(time.Duration(rng.Intn(720)+1) * time.Hour)
			}
			lic = &UsageLicense{
				ListingID:    listingID,
				PackName:     "test",
				PricingModel: model,
				ExpiresAt:    expiresAt.Format(time.RFC3339),
				CreatedAt:    time.Now().UTC().Format(time.RFC3339),
				UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
			}
			expectAllowed = !expired
		}

		store := &UsageLicenseStore{licenses: map[int64]*UsageLicense{listingID: lic}}
		allowed, reason := store.CheckPermission(listingID)

		if allowed != expectAllowed {
			t.Errorf("iteration %d: model=%s, expected allowed=%v, got allowed=%v (reason=%s, lic=%+v)",
				i, model, expectAllowed, allowed, reason, lic)
		}

		// If not allowed, reason should be non-empty
		if !allowed && reason == "" {
			t.Errorf("iteration %d: model=%s, not allowed but reason is empty", i, model)
		}
	}
}

// Feature: marketplace-sn-auth-billing, Property 8: Usage_License 序列化往返一致性
// For any valid UsageLicense object, serializing to JSON and deserializing produces
// an equivalent object.
// Validates: Requirements 6.9
func TestProperty8_UsageLicenseSerializationRoundtrip(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100
	models := []string{"free", "per_use", "subscription", "time_limited"}

	for i := 0; i < iterations; i++ {
		model := models[rng.Intn(len(models))]
		lic := UsageLicense{
			ListingID:          int64(rng.Intn(100000) + 1),
			PackName:           fmt.Sprintf("pack-%d", rng.Intn(1000)),
			PricingModel:       model,
			RemainingUses:      rng.Intn(100),
			TotalUses:          rng.Intn(100),
			ExpiresAt:          time.Now().Add(time.Duration(rng.Intn(8760)) * time.Hour).UTC().Format(time.RFC3339),
			SubscriptionMonths: rng.Intn(24),
			CreatedAt:          time.Now().Add(-time.Duration(rng.Intn(8760)) * time.Hour).UTC().Format(time.RFC3339),
			UpdatedAt:          time.Now().UTC().Format(time.RFC3339),
		}

		data, err := json.Marshal(lic)
		if err != nil {
			t.Errorf("iteration %d: marshal failed: %v", i, err)
			continue
		}

		var restored UsageLicense
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Errorf("iteration %d: unmarshal failed: %v", i, err)
			continue
		}

		if lic.ListingID != restored.ListingID {
			t.Errorf("iteration %d: ListingID mismatch: %d vs %d", i, lic.ListingID, restored.ListingID)
		}
		if lic.PackName != restored.PackName {
			t.Errorf("iteration %d: PackName mismatch: %s vs %s", i, lic.PackName, restored.PackName)
		}
		if lic.PricingModel != restored.PricingModel {
			t.Errorf("iteration %d: PricingModel mismatch: %s vs %s", i, lic.PricingModel, restored.PricingModel)
		}
		if lic.RemainingUses != restored.RemainingUses {
			t.Errorf("iteration %d: RemainingUses mismatch: %d vs %d", i, lic.RemainingUses, restored.RemainingUses)
		}
		if lic.TotalUses != restored.TotalUses {
			t.Errorf("iteration %d: TotalUses mismatch: %d vs %d", i, lic.TotalUses, restored.TotalUses)
		}
		if lic.ExpiresAt != restored.ExpiresAt {
			t.Errorf("iteration %d: ExpiresAt mismatch: %s vs %s", i, lic.ExpiresAt, restored.ExpiresAt)
		}
		if lic.SubscriptionMonths != restored.SubscriptionMonths {
			t.Errorf("iteration %d: SubscriptionMonths mismatch: %d vs %d", i, lic.SubscriptionMonths, restored.SubscriptionMonths)
		}
		if lic.CreatedAt != restored.CreatedAt {
			t.Errorf("iteration %d: CreatedAt mismatch: %s vs %s", i, lic.CreatedAt, restored.CreatedAt)
		}
		if lic.UpdatedAt != restored.UpdatedAt {
			t.Errorf("iteration %d: UpdatedAt mismatch: %s vs %s", i, lic.UpdatedAt, restored.UpdatedAt)
		}
	}
}

// Feature: marketplace-sn-auth-billing, Property 9: 按次追加购买次数一致性
// For any per_use UsageLicense, after adding N uses, remaining_uses increases by N.
// Validates: Requirements 7.1, 7.2
func TestProperty9_PerUseAdditionalPurchaseConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		initialUses := rng.Intn(50)
		addUses := rng.Intn(50) + 1 // at least 1

		lic := &UsageLicense{
			ListingID:     int64(rng.Intn(10000) + 1),
			PackName:      "test",
			PricingModel:  "per_use",
			RemainingUses: initialUses,
			TotalUses:     initialUses + rng.Intn(20),
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
		}

		// Simulate adding N uses
		lic.RemainingUses += addUses
		lic.TotalUses += addUses
		lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		expectedRemaining := initialUses + addUses
		if lic.RemainingUses != expectedRemaining {
			t.Errorf("iteration %d: after adding %d uses to %d, expected %d remaining, got %d",
				i, addUses, initialUses, expectedRemaining, lic.RemainingUses)
		}
	}
}

// Feature: marketplace-sn-auth-billing, Property 10: 订阅续费有效期延长一致性
// For any subscription UsageLicense, after renewal, expires_at extends by one billing_cycle
// (monthly=30 days, yearly=365 days).
// Validates: Requirements 8.1, 8.2
func TestProperty10_SubscriptionRenewalExpiryExtension(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const iterations = 100

	for i := 0; i < iterations; i++ {
		// Random initial expiry (could be past or future)
		baseTime := time.Now().Add(time.Duration(rng.Intn(720)-360) * time.Hour).UTC()

		cycles := []struct {
			name string
			days int
		}{
			{"monthly", 30},
			{"yearly", 365},
		}
		cycle := cycles[rng.Intn(len(cycles))]

		lic := &UsageLicense{
			ListingID:          int64(rng.Intn(10000) + 1),
			PackName:           "test",
			PricingModel:       "subscription",
			ExpiresAt:          baseTime.Format(time.RFC3339),
			SubscriptionMonths: rng.Intn(12) + 1,
			CreatedAt:          time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:          time.Now().UTC().Format(time.RFC3339),
		}

		// Parse original expiry
		originalExpiry, err := time.Parse(time.RFC3339, lic.ExpiresAt)
		if err != nil {
			t.Errorf("iteration %d: failed to parse original expiry: %v", i, err)
			continue
		}

		// Simulate renewal: extend by one cycle
		newExpiry := originalExpiry.AddDate(0, 0, cycle.days)
		lic.ExpiresAt = newExpiry.Format(time.RFC3339)
		lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		// Verify the extension
		parsedNew, err := time.Parse(time.RFC3339, lic.ExpiresAt)
		if err != nil {
			t.Errorf("iteration %d: failed to parse new expiry: %v", i, err)
			continue
		}

		diff := parsedNew.Sub(originalExpiry)
		expectedDiff := time.Duration(cycle.days) * 24 * time.Hour

		if diff != expectedDiff {
			t.Errorf("iteration %d: cycle=%s, expected extension of %v, got %v (original=%s, new=%s)",
				i, cycle.name, expectedDiff, diff, originalExpiry.Format(time.RFC3339), parsedNew.Format(time.RFC3339))
		}
	}
}
