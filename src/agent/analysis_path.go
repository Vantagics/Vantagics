package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AnalysisPath tracks the complete analysis storyline for a session
// This enables automatic report generation and analysis replay
type AnalysisPath struct {
	SessionID string               `json:"session_id"`
	CreatedAt int64                `json:"created_at"`
	UpdatedAt int64                `json:"updated_at"`
	Steps     []PathStep           `json:"steps"`
	Findings  []ConfirmedFinding   `json:"findings"`
}

// PathStep represents a single step in the analysis journey
// Following the Phenomenon → Action → Conclusion pattern
type PathStep struct {
	StepID      string     `json:"step_id"`
	Timestamp   int64      `json:"timestamp"`
	Phenomenon  string     `json:"phenomenon"`   // What was observed: "5月销量下降30%"
	Action      string     `json:"action"`       // What was done: "对比去年同期数据"
	Conclusion  string     `json:"conclusion"`   // What was found: "北方区缺货导致"
	Evidence    []Evidence `json:"evidence"`     // Supporting evidence (charts, queries)
	UserQuery   string     `json:"user_query"`   // Original user question
	AIResponse  string     `json:"ai_response"`  // AI's response
}

// Evidence represents supporting material for a step
type Evidence struct {
	Type        string `json:"type"`        // "chart", "query", "data", "calculation"
	Description string `json:"description"` // Human-readable description
	Data        string `json:"data"`        // Actual data (base64 chart, SQL query, etc.)
}

// ConfirmedFinding represents a user-confirmed or important finding
type ConfirmedFinding struct {
	FindingID    string   `json:"finding_id"`
	Content      string   `json:"content"`      // The finding text
	ConfirmedBy  string   `json:"confirmed_by"` // "user_marked", "auto_extracted", "llm_suggested"
	Importance   int      `json:"importance"`   // 1-5 scale
	Timestamp    int64    `json:"timestamp"`
	RelatedSteps []string `json:"related_steps"` // Related AnalysisStep IDs
	Tags         []string `json:"tags,omitempty"` // Optional categorization
}

// AnalysisPathManager manages analysis paths per session
type AnalysisPathManager struct {
	dataDir string
	paths   map[string]*AnalysisPath // sessionID -> path
	mu      sync.RWMutex
}

// NewAnalysisPathManager creates a new manager
func NewAnalysisPathManager(dataDir string) *AnalysisPathManager {
	pathDir := filepath.Join(dataDir, "analysis_paths")
	_ = os.MkdirAll(pathDir, 0755)

	return &AnalysisPathManager{
		dataDir: dataDir,
		paths:   make(map[string]*AnalysisPath),
	}
}

// AddStep adds a new step to the analysis path
func (m *AnalysisPathManager) AddStep(sessionID string, step PathStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.getOrCreatePath(sessionID)
	
	// Generate step ID if not provided
	if step.StepID == "" {
		step.StepID = fmt.Sprintf("step_%d", time.Now().UnixNano())
	}
	
	// Set timestamp
	if step.Timestamp == 0 {
		step.Timestamp = time.Now().Unix()
	}
	
	path.Steps = append(path.Steps, step)
	path.UpdatedAt = time.Now().Unix()
	
	return m.savePath(sessionID, path)
}

// AddFinding adds a confirmed finding
func (m *AnalysisPathManager) AddFinding(sessionID string, finding ConfirmedFinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.getOrCreatePath(sessionID)
	
	// Generate finding ID if not provided
	if finding.FindingID == "" {
		finding.FindingID = fmt.Sprintf("finding_%d", time.Now().UnixNano())
	}
	
	// Set timestamp
	if finding.Timestamp == 0 {
		finding.Timestamp = time.Now().Unix()
	}
	
	path.Findings = append(path.Findings, finding)
	path.UpdatedAt = time.Now().Unix()
	
	return m.savePath(sessionID, path)
}

// GetPath retrieves the analysis path for a session
func (m *AnalysisPathManager) GetPath(sessionID string) *AnalysisPath {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path, exists := m.paths[sessionID]
	if !exists {
		// Try loading from disk
		if loaded := m.loadPath(sessionID); loaded != nil {
			m.mu.RUnlock()
			m.mu.Lock()
			m.paths[sessionID] = loaded
			m.mu.Unlock()
			m.mu.RLock()
			return loaded
		}
		return nil
	}

	return path
}

