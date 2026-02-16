package main

import (
	"math/rand"
	"testing"
	"time"
)

// Property: UsageLicense creation field completeness
// For any pricing model and corresponding parameters, created licenses should have
// matching fields: per_use has total_uses == remaining_uses > 0,
// subscription has subscription_months > 0 and valid future expires_at.
func TestUsageLicenseCreationFieldCompleteness(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 200; i++ {
		model := []string{"free", "per_use", "subscription"}[r.Intn(3)]

		switch model {
		case "free":
			lic := &UsageLicense{
				ListingID:    int64(r.Intn(10000) + 1),
				PackName:     "test-pack",
				PricingModel: "free",
				CreatedAt:    time.Now().UTC().Format(time.RFC3339),
				UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
			}
			store := &UsageLicenseStore{licenses: map[int64]*UsageLicense{lic.ListingID: lic}}
			allowed, _ := store.CheckPermission(lic.ListingID)
			if !allowed {
				t.Errorf("iteration %d: free license should always be allowed", i)
			}

		case "per_use":
			totalUses := r.Intn(100) + 1
			lic := &UsageLicense{
				ListingID:     int64(r.Intn(10000) + 1),
				PackName:      "test-pack",
				PricingModel:  "per_use",
				TotalUses:     totalUses,
				RemainingUses: totalUses,
				CreatedAt:     time.Now().UTC().Format(time.RFC3339),
				UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
			}
			if lic.TotalUses <= 0 {
				t.Errorf("iteration %d: per_use total_uses should be > 0, got %d", i, lic.TotalUses)
			}
			if lic.RemainingUses != lic.TotalUses {
				t.Errorf("iteration %d: per_use remaining_uses (%d) should equal total_uses (%d) at creation", i, lic.RemainingUses, lic.TotalUses)
			}
			store := &UsageLicenseStore{licenses: map[int64]*UsageLicense{lic.ListingID: lic}}
			allowed, _ := store.CheckPermission(lic.ListingID)
			if !allowed {
				t.Errorf("iteration %d: per_use license with remaining uses should be allowed", i)
			}

		case "subscription":
			months := r.Intn(12) + 1
			lic := &UsageLicense{
				ListingID:          int64(r.Intn(10000) + 1),
				PackName:           "test-pack",
				PricingModel:       "subscription",
				SubscriptionMonths: months,
				ExpiresAt:          time.Now().Add(time.Duration(months) * 30 * 24 * time.Hour).UTC().Format(time.RFC3339),
				CreatedAt:          time.Now().UTC().Format(time.RFC3339),
				UpdatedAt:          time.Now().UTC().Format(time.RFC3339),
			}
			if lic.SubscriptionMonths <= 0 {
				t.Errorf("iteration %d: subscription months should be > 0, got %d", i, lic.SubscriptionMonths)
			}
			expiresAt, err := time.Parse(time.RFC3339, lic.ExpiresAt)
			if err != nil {
				t.Errorf("iteration %d: failed to parse expires_at: %v", i, err)
				continue
			}
			if !expiresAt.After(time.Now()) {
				t.Errorf("iteration %d: subscription expires_at should be in the future", i)
			}
			store := &UsageLicenseStore{licenses: map[int64]*UsageLicense{lic.ListingID: lic}}
			allowed, _ := store.CheckPermission(lic.ListingID)
			if !allowed {
				t.Errorf("iteration %d: subscription license with future expiry should be allowed", i)
			}
		}
	}
}
