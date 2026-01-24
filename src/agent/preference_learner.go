package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UserPreferences stores learned user preferences and behaviors
type UserPreferences struct {
	PreferredChartTypes map[string]string  `json:"preferred_chart_types"` // dimension -> chart type
	ColorScheme         string             `json:"color_scheme"`          // "light", "dark", "auto"
	DefaultFilters      map[string]string  `json:"default_filters"`       // dimension -> default value
	FrequentMetrics     map[string]int     `json:"frequent_metrics"`      // metric name -> usage count
	ChartUsageHistory   []ChartUsageRecord `json:"chart_usage_history"`   // Track usage patterns
}

// ChartUsageRecord tracks when and how a chart type was used
type ChartUsageRecord struct {
	Dimension string `json:"dimension"`
	ChartType string `json:"chart_type"` // "bar", "line", "pie", "scatter", etc.
	Timestamp int64  `json:"timestamp"`
}

// BusinessRule represents a business logic rule
type BusinessRule struct {
	RuleID      string `json:"rule_id"`
	Name        string `json:"name"`
	Definition  string `json:"definition"`
	Calculation string `json:"calculation,omitempty"` // Formula or logic
	CreatedAt   int64  `json:"created_at"`
}

// Benchmark represents industry or custom benchmarks
type Benchmark struct {
	BenchmarkID string  `json:"benchmark_id"`
	Metric      string  `json:"metric"`     // e.g., "毛利率"
	Threshold   float64 `json:"threshold"`  // e.g., 0.20 (20%)
	Industry    string  `json:"industry"`   // e.g., "retail", "manufacturing"
	Type        string  `json:"type"`       // "target", "warning", "critical"
	CreatedAt   int64   `json:"created_at"`
}

// IntentSelectionRecord 意图选择记录
// Records user's intent selection for preference learning
// Validates: Requirements 2.1
type IntentSelectionRecord struct {
	DataSourceID string    `json:"data_source_id"`
	IntentType   string    `json:"intent_type"`  // trend, comparison, distribution, etc.
	IntentTitle  string    `json:"intent_title"`
	SelectCount  int       `json:"select_count"`
	LastSelected time.Time `json:"last_selected"`
}

// IntentSelectionsStore stores intent selections by data source
type IntentSelectionsStore struct {
	Selections map[string][]IntentSelectionRecord `json:"selections"` // data_source_id -> records
}

// PreferenceLearner learns and manages user preferences
type PreferenceLearner struct {
	dataDir          string
	preferences      UserPreferences
	rules            []BusinessRule
	benchmarks       []Benchmark
	intentSelections IntentSelectionsStore // Intent selection records for preference learning
	mu               sync.RWMutex
}

// NewPreferenceLearner creates a new preference learner
func NewPreferenceLearner(dataDir string) *PreferenceLearner {
	prefsDir := filepath.Join(dataDir, "preferences")
	_ = os.MkdirAll(prefsDir, 0755)

	learner := &PreferenceLearner{
		dataDir: dataDir,
		preferences: UserPreferences{
			PreferredChartTypes: make(map[string]string),
			DefaultFilters:      make(map[string]string),
			FrequentMetrics:     make(map[string]int),
			ChartUsageHistory:   []ChartUsageRecord{},
		},
		rules:      []BusinessRule{},
		benchmarks: []Benchmark{},
		intentSelections: IntentSelectionsStore{
			Selections: make(map[string][]IntentSelectionRecord),
		},
	}

	learner.loadPreferences()
	learner.loadIntentSelections()
	return learner
}

// TrackChartUsage records chart type usage for learning
func (p *PreferenceLearner) TrackChartUsage(dimension, chartType string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Add to usage history
	record := ChartUsageRecord{
		Dimension: dimension,
		ChartType: chartType,
		Timestamp: 0, // Set by caller if needed
	}
	p.preferences.ChartUsageHistory = append(p.preferences.ChartUsageHistory, record)

	// Keep only last 100 records
	if len(p.preferences.ChartUsageHistory) > 100 {
		p.preferences.ChartUsageHistory = p.preferences.ChartUsageHistory[len(p.preferences.ChartUsageHistory)-100:]
	}

	// Update preferred chart type if this dimension has been used 3+ times with same type
	count := 0
	for i := len(p.preferences.ChartUsageHistory) - 1; i >= 0 && i >= len(p.preferences.ChartUsageHistory)-10; i-- {
		if p.preferences.ChartUsageHistory[i].Dimension == dimension && p.preferences.ChartUsageHistory[i].ChartType == chartType {
			count++
		}
	}

	if count >= 3 {
		p.preferences.PreferredChartTypes[dimension] = chartType
	}

	return p.savePreferences()
}

