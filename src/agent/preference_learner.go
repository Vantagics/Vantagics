package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
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

// PreferenceLearner learns and manages user preferences
type PreferenceLearner struct {
	dataDir     string
	preferences UserPreferences
	rules       []BusinessRule
	benchmarks  []Benchmark
	mu          sync.RWMutex
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
	}

	learner.loadPreferences()
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
