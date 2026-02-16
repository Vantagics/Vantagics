package agent

// Feature: credits-upload-interval-fix, Property 2: ShouldReportNow 变化量检查
// **Validates: Requirements 1.3**

import (
	"testing"
	"testing/quick"
	"time"
)

// TestPropertyShouldReportNow verifies that ShouldReportNow() returns the correct
// boolean for any (lastReportAt, usedCredits, lastReportedCredits) combination.
//
// Property 2: For any (lastReportAt, usedCredits, lastReportedCredits) combination,
// when lastReportAt is non-zero and >= 1 hour ago, ShouldReportNow() returns true
// if and only if usedCredits != lastReportedCredits.
func TestPropertyShouldReportNow(t *testing.T) {
	config := &quick.Config{MaxCount: 100}

	// Sub-property 1: lastReportAt non-zero, >= 1 hour ago, credits differ => true
	t.Run("IntervalMet_CreditsDiffer_ReturnsTrue", func(t *testing.T) {
		err := quick.Check(func(extraMinRaw uint16, usedRaw uint16, deltaRaw uint16) bool {
			// Duration >= 1 hour: [60, 300] minutes
			extraMin := int(extraMinRaw%241) + 60
			duration := time.Duration(extraMin) * time.Minute

			usedCredits := float64(usedRaw%1000) + 1.0
			// Ensure lastReportedCredits != usedCredits by adding a non-zero delta
			delta := float64(deltaRaw%999) + 1.0
			lastReportedCredits := usedCredits + delta

			client := &LicenseClient{
				data:                &ActivationData{UsedCredits: usedCredits},
				lastReportAt:        time.Now().Add(-duration),
				lastReportedCredits: lastReportedCredits,
			}

			result := client.ShouldReportNow()
			if !result {
				t.Logf("Expected true: duration=%v, used=%.1f, lastReported=%.1f", duration, usedCredits, lastReportedCredits)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 2: lastReportAt non-zero, >= 1 hour ago, credits same => false
	t.Run("IntervalMet_CreditsSame_ReturnsFalse", func(t *testing.T) {
		err := quick.Check(func(extraMinRaw uint16, creditsRaw uint16) bool {
			extraMin := int(extraMinRaw%241) + 60
			duration := time.Duration(extraMin) * time.Minute

			credits := float64(creditsRaw%1000) + 1.0

			client := &LicenseClient{
				data:                &ActivationData{UsedCredits: credits},
				lastReportAt:        time.Now().Add(-duration),
				lastReportedCredits: credits, // same as usedCredits
			}

			result := client.ShouldReportNow()
			if result {
				t.Logf("Expected false: duration=%v, used=%.1f, lastReported=%.1f", duration, credits, credits)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 3: lastReportAt non-zero, < 1 hour ago => always false
	t.Run("IntervalNotMet_AlwaysFalse", func(t *testing.T) {
		err := quick.Check(func(minRaw uint16, usedRaw uint16, reportedRaw uint16) bool {
			// Duration in [1, 59] minutes (strictly within 1 hour)
			minutes := int(minRaw%59) + 1
			duration := time.Duration(minutes) * time.Minute

			usedCredits := float64(usedRaw % 1000)
			lastReportedCredits := float64(reportedRaw % 1000)

			client := &LicenseClient{
				data:                &ActivationData{UsedCredits: usedCredits},
				lastReportAt:        time.Now().Add(-duration),
				lastReportedCredits: lastReportedCredits,
			}

			result := client.ShouldReportNow()
			if result {
				t.Logf("Expected false: duration=%v, used=%.1f, lastReported=%.1f", duration, usedCredits, lastReportedCredits)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 4: lastReportAt is zero, usedCredits > 0 => true
	t.Run("ZeroLastReport_PositiveCredits_ReturnsTrue", func(t *testing.T) {
		err := quick.Check(func(creditsRaw uint16) bool {
			credits := float64(creditsRaw%1000) + 1.0 // always > 0

			client := &LicenseClient{
				data: &ActivationData{UsedCredits: credits},
				// lastReportAt is zero value
			}

			result := client.ShouldReportNow()
			if !result {
				t.Logf("Expected true for zero lastReportAt with credits=%.1f", credits)
				return false
			}
			return true
		}, config)
		if err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// Sub-property 5: lastReportAt is zero, usedCredits == 0 => false
	t.Run("ZeroLastReport_ZeroCredits_ReturnsFalse", func(t *testing.T) {
		client := &LicenseClient{
			data: &ActivationData{UsedCredits: 0},
			// lastReportAt is zero value
		}

		result := client.ShouldReportNow()
		if result {
			t.Errorf("Expected false for zero lastReportAt with zero credits")
		}
	})

	// Sub-property 6: data is nil => always false
	t.Run("NilData_ReturnsFalse", func(t *testing.T) {
		client := &LicenseClient{
			data: nil,
		}

		result := client.ShouldReportNow()
		if result {
			t.Errorf("Expected false when data is nil")
		}
	})
}
