package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IntentRanker 意图排序�?
// 根据用户历史选择对意图建议进行排�?
// 简化自 PreferenceLearner 的意图选择功能
// Validates: Requirements 5.1, 5.2
type IntentRanker struct {
	preferencesStore *PreferencesStore
	threshold        int // 最小选择次数阈�?
	dataDir          string
	mu               sync.RWMutex
}

// PreferencesStore 偏好存储
// 存储用户的意图选择记录，按数据源分�?
type PreferencesStore struct {
	Selections map[string][]SelectionRecord `json:"selections"` // dataSourceID -> records
}

// SelectionRecord 选择记录
// 记录用户对某种意图类型的选择历史
type SelectionRecord struct {
	IntentType   string    `json:"intent_type"`   // 意图类型（通常是意图标题）
	SelectCount  int       `json:"select_count"`  // 选择次数
	LastSelected time.Time `json:"last_selected"` // 最后选择时间
}

// NewIntentRanker 创建意图排序�?
// Parameters:
//   - dataDir: 数据存储目录
//   - threshold: 最小选择次数阈值，低于此值时保持原始排序
//
// Returns: 初始化后�?IntentRanker 实例
func NewIntentRanker(dataDir string, threshold int) *IntentRanker {
	// 确保目录存在
	prefsDir := filepath.Join(dataDir, "preferences")
	_ = os.MkdirAll(prefsDir, 0755)

	// 使用默认阈值如果传入值无�?
	if threshold <= 0 {
		threshold = DefaultPreferenceThreshold
	}

	ranker := &IntentRanker{
		preferencesStore: &PreferencesStore{
			Selections: make(map[string][]SelectionRecord),
		},
		threshold: threshold,
		dataDir:   dataDir,
	}

	// 加载已有的偏好数�?
	ranker.load()

	return ranker
}

// getStorePath 获取存储文件路径
func (r *IntentRanker) getStorePath() string {
	return filepath.Join(r.dataDir, "preferences", "intent_preferences.json")
}

// load 从文件加载偏好数�?
func (r *IntentRanker) load() error {
	path := r.getStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在是正常情况，初始化空存�?
			return nil
		}
		return err
	}

	var store PreferencesStore
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	// 确保 map 已初始化
	if store.Selections == nil {
		store.Selections = make(map[string][]SelectionRecord)
	}

	r.preferencesStore = &store
	return nil
}

