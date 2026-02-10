package agent

import (
	"math"
	"os"
	"testing"
	"testing/quick"
	"time"
)

func TestIsCreditsMode_NilData(t *testing.T) {
	client := NewLicenseClient(nil)
	if client.IsCreditsMode() {
		t.Error("IsCreditsMode() should return false when data is nil")
	}
}

func TestIsCreditsMode_NotCreditsMode(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:   false,
		TotalCredits:  0,
		DailyAnalysis: 10,
	}
	if client.IsCreditsMode() {
		t.Error("IsCreditsMode() should return false when CreditsMode is false")
	}
}

func TestIsCreditsMode_CreditsMode(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 100.0,
	}
	if !client.IsCreditsMode() {
		t.Error("IsCreditsMode() should return true when CreditsMode is true")
	}
}

func TestIsCreditsMode_CreditsModeWithZeroCredits(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 0,
	}
	if !client.IsCreditsMode() {
		t.Error("IsCreditsMode() should return true when CreditsMode is true even with TotalCredits=0")
	}
}

func TestGetCreditsStatus_NilData(t *testing.T) {
	client := NewLicenseClient(nil)
	total, used, isCredits := client.GetCreditsStatus()
	if total != 0 || used != 0 || isCredits {
		t.Errorf("GetCreditsStatus() with nil data = (%v, %v, %v), want (0, 0, false)", total, used, isCredits)
	}
}

func TestGetCreditsStatus_CreditsMode(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 100.0,
		UsedCredits:  30.0,
	}
	total, used, isCredits := client.GetCreditsStatus()
	if total != 100.0 {
		t.Errorf("GetCreditsStatus() totalCredits = %v, want 100.0", total)
	}
	if used != 30.0 {
		t.Errorf("GetCreditsStatus() usedCredits = %v, want 30.0", used)
	}
	if !isCredits {
		t.Error("GetCreditsStatus() isCreditsMode should be true when CreditsMode is true")
	}
}

func TestGetCreditsStatus_DailyLimitMode(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:   false,
		TotalCredits:  0,
		UsedCredits:   0,
		DailyAnalysis: 10,
	}
	total, used, isCredits := client.GetCreditsStatus()
	if total != 0 {
		t.Errorf("GetCreditsStatus() totalCredits = %v, want 0", total)
	}
	if used != 0 {
		t.Errorf("GetCreditsStatus() usedCredits = %v, want 0", used)
	}
	if isCredits {
		t.Error("GetCreditsStatus() isCreditsMode should be false when CreditsMode is false")
	}
}

func TestGetCreditsStatus_ReturnsCurrentValues(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 50.0,
		UsedCredits:  48.5,
	}
	total, used, isCredits := client.GetCreditsStatus()
	if total != 50.0 || used != 48.5 || !isCredits {
		t.Errorf("GetCreditsStatus() = (%v, %v, %v), want (50.0, 48.5, true)", total, used, isCredits)
	}
}

func TestCanAnalyze_CreditsMode_Unlimited(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 0,
	}
	allowed, msg := client.CanAnalyze()
	if !allowed {
		t.Errorf("CanAnalyze() should allow when CreditsMode=true and TotalCredits=0 (unlimited), got msg: %s", msg)
	}
}

func TestCanAnalyze_CreditsMode_Insufficient(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 50.0,
		UsedCredits:  49.0,
	}
	allowed, _ := client.CanAnalyze()
	if allowed {
		t.Error("CanAnalyze() should deny when remaining credits < CreditsPerAnalysis")
	}
}

func TestIncrementAnalysis_CreditsMode_DeductsCredits(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 100.0,
		UsedCredits:  0.0,
	}

	client.IncrementAnalysis()

	if client.data.UsedCredits != CreditsPerAnalysis {
		t.Errorf("IncrementAnalysis() UsedCredits = %v, want %v", client.data.UsedCredits, CreditsPerAnalysis)
	}
}

func TestIncrementAnalysis_CreditsMode_MultipleDeductions(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 100.0,
		UsedCredits:  10.0,
	}

	client.IncrementAnalysis()
	client.IncrementAnalysis()

	expected := 10.0 + 2*CreditsPerAnalysis
	if client.data.UsedCredits != expected {
		t.Errorf("IncrementAnalysis() after 2 calls UsedCredits = %v, want %v", client.data.UsedCredits, expected)
	}
}

func TestIncrementAnalysis_CreditsMode_SkipsDailyCount(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:   true,
		TotalCredits:  100.0,
		UsedCredits:   0.0,
		DailyAnalysis: 5,
	}

	client.IncrementAnalysis()

	if client.analysisCount != 0 {
		t.Errorf("IncrementAnalysis() in credits mode should not increment analysisCount, got %d", client.analysisCount)
	}
}

