package agent

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Skill represents a skill with its metadata
type Skill struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Path        string    `json:"path"`
	InstalledAt time.Time `json:"installed_at"`
	Enabled     bool      `json:"enabled"`
}

// SkillConfig stores the enabled/disabled state of skills
type SkillConfig struct {
	EnabledSkills map[string]bool `json:"enabled_skills"`
}

// SkillService manages skills installation and listing
type SkillService struct {
	skillsDir  string
	configPath string
	logger     func(string)
}

// NewSkillService creates a new skill service
func NewSkillService(dataDir string, logger func(string)) *SkillService {
	skillsDir := filepath.Join(dataDir, "skills")
	configPath := filepath.Join(dataDir, "skills_config.json")
	
	// Ensure skills directory exists
	os.MkdirAll(skillsDir, 0755)
	
	return &SkillService{
		skillsDir:  skillsDir,
		configPath: configPath,
		logger:     logger,
	}
}

// loadConfig loads the skill configuration
func (s *SkillService) loadConfig() (*SkillConfig, error) {
	config := &SkillConfig{
		EnabledSkills: make(map[string]bool),
	}
	
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist yet, return empty config
			return config, nil
		}
		return nil, err
	}
	
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// saveConfig saves the skill configuration
func (s *SkillService) saveConfig(config *SkillConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(s.configPath, data, 0644)
}

// ListSkills returns all installed skills
func (s *SkillService) ListSkills() ([]Skill, error) {
	s.logger("[SKILLS] Listing skills from: " + s.skillsDir)
	
	// Load configuration
	config, err := s.loadConfig()
	if err != nil {
		s.logger(fmt.Sprintf("[SKILLS] Failed to load config: %v", err))
		config = &SkillConfig{EnabledSkills: make(map[string]bool)}
	}
	
	var skills []Skill
	
	// Read skills directory
	entries, err := os.ReadDir(s.skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %v", err)
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		skillName := entry.Name()
		skillPath := filepath.Join(s.skillsDir, skillName)
		skillMdPath := filepath.Join(skillPath, "SKILL.md")
		
		// Check if SKILL.md exists
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			s.logger(fmt.Sprintf("[SKILLS] Skipping %s: no SKILL.md found", skillName))
			continue
		}
		
		// Read SKILL.md content
		content, err := os.ReadFile(skillMdPath)
		if err != nil {
			s.logger(fmt.Sprintf("[SKILLS] Failed to read SKILL.md for %s: %v", skillName, err))
			continue
		}
		
		// Extract description (first paragraph after title)
		description := extractDescription(string(content))
		
		// Get installation time from directory modification time
		info, _ := entry.Info()
		installedAt := time.Now()
		if info != nil {
			installedAt = info.ModTime()
		}
		
		// Check if skill is enabled (default to true for new skills)
		enabled, exists := config.EnabledSkills[skillName]
		if !exists {
			enabled = true
			config.EnabledSkills[skillName] = true
		}
		
		skills = append(skills, Skill{
			Name:        skillName,
			Description: description,
			Content:     string(content),
			Path:        skillPath,
			InstalledAt: installedAt,
			Enabled:     enabled,
		})
	}
	
	// Save config to persist any new skills
	if err := s.saveConfig(config); err != nil {
		s.logger(fmt.Sprintf("[SKILLS] Failed to save config: %v", err))
	}
	
	s.logger(fmt.Sprintf("[SKILLS] Found %d skills", len(skills)))
	return skills, nil
}

// InstallFromZip installs skills from a ZIP file
func (s *SkillService) InstallFromZip(zipPath string) ([]string, error) {
	s.logger("[SKILLS] Installing from ZIP: " + zipPath)
	
	// Create temporary directory for extraction
	tempDir := filepath.Join(filepath.Dir(s.skillsDir), "temp", "skills_install")
	os.RemoveAll(tempDir) // Clean up any previous temp files
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after installation
	
	// Extract ZIP
	if err := unzip(zipPath, tempDir); err != nil {
		return nil, fmt.Errorf("failed to extract ZIP: %v", err)
	}
	
	// Find skill directories (directories containing SKILL.md)
	skillDirs, err := findSkillDirectories(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find skill directories: %v", err)
	}
	
	if len(skillDirs) == 0 {
		return nil, fmt.Errorf("no valid skills found in ZIP (each skill must have a SKILL.md file)")
	}
	
	// Check for name conflicts
	var conflicts []string
	for _, skillDir := range skillDirs {
		skillName := filepath.Base(skillDir)
		targetPath := filepath.Join(s.skillsDir, skillName)
		if _, err := os.Stat(targetPath); err == nil {
			conflicts = append(conflicts, skillName)
		}
	}
	
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("skill name conflicts: %s already exist", strings.Join(conflicts, ", "))
	}
	
	// Copy skills to skills directory and create skill.json
	var installed []string
	for _, skillDir := range skillDirs {
		skillName := filepath.Base(skillDir)
		targetPath := filepath.Join(s.skillsDir, skillName)
		
		if err := copyDir(skillDir, targetPath); err != nil {
			s.logger(fmt.Sprintf("[SKILLS] Failed to copy %s: %v", skillName, err))
			continue
		}
		
		// Create skill.json if it doesn't exist
		if err := s.ensureSkillJSON(targetPath, skillName); err != nil {
			s.logger(fmt.Sprintf("[SKILLS] Warning: Failed to create skill.json for %s: %v", skillName, err))
		}
		
		installed = append(installed, skillName)
		s.logger(fmt.Sprintf("[SKILLS] Installed: %s", skillName))
	}
	
	if len(installed) == 0 {
		return nil, fmt.Errorf("failed to install any skills")
	}
	
	s.logger(fmt.Sprintf("[SKILLS] Successfully installed %d skills", len(installed)))
	return installed, nil
}

