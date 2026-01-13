package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillManifest defines the metadata for a skill
type SkillManifest struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Version          string            `json:"version"`
	Author           string            `json:"author"`
	Category         string            `json:"category"`
	Keywords         []string          `json:"keywords"`
	RequiredColumns  []string          `json:"required_columns"`
	Tools            []string          `json:"tools"`
	Language         string            `json:"language"` // "python", "sql", "hybrid"
	CodeTemplate     string            `json:"code_template,omitempty"`
	PromptTemplate   string            `json:"prompt_template,omitempty"`
	Parameters       map[string]string `json:"parameters,omitempty"`
	Enabled          bool              `json:"enabled"`
	Icon             string            `json:"icon,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
}

// ConfigurableSkill implements a skill loaded from configuration
type ConfigurableSkill struct {
	Manifest     SkillManifest
	BasePath     string
	CodeContent  string
	PromptContent string
}

// Name returns the skill identifier
func (s *ConfigurableSkill) Name() string {
	return s.Manifest.ID
}

// Description returns a human-readable description
func (s *ConfigurableSkill) Description() string {
	return s.Manifest.Description
}

// Keywords returns trigger keywords that detect this skill
func (s *ConfigurableSkill) Keywords() []string {
	return s.Manifest.Keywords
}

// RequiredColumns returns the column types needed
func (s *ConfigurableSkill) RequiredColumns() []string {
	return s.Manifest.RequiredColumns
}

// CanExecute checks if the skill can run with the given schema
func (s *ConfigurableSkill) CanExecute(tables []TableInfo) bool {
	if len(s.Manifest.RequiredColumns) == 0 {
		return true
	}

	// Check if we have tables with required columns
	for _, table := range tables {
		hasRequired := make(map[string]bool)

		for _, col := range table.Columns {
			lower := strings.ToLower(col)

			for _, req := range s.Manifest.RequiredColumns {
				reqLower := strings.ToLower(req)
				if strings.Contains(lower, reqLower) || strings.Contains(reqLower, lower) {
					hasRequired[req] = true
				}
			}
		}

		// Check if all required columns found
		if len(hasRequired) == len(s.Manifest.RequiredColumns) {
			return true
		}
	}

	return false
}

// Execute runs the skill
func (s *ConfigurableSkill) Execute(ctx context.Context, executor DataExecutor, dataSourceID string, onProgress ProgressCallback) (*TemplateResult, error) {
	if !s.Manifest.Enabled {
		return &TemplateResult{
			Success: false,
			Error:   "Skill is disabled",
		}, nil
	}

	if onProgress != nil {
		onProgress("initializing", 5, fmt.Sprintf("Initializing %s...", s.Manifest.Name), 1, 6)
	}

	// Get schema
	tables, err := executor.GetSchema(ctx, dataSourceID)
	if err != nil {
		return &TemplateResult{Success: false, Error: fmt.Sprintf("Failed to get schema: %v", err)}, nil
	}

	if !s.CanExecute(tables) {
		return &TemplateResult{
			Success: false,
			Error:   fmt.Sprintf("Skill %s cannot execute: missing required columns", s.Manifest.Name),
		}, nil
	}

	if onProgress != nil {
		onProgress("preparing", 20, "Preparing analysis...", 2, 6)
	}

	// Find columns matching requirements
	columnMap := s.findColumns(tables)

	if onProgress != nil {
		onProgress("executing", 40, "Executing analysis...", 3, 6)
	}

	// Execute based on language
	var result *TemplateResult
	switch s.Manifest.Language {
	case "python":
		result, err = s.executePython(ctx, executor, dataSourceID, columnMap, onProgress)
	case "sql":
		result, err = s.executeSQL(ctx, executor, dataSourceID, columnMap, onProgress)
	case "hybrid":
		result, err = s.executeHybrid(ctx, executor, dataSourceID, columnMap, onProgress)
	default:
		return &TemplateResult{
			Success: false,
			Error:   fmt.Sprintf("Unsupported language: %s", s.Manifest.Language),
		}, nil
	}

	if err != nil {
		return result, err
	}

	if onProgress != nil {
		onProgress("complete", 100, "Analysis complete", 6, 6)
	}

	return result, nil
}

// findColumns finds the best matching columns for required column types
func (s *ConfigurableSkill) findColumns(tables []TableInfo) map[string]string {
	columnMap := make(map[string]string)

	for _, table := range tables {
		for _, col := range table.Columns {
			lower := strings.ToLower(col)

			for _, req := range s.Manifest.RequiredColumns {
				reqLower := strings.ToLower(req)
				if _, exists := columnMap[req]; !exists {
					if strings.Contains(lower, reqLower) || strings.Contains(reqLower, lower) {
						columnMap[req] = col
						columnMap["table"] = table.Name
					}
				}
			}
		}

		if len(columnMap) >= len(s.Manifest.RequiredColumns) {
			break
		}
	}

	return columnMap
}

// executePython executes Python-based skill
func (s *ConfigurableSkill) executePython(ctx context.Context, executor DataExecutor, dataSourceID string, columnMap map[string]string, onProgress ProgressCallback) (*TemplateResult, error) {
	if s.CodeContent == "" {
		return &TemplateResult{
			Success: false,
			Error:   "No Python code template found",
		}, nil
	}

	// Replace placeholders in code
	code := s.replacePlaceholders(s.CodeContent, columnMap)

	output, err := executor.ExecutePython(ctx, code, "")
	if err != nil {
		return &TemplateResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("Python execution failed: %v", err),
		}, nil
	}

	return &TemplateResult{
		Success: true,
		Output:  output,
	}, nil
}

// executeSQL executes SQL-based skill
func (s *ConfigurableSkill) executeSQL(ctx context.Context, executor DataExecutor, dataSourceID string, columnMap map[string]string, onProgress ProgressCallback) (*TemplateResult, error) {
	if s.CodeContent == "" {
		return &TemplateResult{
			Success: false,
			Error:   "No SQL template found",
		}, nil
	}

	// Replace placeholders in SQL
	query := s.replacePlaceholders(s.CodeContent, columnMap)

	data, err := executor.ExecuteSQL(ctx, dataSourceID, query)
	if err != nil {
		return &TemplateResult{
			Success: false,
			Error:   fmt.Sprintf("SQL execution failed: %v", err),
		}, nil
	}

	// Format results
	output := fmt.Sprintf("Query returned %d rows\n", len(data))
	if len(data) > 0 {
		jsonData, _ := json.MarshalIndent(data[:min(10, len(data))], "", "  ")
		output += fmt.Sprintf("\nFirst %d rows:\n%s", min(10, len(data)), string(jsonData))
	}

	return &TemplateResult{
		Success: true,
		Output:  output,
	}, nil
}

// executeHybrid executes hybrid SQL+Python skill
func (s *ConfigurableSkill) executeHybrid(ctx context.Context, executor DataExecutor, dataSourceID string, columnMap map[string]string, onProgress ProgressCallback) (*TemplateResult, error) {
	// First execute SQL to get data, then process with Python
	// This is a simplified implementation - can be extended
	return s.executePython(ctx, executor, dataSourceID, columnMap, onProgress)
}

// replacePlaceholders replaces {{column_name}} placeholders in templates
func (s *ConfigurableSkill) replacePlaceholders(template string, columnMap map[string]string) string {
	result := template
	for key, value := range columnMap {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	// Also support parameter replacements
	for key, value := range s.Manifest.Parameters {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// LoadSkillFromDirectory loads a skill from a directory
func LoadSkillFromDirectory(dirPath string) (*ConfigurableSkill, error) {
	// Load manifest
	manifestPath := filepath.Join(dirPath, "skill.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill manifest: %v", err)
	}

	var manifest SkillManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse skill manifest: %v", err)
	}

	skill := &ConfigurableSkill{
		Manifest: manifest,
		BasePath: dirPath,
	}

	// Load code template if specified
	if manifest.CodeTemplate != "" {
		codePath := filepath.Join(dirPath, manifest.CodeTemplate)
		codeData, err := os.ReadFile(codePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read code template: %v", err)
		}
		skill.CodeContent = string(codeData)
	}

	// Load prompt template if specified
	if manifest.PromptTemplate != "" {
		promptPath := filepath.Join(dirPath, manifest.PromptTemplate)
		promptData, err := os.ReadFile(promptPath)
		if err == nil {
			skill.PromptContent = string(promptData)
		}
	}

	return skill, nil
}

// LoadSkillsFromDirectory loads all skills from a directory
func LoadSkillsFromDirectory(skillsDir string) ([]*ConfigurableSkill, error) {
	var skills []*ConfigurableSkill

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Create skills directory if it doesn't exist
			if err := os.MkdirAll(skillsDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create skills directory: %v", err)
			}
			return skills, nil
		}
		return nil, fmt.Errorf("failed to read skills directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())
		skill, err := LoadSkillFromDirectory(skillPath)
		if err != nil {
			// Log error but continue loading other skills
			fmt.Printf("Warning: Failed to load skill from %s: %v\n", skillPath, err)
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// RegisterConfigurableSkills registers all configurable skills to the template registry
func RegisterConfigurableSkills(skillsDir string) error {
	skills, err := LoadSkillsFromDirectory(skillsDir)
	if err != nil {
		return err
	}

	for _, skill := range skills {
		if skill.Manifest.Enabled {
			Register(skill)
		}
	}

	return nil
}