func TestIncrementAnalysis_CreditsMode_Unlimited_NoDeduction(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 0,
		UsedCredits:  0,
	}

	client.IncrementAnalysis()

	if client.data.UsedCredits != 0 {
		t.Errorf("IncrementAnalysis() with unlimited credits should not deduct, got UsedCredits=%v", client.data.UsedCredits)
	}
}

func TestIncrementAnalysis_CreditsMode_LogsDeduction(t *testing.T) {
	var logMessages []string
	logFn := func(msg string) {
		logMessages = append(logMessages, msg)
	}

	client := NewLicenseClient(logFn)
	client.data = &ActivationData{
		CreditsMode:  true,
		TotalCredits: 100.0,
		UsedCredits:  0.0,
	}

	client.IncrementAnalysis()

	found := false
	for _, msg := range logMessages {
		if msg == "[LICENSE] Credits used: 1.5/100.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("IncrementAnalysis() should log credits deduction, got messages: %v", logMessages)
	}
}

func TestIncrementAnalysis_DailyMode_StillWorks(t *testing.T) {
	client := NewLicenseClient(nil)
	client.data = &ActivationData{
		CreditsMode:   false,
		TotalCredits:  0,
		DailyAnalysis: 10,
	}

	client.IncrementAnalysis()

	if client.analysisCount != 1 {
		t.Errorf("IncrementAnalysis() in daily mode should increment analysisCount, got %d", client.analysisCount)
	}
	if client.data.UsedCredits != 0 {
		t.Errorf("IncrementAnalysis() in daily mode should not change UsedCredits, got %v", client.data.UsedCredits)
	}
}

// Feature: license-credits-support, Property 1: 模式判定一致性
// **Validates: Requirements 1.1, 1.2, 1.3**
//
// For any ActivationData, IsCreditsMode() returns true if and only if CreditsMode is true;
// when CreditsMode is false and daily_analysis > 0, it is Daily_Limit_Mode;
// when CreditsMode is false and daily_analysis == 0, it is unlimited mode.
func TestPropertyModeDetection(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
	}

	// Property 1a: IsCreditsMode() returns true iff CreditsMode field is true
	isCreditsModeProperty := func(creditsMode bool, totalCredits float64, dailyAnalysis uint16) bool {
		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:   creditsMode,
			TotalCredits:  totalCredits,
			DailyAnalysis: int(dailyAnalysis),
		}

		result := client.IsCreditsMode()

		// IsCreditsMode() should return true iff CreditsMode is true
		return result == creditsMode
	}

	if err := quick.Check(isCreditsModeProperty, config); err != nil {
		t.Errorf("Property 1a (IsCreditsMode consistency) failed: %v", err)
	}

	// Property 1b: When CreditsMode is false and DailyAnalysis > 0, it is Daily_Limit_Mode
	// (CanAnalyze enforces daily limit, not credits)
	dailyLimitModeProperty := func(dailyAnalysis uint16) bool {
		// Ensure dailyAnalysis > 0 for Daily_Limit_Mode
		da := int(dailyAnalysis%100) + 1 // 1..100

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:   false,
			TotalCredits:  0,
			DailyAnalysis: da,
		}

		// Should NOT be credits mode
		if client.IsCreditsMode() {
			return false
		}

		// GetCreditsStatus should report not credits mode
		_, _, isCredits := client.GetCreditsStatus()
		if isCredits {
			return false
		}

		return true
	}

	if err := quick.Check(dailyLimitModeProperty, config); err != nil {
		t.Errorf("Property 1b (Daily_Limit_Mode detection) failed: %v", err)
	}

	// Property 1c: When CreditsMode is false and DailyAnalysis == 0, it is unlimited mode
	// (CanAnalyze always returns true)
	unlimitedModeProperty := func(totalCredits float64) bool {
		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:   false,
			TotalCredits:  0,
			DailyAnalysis: 0,
		}

		// Should NOT be credits mode
		if client.IsCreditsMode() {
			return false
		}

		// CanAnalyze should always allow (unlimited)
		allowed, _ := client.CanAnalyze()
		if !allowed {
			return false
		}

		return true
	}

	if err := quick.Check(unlimitedModeProperty, config); err != nil {
		t.Errorf("Property 1c (Unlimited mode detection) failed: %v", err)
	}

	// Property 1d: Nil data means not credits mode (no activation)
	nilDataProperty := func() bool {
		client := NewLicenseClient(nil)
		// data is nil by default
		return !client.IsCreditsMode()
	}

	// Run nil data check (not random, but validates the nil guard)
	if !nilDataProperty() {
		t.Error("Property 1d: IsCreditsMode() should return false when data is nil")
	}
}