// GetPreferredChartType returns the preferred chart type for a dimension
func (p *PreferenceLearner) GetPreferredChartType(dimension string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if chartType, exists := p.preferences.PreferredChartTypes[dimension]; exists {
		return chartType
	}
	return "" // No preference learned yet
}

// TrackMetricUsage increments usage count for a metric
func (p *PreferenceLearner) TrackMetricUsage(metric string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.preferences.FrequentMetrics[metric]++
	return p.savePreferences()
}

// GetFrequentMetrics returns top N most frequently used metrics
func (p *PreferenceLearner) GetFrequentMetrics(topN int) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Sort by usage count
	type metricCount struct {
		Metric string
		Count  int
	}

	metrics := make([]metricCount, 0, len(p.preferences.FrequentMetrics))
	for metric, count := range p.preferences.FrequentMetrics {
		metrics = append(metrics, metricCount{Metric: metric, Count: count})
	}

	// Simple bubble sort (fine for small lists)
	for i := 0; i < len(metrics); i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[j].Count > metrics[i].Count {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	// Return top N
	result := []string{}
	for i := 0; i < topN && i < len(metrics); i++ {
		result = append(result, metrics[i].Metric)
	}

	return result
}

// AddBusinessRule adds a new business rule
func (p *PreferenceLearner) AddBusinessRule(rule BusinessRule) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.rules = append(p.rules, rule)
	return p.saveRules()
}

// GetBusinessRules returns all business rules
func (p *PreferenceLearner) GetBusinessRules() []BusinessRule {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.rules
}

// AddBenchmark adds a new benchmark
func (p *PreferenceLearner) AddBenchmark(benchmark Benchmark) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.benchmarks = append(p.benchmarks, benchmark)
	return p.saveBenchmarks()
}

// CheckBenchmark checks if a value meets benchmark criteria
func (p *PreferenceLearner) CheckBenchmark(metric string, value float64) *BenchmarkAlert {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, b := range p.benchmarks {
		if b.Metric == metric {
			if b.Type == "target" && value < b.Threshold {
				return &BenchmarkAlert{
					Metric:    metric,
					Value:     value,
					Threshold: b.Threshold,
					Type:      "below_target",
					Message:   "低于目标值",
				}
			} else if b.Type == "warning" && value < b.Threshold {
				return &BenchmarkAlert{
					Metric:    metric,
					Value:     value,
					Threshold: b.Threshold,
					Type:      "warning",
					Message:   "接近警告阈值",
				}
			}
		}
	}

	return nil
}