// GenerateStoryline creates a narrative summary of the analysis path
func (path *AnalysisPath) GenerateStoryline() string {
	if path == nil || len(path.Steps) == 0 {
		return "分析尚未开始。"
	}

	var story strings.Builder
	story.WriteString("## 分析路径总结\n\n")

	for i, step := range path.Steps {
		story.WriteString(fmt.Sprintf("### 步骤 %d\n", i+1))
		
		if step.Phenomenon != "" {
			story.WriteString(fmt.Sprintf("**发现现象**: %s\n\n", step.Phenomenon))
		}
		
		if step.Action != "" {
			story.WriteString(fmt.Sprintf("**采取行动**: %s\n\n", step.Action))
		}
		
		if step.Conclusion != "" {
			story.WriteString(fmt.Sprintf("**得出结论**: %s\n\n", step.Conclusion))
		}
		
		if len(step.Evidence) > 0 {
			story.WriteString("**支持证据**:\n")
			for _, ev := range step.Evidence {
				story.WriteString(fmt.Sprintf("- %s: %s\n", ev.Type, ev.Description))
			}
			story.WriteString("\n")
		}
		
		story.WriteString("---\n\n")
	}

	// Add confirmed findings
	if len(path.Findings) > 0 {
		story.WriteString("## 重要发现\n\n")
		for _, finding := range path.Findings {
			importance := strings.Repeat("⭐", finding.Importance)
			story.WriteString(fmt.Sprintf("- %s %s\n", importance, finding.Content))
		}
	}

	return story.String()
}

// ExtractStepFromInteraction analyzes user query and AI response to extract a step
func ExtractStepFromInteraction(userQuery, aiResponse string, sqlQueries []string, charts []string) PathStep {
	step := PathStep{
		StepID:     fmt.Sprintf("step_%d", time.Now().UnixNano()),
		Timestamp:  time.Now().Unix(),
		UserQuery:  userQuery,
		AIResponse: aiResponse,
		Evidence:   []Evidence{},
	}

	// Extract phenomenon (what was observed)
	step.Phenomenon = extractPhenomenon(aiResponse)
	
	// Action is the user's query (what they asked to do)
	step.Action = summarizeAction(userQuery)
	
	// Conclusion is extracted from AI's response
	step.Conclusion = extractConclusion(aiResponse)
	
	// Add SQL queries as evidence
	for _, query := range sqlQueries {
		step.Evidence = append(step.Evidence, Evidence{
			Type:        "query",
			Description: "SQL查询",
			Data:        query,
		})
	}
	
	// Add charts as evidence
	for _, chart := range charts {
		step.Evidence = append(step.Evidence, Evidence{
			Type:        "chart",
			Description: "可视化图表",
			Data:        chart,
		})
	}

	return step
}

// Helper functions for extraction

func extractPhenomenon(response string) string {
	// Look for patterns indicating observations
	phenomenonKeywords := []string{
		"发现", "观察到", "显示", "表明", "数据显示",
		"结果显示", "分析发现", "可以看到",
	}
	
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, keyword := range phenomenonKeywords {
			if strings.Contains(line, keyword) && len(line) < 200 {
				return line
			}
		}
	}
	
	return ""
}

func summarizeAction(userQuery string) string {
	// Simplify the user's query to action description
	if len(userQuery) <= 100 {
		return userQuery
	}
	return userQuery[:100] + "..."
}

func extractConclusion(response string) string {
	// Look for conclusion patterns
	conclusionKeywords := []string{
		"因此", "所以", "结论", "综上", "总结",
		"说明", "表明", "证明", "可见",
	}
	
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, keyword := range conclusionKeywords {
			if strings.Contains(line, keyword) && len(line) < 200 && len(line) > 10 {
				return line
			}
		}
	}
	
	// If no explicit conclusion, try to extract from end of response
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if len(lastLine) > 10 && len(lastLine) < 200 {
			return lastLine
		}
	}
	
	return ""
}

// Internal helper methods

func (m *AnalysisPathManager) getOrCreatePath(sessionID string) *AnalysisPath {
	path, exists := m.paths[sessionID]
	if !exists {
		path = &AnalysisPath{
			SessionID: sessionID,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			Steps:     []PathStep{},
			Findings:  []ConfirmedFinding{},
		}
		m.paths[sessionID] = path
	}
	return path
}

func (m *AnalysisPathManager) getPathFilePath(sessionID string) string {
	return filepath.Join(m.dataDir, "analysis_paths", sessionID+".json")
}

func (m *AnalysisPathManager) savePath(sessionID string, path *AnalysisPath) error {
	filePath := m.getPathFilePath(sessionID)
	
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(path, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (m *AnalysisPathManager) loadPath(sessionID string) *AnalysisPath {
	filePath := m.getPathFilePath(sessionID)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var path AnalysisPath
	if err := json.Unmarshal(data, &path); err != nil {
		return nil
	}

	return &path
}

// ClearPath removes the analysis path for a session
func (m *AnalysisPathManager) ClearPath(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.paths, sessionID)
	
	// Delete from disk
	filePath := m.getPathFilePath(sessionID)
	_ = os.Remove(filePath)
}