// Feature: license-credits-support, Property 3: CanAnalyze credits 阈值检查
// **Validates: Requirements 3.1, 3.2**
//
// For any ActivationData in Credits_Mode (total_credits > 0),
// CanAnalyze() returns true if and only if total_credits - used_credits >= 1.5.
func TestPropertyCanAnalyzeCreditsThreshold(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
	}

	// Property 3: CanAnalyze credits threshold check
	// Generate random total_credits (positive, so we're in credits mode with a limit)
	// and random used_credits, then verify CanAnalyze() matches the threshold condition.
	canAnalyzeThresholdProperty := func(totalRaw, usedRaw uint16) bool {
		// Map to reasonable positive float64 values
		// totalCredits must be > 0 to be in credits mode with a limit
		totalCredits := float64(totalRaw%1000) + CreditsPerAnalysis // Ensure > 0, range [1.5, 1000.5]
		usedCredits := float64(usedRaw%1000)                       // range [0, 999]

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
		}

		allowed, msg := client.CanAnalyze()
		remaining := totalCredits - usedCredits
		if remaining < 0 {
			remaining = 0
		}
		expectedAllowed := remaining >= CreditsPerAnalysis

		// CanAnalyze should return true iff remaining >= CreditsPerAnalysis
		if allowed != expectedAllowed {
			t.Logf("totalCredits=%.1f, usedCredits=%.1f, remaining=%.1f, allowed=%v, expected=%v",
				totalCredits, usedCredits, remaining, allowed, expectedAllowed)
			return false
		}

		// When not allowed, message should be non-empty
		if !allowed && msg == "" {
			t.Logf("CanAnalyze returned false but message is empty")
			return false
		}

		// When allowed, message should be empty
		if allowed && msg != "" {
			t.Logf("CanAnalyze returned true but message is non-empty: %s", msg)
			return false
		}

		return true
	}

	if err := quick.Check(canAnalyzeThresholdProperty, config); err != nil {
		t.Errorf("Property 3 (CanAnalyze credits threshold) failed: %v", err)
	}

	// Property 3b: Boundary test - exactly at threshold
	// When remaining == CreditsPerAnalysis exactly, should be allowed
	boundaryProperty := func(baseCredits uint16) bool {
		totalCredits := float64(baseCredits%500) + CreditsPerAnalysis // Ensure total > CreditsPerAnalysis
		usedCredits := totalCredits - CreditsPerAnalysis              // Exactly at threshold

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
		}

		allowed, _ := client.CanAnalyze()
		return allowed // Should always be true at exact threshold
	}

	if err := quick.Check(boundaryProperty, config); err != nil {
		t.Errorf("Property 3b (CanAnalyze boundary at exact threshold) failed: %v", err)
	}

	// Property 3c: Just below threshold - should be denied
	belowThresholdProperty := func(baseCredits uint16) bool {
		totalCredits := float64(baseCredits%500) + CreditsPerAnalysis + 1 // Ensure total > CreditsPerAnalysis
		// Set used so remaining is just below threshold (remaining = 1.4)
		usedCredits := totalCredits - CreditsPerAnalysis + 0.1

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
		}

		allowed, _ := client.CanAnalyze()
		return !allowed // Should always be denied just below threshold
	}

	if err := quick.Check(belowThresholdProperty, config); err != nil {
		t.Errorf("Property 3c (CanAnalyze just below threshold) failed: %v", err)
	}
}