// BenchmarkAlert represents a benchmark violation
type BenchmarkAlert struct {
	Metric    string  `json:"metric"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Type      string  `json:"type"`
	Message   string  `json:"message"`
}

// SetColorScheme sets the user's preferred color scheme
func (p *PreferenceLearner) SetColorScheme(scheme string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.preferences.ColorScheme = scheme
	return p.savePreferences()
}

// GetColorScheme returns the user's preferred color scheme
func (p *PreferenceLearner) GetColorScheme() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.preferences.ColorScheme == "" {
		return "auto" // Default
	}
	return p.preferences.ColorScheme
}

// Internal persistence methods

func (p *PreferenceLearner) getPreferencesPath() string {
	return filepath.Join(p.dataDir, "preferences", "user_preferences.json")
}

func (p *PreferenceLearner) getRulesPath() string {
	return filepath.Join(p.dataDir, "preferences", "business_rules.json")
}

func (p *PreferenceLearner) getBenchmarksPath() string {
	return filepath.Join(p.dataDir, "preferences", "benchmarks.json")
}

func (p *PreferenceLearner) loadPreferences() {
	path := p.getPreferencesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist yet
	}

	_ = json.Unmarshal(data, &p.preferences)
}

func (p *PreferenceLearner) savePreferences() error {
	path := p.getPreferencesPath()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(p.preferences, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (p *PreferenceLearner) saveRules() error {
	path := p.getRulesPath()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(p.rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (p *PreferenceLearner) saveBenchmarks() error {
	path := p.getBenchmarksPath()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(p.benchmarks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Intent selection storage methods

func (p *PreferenceLearner) getIntentSelectionsPath() string {
	return filepath.Join(p.dataDir, "preferences", "intent_selections.json")
}

func (p *PreferenceLearner) loadIntentSelections() {
	path := p.getIntentSelectionsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist yet
	}

	_ = json.Unmarshal(data, &p.intentSelections)

	// Ensure the map is initialized
	if p.intentSelections.Selections == nil {
		p.intentSelections.Selections = make(map[string][]IntentSelectionRecord)
	}
}

func (p *PreferenceLearner) saveIntentSelections() error {
	path := p.getIntentSelectionsPath()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(p.intentSelections, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetIntentSelections returns all intent selection records for a data source
// This is used for preference learning and ranking
func (p *PreferenceLearner) GetIntentSelections(dataSourceID string) []IntentSelectionRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if records, exists := p.intentSelections.Selections[dataSourceID]; exists {
		// Return a copy to avoid external modification
		result := make([]IntentSelectionRecord, len(records))
		copy(result, records)
		return result
	}
	return []IntentSelectionRecord{}
}

// MinSelectionsForPreference is the minimum number of total selections required
// before preference-based ranking is applied. Below this threshold, default sorting is used.
// Validates: Requirements 2.6
const MinSelectionsForPreference = 5

// TrackIntentSelection records user's intent selection for preference learning
// This method tracks which intents users select to learn their preferences over time.
// Selections are tracked per data source to support different analysis patterns for different datasets.
//
// Parameters:
//   - dataSourceID: the ID of the data source where the intent was selected
//   - intent: the IntentSuggestion that was selected by the user
//
// Returns error if saving the selection fails
//
// Validates: Requirements 2.1, 2.2, 2.5
func (p *PreferenceLearner) TrackIntentSelection(dataSourceID string, intent IntentSuggestion) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Ensure the selections map is initialized
	if p.intentSelections.Selections == nil {
		p.intentSelections.Selections = make(map[string][]IntentSelectionRecord)
	}

	// Get existing records for this data source
	records := p.intentSelections.Selections[dataSourceID]

	// Determine the intent type from the suggestion
	// Use the Title as the intent type identifier since it represents the analysis type
	intentType := intent.Title
	if intentType == "" {
		intentType = "unknown"
	}

	// Look for an existing record with the same intent type
	found := false
	for i := range records {
		if records[i].IntentType == intentType {
			// Increment the selection count (Requirement 2.2)
			records[i].SelectCount++
			// Update the last selected timestamp
			records[i].LastSelected = time.Now()
			// Update the title in case it changed
			records[i].IntentTitle = intent.Title
			found = true
			break
		}
	}

	// If no existing record found, create a new one (Requirement 2.1)
	if !found {
		newRecord := IntentSelectionRecord{
			DataSourceID: dataSourceID,
			IntentType:   intentType,
			IntentTitle:  intent.Title,
			SelectCount:  1,
			LastSelected: time.Now(),
		}
		records = append(records, newRecord)
	}

	// Update the selections for this data source (Requirement 2.5 - per data source tracking)
	p.intentSelections.Selections[dataSourceID] = records

	// Persist the selections to disk
	return p.saveIntentSelections()
}


// GetIntentRankingBoost calculates a ranking boost value for an intent type based on user's selection history.
// The boost value is used to re-rank intent suggestions, with higher values indicating more preferred intents.
//
// The method implements the following logic:
// 1. If total selections for the data source are less than MinSelectionsForPreference (5),
//    returns 0.0 to use default sorting (Requirement 2.6)
// 2. Otherwise, calculates a boost value based on the selection frequency ratio
//    (Requirement 2.3)
//
// Parameters:
//   - dataSourceID: the ID of the data source to get preferences for
//   - intentType: the type of intent (typically the intent title) to get boost for
//
// Returns:
//   - float64: boost value between 0.0 and 1.0, where higher values indicate stronger preference
//     - 0.0 means no boost (insufficient data or intent never selected)
//     - Values closer to 1.0 indicate higher selection frequency relative to other intents
//
// Validates: Requirements 2.3, 2.6
func (p *PreferenceLearner) GetIntentRankingBoost(dataSourceID string, intentType string) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get selections for this data source
	records, exists := p.intentSelections.Selections[dataSourceID]
	if !exists || len(records) == 0 {
		// No selections recorded for this data source
		return 0.0
	}

	// Calculate total selections for this data source
	totalSelections := 0
	for _, record := range records {
		totalSelections += record.SelectCount
	}

	// Check if we have sufficient data (Requirement 2.6)
	// If total selections are less than threshold, use default sorting (return 0.0)
	if totalSelections < MinSelectionsForPreference {
		return 0.0
	}

	// Find the selection count for the requested intent type
	intentSelectCount := 0
	for _, record := range records {
		if record.IntentType == intentType {
			intentSelectCount = record.SelectCount
			break
		}
	}

	// If this intent type was never selected, return 0.0
	if intentSelectCount == 0 {
		return 0.0
	}

	// Calculate boost as the ratio of this intent's selections to total selections
	// This gives a value between 0.0 and 1.0 (Requirement 2.3)
	// Higher selection frequency results in higher boost value
	boost := float64(intentSelectCount) / float64(totalSelections)

	return boost
}
