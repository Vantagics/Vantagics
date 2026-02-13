package config

import (
	"encoding/json"
	"math/rand"
	"testing"
	"testing/quick"
	"time"
)

// generateRandomEmail generates a random valid email string: local@domain.tld
func generateRandomEmail(r *rand.Rand) string {
	const localChars = "abcdefghijklmnopqrstuvwxyz0123456789._-"
	const domainChars = "abcdefghijklmnopqrstuvwxyz0123456789"
	const tldChars = "abcdefghijklmnopqrstuvwxyz"

	localLen := r.Intn(20) + 1
	local := make([]byte, localLen)
	for i := range local {
		local[i] = localChars[r.Intn(len(localChars))]
	}

	domainLen := r.Intn(10) + 1
	domain := make([]byte, domainLen)
	for i := range domain {
		domain[i] = domainChars[r.Intn(len(domainChars))]
	}

	tldLen := r.Intn(4) + 2
	tld := make([]byte, tldLen)
	for i := range tld {
		tld[i] = tldChars[r.Intn(len(tldChars))]
	}

	return string(local) + "@" + string(domain) + "." + string(tld)
}

// Feature: sn-email-binding, Property 2: Config email persistence round-trip
// **Validates: Requirements 2.1, 2.2, 2.4**
func TestProperty2_ConfigEmailPersistenceRoundTrip(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		email := generateRandomEmail(r)

		// Create a Config with the generated email
		original := Config{
			LicenseEmail: email,
		}

		// Serialize to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Logf("seed=%d: marshal failed: %v", seed, err)
			return false
		}

		// Deserialize back
		var restored Config
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Logf("seed=%d: unmarshal failed: %v", seed, err)
			return false
		}

		// Property: email must be preserved through the round-trip
		if restored.LicenseEmail != email {
			t.Logf("seed=%d: email mismatch: original=%q, restored=%q", seed, email, restored.LicenseEmail)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 2 (Config email persistence round-trip) failed: %v", err)
	}
}