// Feature: license-credits-support, Property 4: Credits 模式绕过每日限制
// **Validates: Requirements 3.3**
//
// For any LicenseClient in Credits_Mode, even when daily_analysis > 0 and
// the daily analysis count has reached the daily_analysis limit, as long as
// remaining credits are sufficient, CanAnalyze() should still return true.
func TestPropertyCreditsModeBypassesDailyLimit(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
	}

	// Property 4a: Credits mode with sufficient credits bypasses daily limit
	// Generate random daily limits and analysis counts where count >= limit (daily limit reached),
	// but credits are sufficient — CanAnalyze() should return true.
	bypassDailyLimitProperty := func(dailyAnalysisRaw, analysisCountRaw, totalCreditsRaw, usedCreditsRaw uint16) bool {
		// Ensure dailyAnalysis > 0 (a real daily limit is configured)
		dailyAnalysis := int(dailyAnalysisRaw%100) + 1 // range [1, 100]

		// Ensure analysisCount >= dailyAnalysis (daily limit reached)
		analysisCount := dailyAnalysis + int(analysisCountRaw%50) // at or above limit

		// Ensure sufficient credits: totalCredits - usedCredits >= CreditsPerAnalysis
		totalCredits := float64(totalCreditsRaw%1000) + CreditsPerAnalysis + 1 // range [2.5, 1001.5]
		usedCredits := float64(usedCreditsRaw%1000)
		// Clamp usedCredits so remaining >= CreditsPerAnalysis
		if totalCredits-usedCredits < CreditsPerAnalysis {
			usedCredits = totalCredits - CreditsPerAnalysis
		}

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:   true,
			TotalCredits:  totalCredits,
			UsedCredits:   usedCredits,
			DailyAnalysis: dailyAnalysis,
		}
		// Simulate that the daily limit has been reached
		client.analysisCount = analysisCount
		client.analysisDate = "2024-01-01" // a past date won't reset; but we set count directly

		allowed, msg := client.CanAnalyze()

		// In credits mode with sufficient credits, should always be allowed
		// regardless of daily limit being reached
		if !allowed {
			t.Logf("FAIL: dailyAnalysis=%d, analysisCount=%d, totalCredits=%.1f, usedCredits=%.1f, remaining=%.1f, msg=%s",
				dailyAnalysis, analysisCount, totalCredits, usedCredits, totalCredits-usedCredits, msg)
			return false
		}

		// Message should be empty when allowed
		if msg != "" {
			t.Logf("FAIL: allowed but msg non-empty: %s", msg)
			return false
		}

		return true
	}

	if err := quick.Check(bypassDailyLimitProperty, config); err != nil {
		t.Errorf("Property 4a (Credits mode bypasses daily limit with sufficient credits) failed: %v", err)
	}

	// Property 4b: Verify that the same configuration WITHOUT credits mode
	// would be blocked by the daily limit. This confirms the bypass is real.
	dailyLimitBlocksWithoutCreditsMode := func(dailyAnalysisRaw uint16) bool {
		dailyAnalysis := int(dailyAnalysisRaw%100) + 1 // range [1, 100]
		today := time.Now().Format("2006-01-02")

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:   false,
			TotalCredits:  0,
			DailyAnalysis: dailyAnalysis,
		}
		// Set analysis count to exactly the daily limit, using today's date
		// so CanAnalyze() won't reset the count
		client.analysisCount = dailyAnalysis
		client.analysisDate = today

		allowed, _ := client.CanAnalyze()

		// Without credits mode, daily limit should block
		if allowed {
			t.Logf("FAIL: daily limit mode should block when count=%d >= limit=%d", dailyAnalysis, dailyAnalysis)
			return false
		}

		return true
	}

	if err := quick.Check(dailyLimitBlocksWithoutCreditsMode, config); err != nil {
		t.Errorf("Property 4b (Daily limit blocks without credits mode) failed: %v", err)
	}

	// Property 4c: Credits mode bypasses daily limit even with varying daily limits
	// and analysis counts well above the limit
	highCountBypassProperty := func(dailyAnalysisRaw, multiplierRaw, totalCreditsRaw uint16) bool {
		dailyAnalysis := int(dailyAnalysisRaw%50) + 1       // range [1, 50]
		multiplier := int(multiplierRaw%10) + 2              // range [2, 11]
		analysisCount := dailyAnalysis * multiplier          // well above limit
		totalCredits := float64(totalCreditsRaw%500) + 100.0 // range [100, 599]

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:   true,
			TotalCredits:  totalCredits,
			UsedCredits:   0, // plenty of credits remaining
			DailyAnalysis: dailyAnalysis,
		}
		client.analysisCount = analysisCount
		client.analysisDate = time.Now().Format("2006-01-02")

		allowed, _ := client.CanAnalyze()
		return allowed
	}

	if err := quick.Check(highCountBypassProperty, config); err != nil {
		t.Errorf("Property 4c (Credits mode bypasses high daily count) failed: %v", err)
	}
}