// save 保存偏好数据到文�?
func (r *IntentRanker) save() error {
	path := r.getStorePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.preferencesStore, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// RankSuggestions 排序意图建议
// 根据用户偏好重新排序意图建议列表
// 当选择次数少于阈值时，保持原始排�?
//
// Parameters:
//   - dataSourceID: 数据源ID
//   - suggestions: 原始意图建议列表
//
// Returns: 排序后的意图建议列表
// Validates: Requirements 5.3, 5.4
func (r *IntentRanker) RankSuggestions(
	dataSourceID string,
	suggestions []IntentSuggestion,
) []IntentSuggestion {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 如果没有建议，直接返�?
	if len(suggestions) == 0 {
		return suggestions
	}

	// 获取总选择次数
	totalCount := r.getSelectionCountInternal(dataSourceID)

	// 如果选择次数少于阈值，保持原始排序
	if totalCount < r.threshold {
		// 返回副本以避免修改原始切�?
		result := make([]IntentSuggestion, len(suggestions))
		copy(result, suggestions)
		return result
	}

	// 获取该数据源的选择记录
	records := r.preferencesStore.Selections[dataSourceID]
	if len(records) == 0 {
		result := make([]IntentSuggestion, len(suggestions))
		copy(result, suggestions)
		return result
	}

	// 创建意图类型到选择次数的映�?
	selectCountMap := make(map[string]int)
	for _, record := range records {
		selectCountMap[record.IntentType] = record.SelectCount
	}

	// 创建带排序权重的建议列表
	type rankedSuggestion struct {
		suggestion IntentSuggestion
		weight     int
		index      int // 原始索引，用于稳定排�?
	}

	ranked := make([]rankedSuggestion, len(suggestions))
	for i, s := range suggestions {
		weight := selectCountMap[s.Title] // 使用标题作为意图类型
		ranked[i] = rankedSuggestion{
			suggestion: s,
			weight:     weight,
			index:      i,
		}
	}

	// 按权重降序排序（稳定排序，相同权重保持原始顺序）
	// 使用简单的冒泡排序实现稳定排序
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			// 只有当权重更高时才交换，保持稳定�?
			if ranked[j].weight > ranked[i].weight {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	// 提取排序后的建议
	result := make([]IntentSuggestion, len(ranked))
	for i, r := range ranked {
		result[i] = r.suggestion
	}

	return result
}

// RecordSelection 记录用户选择
// 记录用户选择的意图，用于偏好学习
//
// Parameters:
//   - dataSourceID: 数据源ID
//   - intent: 用户选择的意图建�?
//
// Returns: 保存失败时返回错�?
// Validates: Requirements 5.1, 5.2
func (r *IntentRanker) RecordSelection(
	dataSourceID string,
	intent IntentSuggestion,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 确保 map 已初始化
	if r.preferencesStore.Selections == nil {
		r.preferencesStore.Selections = make(map[string][]SelectionRecord)
	}

	// 获取该数据源的现有记�?
	records := r.preferencesStore.Selections[dataSourceID]

	// 使用标题作为意图类型
	intentType := intent.Title
	if intentType == "" {
		intentType = "unknown"
	}

	// 查找是否已有该意图类型的记录
	found := false
	for i := range records {
		if records[i].IntentType == intentType {
			// 增加选择次数
			records[i].SelectCount++
			records[i].LastSelected = time.Now()
			found = true
			break
		}
	}

	// 如果没有找到，创建新记录
	if !found {
		newRecord := SelectionRecord{
			IntentType:   intentType,
			SelectCount:  1,
			LastSelected: time.Now(),
		}
		records = append(records, newRecord)
	}

	// 更新存储
	r.preferencesStore.Selections[dataSourceID] = records

	// 持久化到文件
	return r.save()
}

// GetSelectionCount 获取总选择次数
// 返回指定数据源的总意图选择次数
//
// Parameters:
//   - dataSourceID: 数据源ID
//
// Returns: 总选择次数
// Validates: Requirements 5.2
func (r *IntentRanker) GetSelectionCount(dataSourceID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.getSelectionCountInternal(dataSourceID)
}

// getSelectionCountInternal 内部方法，获取总选择次数（不加锁�?
func (r *IntentRanker) getSelectionCountInternal(dataSourceID string) int {
	records, exists := r.preferencesStore.Selections[dataSourceID]
	if !exists {
		return 0
	}

	total := 0
	for _, record := range records {
		total += record.SelectCount
	}
	return total
}

// GetThreshold 获取当前阈�?
func (r *IntentRanker) GetThreshold() int {
	return r.threshold
}

// SetThreshold 设置阈�?
func (r *IntentRanker) SetThreshold(threshold int) {
	if threshold > 0 {
		r.threshold = threshold
	}
}

// GetSelectionRecords 获取指定数据源的所有选择记录
// 返回记录的副本以避免外部修改
func (r *IntentRanker) GetSelectionRecords(dataSourceID string) []SelectionRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records, exists := r.preferencesStore.Selections[dataSourceID]
	if !exists {
		return []SelectionRecord{}
	}

	// 返回副本
	result := make([]SelectionRecord, len(records))
	copy(result, records)
	return result
}

// ClearSelections 清除指定数据源的所有选择记录
// 用于测试或重置偏�?
func (r *IntentRanker) ClearSelections(dataSourceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.preferencesStore.Selections, dataSourceID)
	return r.save()
}

// ClearAllSelections 清除所有选择记录
// 用于测试或完全重�?
func (r *IntentRanker) ClearAllSelections() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.preferencesStore.Selections = make(map[string][]SelectionRecord)
	return r.save()
}
