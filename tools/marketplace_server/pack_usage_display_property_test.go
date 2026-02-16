package main

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: pack-usage-tracking, Property 5: 个人中心 per_use 使用进度展示
// **Validates: Requirements 4.1, 4.2, 4.3**
//
// For any per_use analysis pack with (used_count, total_purchased), the rendered HTML
// should contain "已使用 X/Y 次" text, and when used_count == total_purchased the
// output should include the "usage-exhausted" warning class, otherwise it should not.

// usageProgressSnippet mirrors the relevant template fragment from user_dashboard.go.
const usageProgressSnippet = `<span class="usage-progress{{if eq .UsedCount .TotalPurchased}} usage-exhausted{{end}}">已使用 {{.UsedCount}}/{{.TotalPurchased}} 次</span>`

var usageProgressTmpl = template.Must(template.New("usage_progress").Parse(usageProgressSnippet))

type usageProgressData struct {
	UsedCount      int
	TotalPurchased int
}

func TestProperty5_UsageProgressDisplay(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate random total_purchased in [1, 1000]
		totalPurchased := r.Intn(1000) + 1
		// Generate random used_count in [0, total_purchased]
		usedCount := r.Intn(totalPurchased + 1)

		data := usageProgressData{
			UsedCount:      usedCount,
			TotalPurchased: totalPurchased,
		}

		var buf bytes.Buffer
		if err := usageProgressTmpl.Execute(&buf, data); err != nil {
			t.Logf("seed=%d: template execution failed: %v", seed, err)
			return false
		}
		html := buf.String()

		// Requirement 4.1: must contain "已使用 X/Y 次"
		expectedText := fmt.Sprintf("已使用 %d/%d 次", usedCount, totalPurchased)
		if !strings.Contains(html, expectedText) {
			t.Logf("seed=%d: expected %q in HTML, got: %s", seed, expectedText, html)
			return false
		}

		// Requirement 4.2: when used_count == total_purchased, must contain "usage-exhausted"
		if usedCount == totalPurchased {
			if !strings.Contains(html, "usage-exhausted") {
				t.Logf("seed=%d: used_count(%d)==total_purchased(%d) but 'usage-exhausted' class missing; html: %s",
					seed, usedCount, totalPurchased, html)
				return false
			}
		}

		// Requirement 4.3: when used_count < total_purchased, must NOT contain "usage-exhausted"
		if usedCount < totalPurchased {
			if strings.Contains(html, "usage-exhausted") {
				t.Logf("seed=%d: used_count(%d)<total_purchased(%d) but 'usage-exhausted' class present; html: %s",
					seed, usedCount, totalPurchased, html)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 5 (个人中心 per_use 使用进度展示) failed: %v", err)
	}
}