// Feature: license-credits-support, Property 5: IncrementAnalysis credits 扣减精度
// **Validates: Requirements 4.1**
//
// For any LicenseClient in Credits_Mode, calling IncrementAnalysis() should
// increase used_credits by exactly CreditsPerAnalysis (1.5).
func TestPropertyIncrementAnalysisCreditsDeduction(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
	}

	// Property 5a: Single IncrementAnalysis call increases used_credits by exactly CreditsPerAnalysis
	singleDeductionProperty := func(totalCreditsRaw, usedCreditsRaw uint16) bool {
		// Ensure we're in credits mode with a limit (totalCredits > 0)
		totalCredits := float64(totalCreditsRaw%1000) + CreditsPerAnalysis + 1 // range [2.5, 1001.5]
		usedCredits := float64(usedCreditsRaw % 500)                           // range [0, 499]

		// Ensure there are enough remaining credits to deduct
		if totalCredits-usedCredits < CreditsPerAnalysis {
			usedCredits = 0
		}

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
		}

		before := client.data.UsedCredits
		client.IncrementAnalysis()
		after := client.data.UsedCredits

		increment := after - before
		if increment != CreditsPerAnalysis {
			t.Logf("FAIL: totalCredits=%.1f, usedCredits(before)=%.1f, usedCredits(after)=%.1f, increment=%.1f, expected=%.1f",
				totalCredits, before, after, increment, CreditsPerAnalysis)
			return false
		}

		return true
	}

	if err := quick.Check(singleDeductionProperty, config); err != nil {
		t.Errorf("Property 5a (Single IncrementAnalysis deduction precision) failed: %v", err)
	}

	// Property 5b: Multiple IncrementAnalysis calls each deduct exactly CreditsPerAnalysis
	multipleDeductionProperty := func(totalCreditsRaw uint16, numCallsRaw uint8) bool {
		// Number of calls: 1..10
		numCalls := int(numCallsRaw%10) + 1
		// Ensure enough total credits for all calls
		totalCredits := float64(numCalls)*CreditsPerAnalysis + float64(totalCreditsRaw%500) + 1

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  0,
		}

		for i := 0; i < numCalls; i++ {
			before := client.data.UsedCredits
			client.IncrementAnalysis()
			after := client.data.UsedCredits

			increment := after - before
			if increment != CreditsPerAnalysis {
				t.Logf("FAIL: call %d/%d, before=%.1f, after=%.1f, increment=%.1f, expected=%.1f",
					i+1, numCalls, before, after, increment, CreditsPerAnalysis)
				return false
			}
		}

		// Also verify total deduction
		expectedTotal := float64(numCalls) * CreditsPerAnalysis
		if client.data.UsedCredits != expectedTotal {
			t.Logf("FAIL: after %d calls, total UsedCredits=%.1f, expected=%.1f",
				numCalls, client.data.UsedCredits, expectedTotal)
			return false
		}

		return true
	}

	if err := quick.Check(multipleDeductionProperty, config); err != nil {
		t.Errorf("Property 5b (Multiple IncrementAnalysis deductions precision) failed: %v", err)
	}

	// Property 5c: IncrementAnalysis with random initial used_credits still deducts exactly CreditsPerAnalysis
	randomInitialCreditsProperty := func(initialUsedRaw uint16) bool {
		initialUsed := float64(initialUsedRaw%1000) + 0.5 // range [0.5, 1000.5] — non-round values
		totalCredits := initialUsed + CreditsPerAnalysis + 100 // ensure plenty of remaining credits

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  initialUsed,
		}

		before := client.data.UsedCredits
		client.IncrementAnalysis()
		after := client.data.UsedCredits

		increment := after - before
		if increment != CreditsPerAnalysis {
			t.Logf("FAIL: initialUsed=%.1f, after=%.1f, increment=%.1f, expected=%.1f",
				before, after, increment, CreditsPerAnalysis)
			return false
		}

		return true
	}

	if err := quick.Check(randomInitialCreditsProperty, config); err != nil {
		t.Errorf("Property 5c (Random initial used_credits deduction precision) failed: %v", err)
	}
}