// EnableSkill enables a skill by name
func (s *SkillService) EnableSkill(skillName string) error {
	s.logger(fmt.Sprintf("[SKILLS] Enabling skill: %s", skillName))
	
	// Check if skill exists
	skillPath := filepath.Join(s.skillsDir, skillName)
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return fmt.Errorf("skill not found: %s", skillName)
	}
	
	// Load config
	config, err := s.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	
	// Enable skill
	config.EnabledSkills[skillName] = true
	
	// Save config
	if err := s.saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}
	
	// Update skill.json if it exists (for templates.SkillManager compatibility)
	s.updateSkillJSON(skillPath, true)
	
	s.logger(fmt.Sprintf("[SKILLS] Enabled: %s", skillName))
	return nil
}

// DisableSkill disables a skill by name
// Note: This only disables the skill, it does not delete it
func (s *SkillService) DisableSkill(skillName string) error {
	s.logger(fmt.Sprintf("[SKILLS] Disabling skill: %s", skillName))
	
	// Check if skill exists
	skillPath := filepath.Join(s.skillsDir, skillName)
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return fmt.Errorf("skill not found: %s", skillName)
	}
	
	// Load config
	config, err := s.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	
	// Disable skill
	config.EnabledSkills[skillName] = false
	
	// Save config
	if err := s.saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}
	
	// Update skill.json if it exists (for templates.SkillManager compatibility)
	s.updateSkillJSON(skillPath, false)
	
	s.logger(fmt.Sprintf("[SKILLS] Disabled: %s", skillName))
	return nil
}

// DeleteSkill deletes a skill by name (removes directory and config)
func (s *SkillService) DeleteSkill(skillName string) error {
	s.logger(fmt.Sprintf("[SKILLS] Deleting skill: %s", skillName))
	
	// Check if skill exists
	skillPath := filepath.Join(s.skillsDir, skillName)
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return fmt.Errorf("skill not found: %s", skillName)
	}
	
	// Load config
	config, err := s.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	
	// Remove from enabled skills map
	delete(config.EnabledSkills, skillName)
	
	// Save config
	if err := s.saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}
	
	// Delete skill directory
	if err := os.RemoveAll(skillPath); err != nil {
		return fmt.Errorf("failed to delete skill directory: %v", err)
	}
	
	s.logger(fmt.Sprintf("[SKILLS] Successfully deleted skill: %s", skillName))
	return nil
}

// updateSkillJSON updates the enabled field in skill.json if it exists
func (s *SkillService) updateSkillJSON(skillPath string, enabled bool) {
	skillJSONPath := filepath.Join(skillPath, "skill.json")
	
	// Check if skill.json exists
	data, err := os.ReadFile(skillJSONPath)
	if err != nil {
		// skill.json doesn't exist, skip
		return
	}
	
	// Parse JSON
	var skillData map[string]interface{}
	if err := json.Unmarshal(data, &skillData); err != nil {
		s.logger(fmt.Sprintf("[SKILLS] Warning: Failed to parse skill.json for %s: %v", filepath.Base(skillPath), err))
		return
	}
	
	// Update enabled field
	skillData["enabled"] = enabled
	
	// Save back
	updatedData, err := json.MarshalIndent(skillData, "", "  ")
	if err != nil {
		s.logger(fmt.Sprintf("[SKILLS] Warning: Failed to marshal skill.json for %s: %v", filepath.Base(skillPath), err))
		return
	}
	
	if err := os.WriteFile(skillJSONPath, updatedData, 0644); err != nil {
		s.logger(fmt.Sprintf("[SKILLS] Warning: Failed to write skill.json for %s: %v", filepath.Base(skillPath), err))
	}
}