// Feature: license-credits-support, Property 6: GetActivationStatus credits 字段一致性
// **Validates: Requirements 5.1, 5.2, 5.3**
//
// For any activated LicenseClient, GetCreditsStatus() returned credits_mode field
// should be consistent with IsCreditsMode(), and total_credits and used_credits
// should be consistent with the ActivationData fields.
// (We test at the LicenseClient level since GetActivationStatus on App simply
// delegates to GetCreditsStatus for credits fields.)
func TestPropertyGetActivationStatusCreditsConsistency(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
	}

	// Property 6a: GetCreditsStatus credits_mode is consistent with IsCreditsMode()
	// and total_credits/used_credits match ActivationData fields
	creditsConsistencyProperty := func(creditsMode bool, totalCreditsRaw, usedCreditsRaw uint16) bool {
		totalCredits := float64(totalCreditsRaw % 1000)
		usedCredits := float64(usedCreditsRaw % 1000)

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  creditsMode,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
		}

		// Get values from GetCreditsStatus
		returnedTotal, returnedUsed, returnedIsCreditsMode := client.GetCreditsStatus()

		// Get value from IsCreditsMode
		isCreditsMode := client.IsCreditsMode()

		// credits_mode from GetCreditsStatus must match IsCreditsMode()
		if returnedIsCreditsMode != isCreditsMode {
			t.Logf("FAIL: GetCreditsStatus credits_mode=%v != IsCreditsMode()=%v",
				returnedIsCreditsMode, isCreditsMode)
			return false
		}

		// total_credits must match ActivationData.TotalCredits
		if returnedTotal != totalCredits {
			t.Logf("FAIL: GetCreditsStatus totalCredits=%.1f != ActivationData.TotalCredits=%.1f",
				returnedTotal, totalCredits)
			return false
		}

		// used_credits must match ActivationData.UsedCredits
		if returnedUsed != usedCredits {
			t.Logf("FAIL: GetCreditsStatus usedCredits=%.1f != ActivationData.UsedCredits=%.1f",
				returnedUsed, usedCredits)
			return false
		}

		return true
	}

	if err := quick.Check(creditsConsistencyProperty, config); err != nil {
		t.Errorf("Property 6a (GetCreditsStatus consistency with ActivationData and IsCreditsMode) failed: %v", err)
	}

	// Property 6b: When CreditsMode is true, credits_mode should be true (Requirement 5.2)
	// When CreditsMode is false, credits_mode should be false (Requirement 5.3)
	creditsModeFieldProperty := func(totalCreditsRaw, usedCreditsRaw uint16, dailyAnalysisRaw uint8) bool {
		totalCredits := float64(totalCreditsRaw % 1000)
		usedCredits := float64(usedCreditsRaw % 1000)
		dailyAnalysis := int(dailyAnalysisRaw % 100)

		// Test with CreditsMode = true
		clientCredits := NewLicenseClient(nil)
		clientCredits.data = &ActivationData{
			CreditsMode:   true,
			TotalCredits:  totalCredits,
			UsedCredits:   usedCredits,
			DailyAnalysis: dailyAnalysis,
		}

		_, _, isCreditsTrue := clientCredits.GetCreditsStatus()
		if !isCreditsTrue {
			t.Logf("FAIL: CreditsMode=true but GetCreditsStatus returned isCreditsMode=false")
			return false
		}

		// Test with CreditsMode = false
		clientDaily := NewLicenseClient(nil)
		clientDaily.data = &ActivationData{
			CreditsMode:   false,
			TotalCredits:  0,
			UsedCredits:   0,
			DailyAnalysis: dailyAnalysis,
		}

		_, _, isCreditsFalse := clientDaily.GetCreditsStatus()
		if isCreditsFalse {
			t.Logf("FAIL: CreditsMode=false but GetCreditsStatus returned isCreditsMode=true")
			return false
		}

		return true
	}

	if err := quick.Check(creditsModeFieldProperty, config); err != nil {
		t.Errorf("Property 6b (credits_mode field matches CreditsMode setting) failed: %v", err)
	}

	// Property 6c: After IncrementAnalysis, GetCreditsStatus reflects updated used_credits
	// This verifies that the status query is always consistent with internal state
	// even after mutations.
	postIncrementConsistencyProperty := func(totalCreditsRaw, usedCreditsRaw uint16) bool {
		// Ensure enough credits for at least one analysis
		totalCredits := float64(totalCreditsRaw%500) + CreditsPerAnalysis + 10
		usedCredits := float64(usedCreditsRaw % 200)
		// Clamp so there's room for at least one deduction
		if totalCredits-usedCredits < CreditsPerAnalysis {
			usedCredits = 0
		}

		client := NewLicenseClient(nil)
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
		}

		// Perform an analysis
		client.IncrementAnalysis()

		// Now check consistency
		returnedTotal, returnedUsed, returnedIsCreditsMode := client.GetCreditsStatus()
		isCreditsMode := client.IsCreditsMode()

		// credits_mode must still be consistent
		if returnedIsCreditsMode != isCreditsMode {
			t.Logf("FAIL: After IncrementAnalysis, credits_mode mismatch: GetCreditsStatus=%v, IsCreditsMode=%v",
				returnedIsCreditsMode, isCreditsMode)
			return false
		}

		// total_credits should not change after IncrementAnalysis
		if returnedTotal != totalCredits {
			t.Logf("FAIL: After IncrementAnalysis, totalCredits changed: %.1f -> %.1f",
				totalCredits, returnedTotal)
			return false
		}

		// used_credits should reflect the deduction
		expectedUsed := usedCredits + CreditsPerAnalysis
		if returnedUsed != expectedUsed {
			t.Logf("FAIL: After IncrementAnalysis, usedCredits=%.1f, expected=%.1f",
				returnedUsed, expectedUsed)
			return false
		}

		// Also verify it matches the internal data directly
		if returnedUsed != client.data.UsedCredits {
			t.Logf("FAIL: GetCreditsStatus usedCredits=%.1f != data.UsedCredits=%.1f",
				returnedUsed, client.data.UsedCredits)
			return false
		}

		return true
	}

	if err := quick.Check(postIncrementConsistencyProperty, config); err != nil {
		t.Errorf("Property 6c (GetCreditsStatus consistency after IncrementAnalysis) failed: %v", err)
	}

	// Property 6d: Nil data returns zero values (not activated case)
	nilDataProperty := func() bool {
		client := NewLicenseClient(nil)
		// data is nil by default

		total, used, isCredits := client.GetCreditsStatus()
		isMode := client.IsCreditsMode()

		if total != 0 || used != 0 || isCredits || isMode {
			t.Logf("FAIL: nil data should return (0, 0, false) and IsCreditsMode=false, got (%.1f, %.1f, %v, %v)",
				total, used, isCredits, isMode)
			return false
		}

		return true
	}

	if !nilDataProperty() {
		t.Error("Property 6d: nil data should return zero values for credits status")
	}
}