// ensureSkillJSON creates or updates skill.json for a skill
func (s *SkillService) ensureSkillJSON(skillPath, skillName string) error {
	skillJSONPath := filepath.Join(skillPath, "skill.json")
	
	// Check if skill.json already exists
	if _, err := os.Stat(skillJSONPath); err == nil {
		// skill.json exists, just update enabled field
		s.updateSkillJSON(skillPath, true)
		return nil
	}
	
	// Read SKILL.md to extract metadata
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return fmt.Errorf("failed to read SKILL.md: %v", err)
	}
	
	// Parse SKILL.md to extract metadata
	metadata := parseSkillMD(string(content), skillName)
	
	// Create skill.json
	skillJSON := map[string]interface{}{
		"id":               skillName,
		"name":             metadata["name"],
		"description":      metadata["description"],
		"version":          metadata["version"],
		"author":           metadata["author"],
		"category":         metadata["category"],
		"keywords":         metadata["keywords"],
		"required_columns": []string{},
		"tools":            []string{"python", "sql"},
		"language":         "python",
		"enabled":          true,
		"icon":             "chart",
		"tags":             []string{},
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(skillJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill.json: %v", err)
	}
	
	// Write to file
	if err := os.WriteFile(skillJSONPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write skill.json: %v", err)
	}
	
	s.logger(fmt.Sprintf("[SKILLS] Created skill.json for %s", skillName))
	return nil
}

// parseSkillMD extracts metadata from SKILL.md content
func parseSkillMD(content, skillName string) map[string]interface{} {
	metadata := map[string]interface{}{
		"name":        skillName,
		"description": "No description available",
		"version":     "1.0.0",
		"author":      "Unknown",
		"category":    "general",
		"keywords":    []string{},
	}
	
	lines := strings.Split(content, "\n")
	currentSection := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Extract title (first # heading)
		if strings.HasPrefix(line, "# ") && metadata["name"] == skillName {
			metadata["name"] = strings.TrimPrefix(line, "# ")
			continue
		}
		
		// Track sections
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.ToLower(strings.TrimPrefix(line, "## "))
			continue
		}
		
		// Extract description (first non-empty line after title)
		if currentSection == "描述" || currentSection == "description" {
			if line != "" && !strings.HasPrefix(line, "#") {
				metadata["description"] = line
				currentSection = "" // Only take first line
			}
		}
		
		// Extract version
		if currentSection == "版本" || currentSection == "version" {
			if line != "" && !strings.HasPrefix(line, "#") {
				metadata["version"] = line
				currentSection = ""
			}
		}
		
		// Extract author
		if currentSection == "作者" || currentSection == "author" {
			if line != "" && !strings.HasPrefix(line, "#") {
				metadata["author"] = line
				currentSection = ""
			}
		}
		
		// Extract category
		if currentSection == "分类" || currentSection == "category" {
			if line != "" && !strings.HasPrefix(line, "#") {
				metadata["category"] = line
				currentSection = ""
			}
		}
	}
	
	return metadata
}

// GetEnabledSkills returns only the enabled skills
func (s *SkillService) GetEnabledSkills() ([]Skill, error) {
	allSkills, err := s.ListSkills()
	if err != nil {
		return nil, err
	}
	
	var enabledSkills []Skill
	for _, skill := range allSkills {
		if skill.Enabled {
			enabledSkills = append(enabledSkills, skill)
		}
	}
	
	s.logger(fmt.Sprintf("[SKILLS] %d enabled skills out of %d total", len(enabledSkills), len(allSkills)))
	return enabledSkills, nil
}

// extractDescription extracts a brief description from SKILL.md content
func extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	
	// Skip title lines (starting with #)
	// Find first non-empty, non-title line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Return first meaningful line, truncate if too long
		if len(line) > 200 {
			return line[:200] + "..."
		}
		return line
	}
	
	return "No description available"
}

// findSkillDirectories finds all directories containing SKILL.md
func findSkillDirectories(rootDir string) ([]string, error) {
	var skillDirs []string
	
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Check if this is a SKILL.md file
		if !info.IsDir() && strings.ToUpper(info.Name()) == "SKILL.MD" {
			// Add the parent directory
			skillDir := filepath.Dir(path)
			skillDirs = append(skillDirs, skillDir)
		}
		
		return nil
	})
	
	return skillDirs, err
}

// unzip extracts a ZIP file to a destination directory
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	
	for _, f := range r.File {
		// Prevent path traversal attacks
		if strings.Contains(f.Name, "..") {
			continue
		}
		
		fpath := filepath.Join(dest, f.Name)
		
		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}
		
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		
		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		
		// Extract file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		
		if err != nil {
			return err
		}
	}
	
	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	
	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		
		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	
	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	return os.Chmod(dst, srcInfo.Mode())
}