// Feature: license-credits-support, Property 2: Credits 持久化往返一致性
// **Validates: Requirements 2.2, 2.3**
//
// For any LicenseClient in Credits_Mode, after performing some IncrementAnalysis() calls
// followed by SaveActivationData(), then loading via LoadActivationData(), the restored
// used_credits should equal the value before saving.
func TestPropertyCreditsPersistenceRoundTrip(t *testing.T) {
	config := &quick.Config{
		MaxCount: 200,
	}

	// Property 2a: Save/Load round-trip preserves used_credits exactly
	// Generate random credits values, save, load into a new client, and verify used_credits matches.
	roundTripProperty := func(totalCreditsRaw, usedCreditsRaw uint16) bool {
		// Map to reasonable positive float64 values
		// totalCredits must be > 0 to be in credits mode
		totalCredits := float64(totalCreditsRaw%1000) + 1.5 // 1.5..1000.5
		usedCredits := float64(usedCreditsRaw%1000) * 0.5   // 0..499.5 (multiples of 0.5)

		// Ensure usedCredits <= totalCredits
		if usedCredits > totalCredits {
			usedCredits = math.Mod(usedCredits, totalCredits+1)
		}

		sn := "TEST-SN-ROUNDTRIP-001"

		// Create a temp directory for this test iteration
		tmpDir, err := os.MkdirTemp("", "license-pbt-roundtrip-*")
		if err != nil {
			t.Logf("FAIL: could not create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tmpDir)

		// Create the saving client with credits mode data
		saveClient := NewLicenseClient(nil)
		saveClient.SetDataDir(tmpDir)
		saveClient.sn = sn
		saveClient.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
			ExpiresAt:    time.Now().Add(24 * time.Hour).Format(time.RFC3339), // Not expired
			DailyAnalysis: 10,
		}

		// Save activation data
		if err := saveClient.SaveActivationData(); err != nil {
			t.Logf("FAIL: SaveActivationData failed: %v", err)
			return false
		}

		// Record the used_credits value before loading
		savedUsedCredits := saveClient.data.UsedCredits

		// Create a new client and load the data
		loadClient := NewLicenseClient(nil)
		loadClient.SetDataDir(tmpDir)

		if err := loadClient.LoadActivationData(sn); err != nil {
			t.Logf("FAIL: LoadActivationData failed: %v", err)
			return false
		}

		// Verify used_credits matches
		if loadClient.data.UsedCredits != savedUsedCredits {
			t.Logf("FAIL: used_credits mismatch: saved=%.2f, loaded=%.2f",
				savedUsedCredits, loadClient.data.UsedCredits)
			return false
		}

		return true
	}

	if err := quick.Check(roundTripProperty, config); err != nil {
		t.Errorf("Property 2a (Save/Load round-trip preserves used_credits) failed: %v", err)
	}

	// Property 2b: After IncrementAnalysis calls, save/load round-trip preserves the accumulated used_credits
	// This tests the full workflow: start with some credits, call IncrementAnalysis N times,
	// save, load, and verify the accumulated used_credits is preserved.
	incrementAndRoundTripProperty := func(totalCreditsRaw uint16, incrementCountRaw uint8) bool {
		// Ensure enough total credits for the increments
		incrementCount := int(incrementCountRaw%10) + 1 // 1..10 increments
		minCreditsNeeded := float64(incrementCount) * CreditsPerAnalysis
		totalCredits := float64(totalCreditsRaw%500) + minCreditsNeeded + 10 // Enough room

		sn := "TEST-SN-ROUNDTRIP-002"

		// Create a temp directory for this test iteration
		tmpDir, err := os.MkdirTemp("", "license-pbt-increment-roundtrip-*")
		if err != nil {
			t.Logf("FAIL: could not create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tmpDir)

		// Create the client with credits mode data (starting with 0 used credits)
		client := NewLicenseClient(nil)
		client.SetDataDir(tmpDir)
		client.sn = sn
		client.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  0,
			ExpiresAt:    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			DailyAnalysis: 10,
		}

		// Call IncrementAnalysis N times (each call also saves to disk)
		for i := 0; i < incrementCount; i++ {
			client.IncrementAnalysis()
		}

		// Record the expected used_credits
		expectedUsedCredits := float64(incrementCount) * CreditsPerAnalysis

		// Verify the client's internal state first
		if client.data.UsedCredits != expectedUsedCredits {
			t.Logf("FAIL: after %d increments, client.data.UsedCredits=%.2f, expected=%.2f",
				incrementCount, client.data.UsedCredits, expectedUsedCredits)
			return false
		}

		// Now load into a new client and verify persistence
		loadClient := NewLicenseClient(nil)
		loadClient.SetDataDir(tmpDir)

		if err := loadClient.LoadActivationData(sn); err != nil {
			t.Logf("FAIL: LoadActivationData failed: %v", err)
			return false
		}

		// Verify used_credits matches the accumulated value
		if loadClient.data.UsedCredits != expectedUsedCredits {
			t.Logf("FAIL: after %d increments, loaded used_credits=%.2f, expected=%.2f",
				incrementCount, loadClient.data.UsedCredits, expectedUsedCredits)
			return false
		}

		return true
	}

	if err := quick.Check(incrementAndRoundTripProperty, config); err != nil {
		t.Errorf("Property 2b (IncrementAnalysis + Save/Load round-trip) failed: %v", err)
	}

	// Property 2c: Round-trip preserves total_credits alongside used_credits
	// Verifies that the full credits state (both total and used) survives the round-trip.
	fullCreditsStateRoundTripProperty := func(totalCreditsRaw, usedCreditsRaw uint16) bool {
		totalCredits := float64(totalCreditsRaw%1000) + 1.5
		usedCredits := float64(usedCreditsRaw % 500)

		if usedCredits > totalCredits {
			usedCredits = math.Mod(usedCredits, totalCredits+1)
		}

		sn := "TEST-SN-ROUNDTRIP-003"

		tmpDir, err := os.MkdirTemp("", "license-pbt-fullstate-*")
		if err != nil {
			t.Logf("FAIL: could not create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tmpDir)

		saveClient := NewLicenseClient(nil)
		saveClient.SetDataDir(tmpDir)
		saveClient.sn = sn
		saveClient.data = &ActivationData{
			CreditsMode:  true,
			TotalCredits: totalCredits,
			UsedCredits:  usedCredits,
			ExpiresAt:    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			DailyAnalysis: 5,
		}

		if err := saveClient.SaveActivationData(); err != nil {
			t.Logf("FAIL: SaveActivationData failed: %v", err)
			return false
		}

		loadClient := NewLicenseClient(nil)
		loadClient.SetDataDir(tmpDir)

		if err := loadClient.LoadActivationData(sn); err != nil {
			t.Logf("FAIL: LoadActivationData failed: %v", err)
			return false
		}

		// Verify total_credits is preserved (stored in ActivationData.Data)
		if loadClient.data.TotalCredits != totalCredits {
			t.Logf("FAIL: total_credits mismatch: saved=%.2f, loaded=%.2f",
				totalCredits, loadClient.data.TotalCredits)
			return false
		}

		// Verify used_credits is preserved
		if loadClient.data.UsedCredits != usedCredits {
			t.Logf("FAIL: used_credits mismatch: saved=%.2f, loaded=%.2f",
				usedCredits, loadClient.data.UsedCredits)
			return false
		}

		// Verify credits_mode is preserved
		if loadClient.data.CreditsMode != true {
			t.Logf("FAIL: CreditsMode not preserved after round-trip")
			return false
		}

		return true
	}

	if err := quick.Check(fullCreditsStateRoundTripProperty, config); err != nil {
		t.Errorf("Property 2c (Full credits state round-trip) failed: %v", err)
	}
}
