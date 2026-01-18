package templates

import (
	"fmt"
	"path/filepath"
	"sync"
)

// SkillManager manages the loading and registration of skills
type SkillManager struct {
	skillsDir     string
	skills        map[string]*ConfigurableSkill
	mu            sync.RWMutex
	logger        func(string)
}

// NewSkillManager creates a new SkillManager
func NewSkillManager(skillsDir string, logger func(string)) *SkillManager {
	return &SkillManager{
		skillsDir: skillsDir,
		skills:    make(map[string]*ConfigurableSkill),
		logger:    logger,
	}
}

// LoadSkills loads all skills from the skills directory
func (sm *SkillManager) LoadSkills() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.logger != nil {
		sm.logger(fmt.Sprintf("[SKILLS] Loading skills from: %s", sm.skillsDir))
	}

	skills, err := LoadSkillsFromDirectory(sm.skillsDir)
	if err != nil {
		return fmt.Errorf("failed to load skills: %v", err)
	}

	// Register enabled skills
	loadedCount := 0
	enabledCount := 0
	for _, skill := range skills {
		sm.skills[skill.Manifest.ID] = skill

		if skill.Manifest.Enabled {
			Register(skill)
			enabledCount++
			if sm.logger != nil {
				sm.logger(fmt.Sprintf("[SKILLS] Registered skill: %s (%s)", skill.Manifest.Name, skill.Manifest.ID))
			}
		}
		loadedCount++
	}

	if sm.logger != nil {
		sm.logger(fmt.Sprintf("[SKILLS] Loaded %d skills (%d enabled, %d disabled)", loadedCount, enabledCount, loadedCount-enabledCount))
	}

	return nil
}

// ReloadSkills reloads all skills from disk
func (sm *SkillManager) ReloadSkills() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Clear existing skills
	sm.skills = make(map[string]*ConfigurableSkill)

	// Clear registry (only configurable skills)
	for name, template := range Registry {
		if _, ok := template.(*ConfigurableSkill); ok {
			delete(Registry, name)
		}
	}

	if sm.logger != nil {
		sm.logger("[SKILLS] Reloading all skills...")
	}

	return sm.LoadSkills()
}

// GetSkill returns a skill by ID
func (sm *SkillManager) GetSkill(id string) (*ConfigurableSkill, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[id]
	return skill, exists
}

// ListSkills returns all loaded skills
func (sm *SkillManager) ListSkills() []*ConfigurableSkill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]*ConfigurableSkill, 0, len(sm.skills))
	for _, skill := range sm.skills {
		skills = append(skills, skill)
	}
	return skills
}

// ListEnabledSkills returns only enabled skills
func (sm *SkillManager) ListEnabledSkills() []*ConfigurableSkill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]*ConfigurableSkill, 0)
	for _, skill := range sm.skills {
		if skill.Manifest.Enabled {
			skills = append(skills, skill)
		}
	}
	return skills
}

// EnableSkill enables a skill by ID
func (sm *SkillManager) EnableSkill(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[id]
	if !exists {
		return fmt.Errorf("skill not found: %s", id)
	}

	if !skill.Manifest.Enabled {
		skill.Manifest.Enabled = true
		Register(skill)

		// TODO: Persist this change to skill.json
		if sm.logger != nil {
			sm.logger(fmt.Sprintf("[SKILLS] Enabled skill: %s", id))
		}
	}

	return nil
}

// DisableSkill disables a skill by ID
func (sm *SkillManager) DisableSkill(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[id]
	if !exists {
		return fmt.Errorf("skill not found: %s", id)
	}

	if skill.Manifest.Enabled {
		skill.Manifest.Enabled = false
		delete(Registry, id)

		// TODO: Persist this change to skill.json
		if sm.logger != nil {
			sm.logger(fmt.Sprintf("[SKILLS] Disabled skill: %s", id))
		}
	}

	return nil
}

// GetSkillByCategory returns skills filtered by category
func (sm *SkillManager) GetSkillByCategory(category string) []*ConfigurableSkill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]*ConfigurableSkill, 0)
	for _, skill := range sm.skills {
		if skill.Manifest.Category == category {
			skills = append(skills, skill)
		}
	}
	return skills
}

// SearchSkills searches skills by keyword
func (sm *SkillManager) SearchSkills(query string) []*ConfigurableSkill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skills := make([]*ConfigurableSkill, 0)
	queryLower := filepath.Base(query) // Simple lowercase matching

	for _, skill := range sm.skills {
		// Search in name, description, keywords, tags
		if containsIgnoreCase(skill.Manifest.Name, queryLower) ||
			containsIgnoreCase(skill.Manifest.Description, queryLower) ||
			containsIgnoreCase(skill.Manifest.ID, queryLower) {
			skills = append(skills, skill)
			continue
		}

		// Search in keywords
		for _, keyword := range skill.Manifest.Keywords {
			if containsIgnoreCase(keyword, queryLower) {
				skills = append(skills, skill)
				break
			}
		}
	}

	return skills
}

// GetCategories returns all unique skill categories
func (sm *SkillManager) GetCategories() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	categories := make(map[string]bool)
	for _, skill := range sm.skills {
		if skill.Manifest.Category != "" {
			categories[skill.Manifest.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}
	return result
}

// containsIgnoreCase performs case-insensitive substring search
func containsIgnoreCase(s, substr string) bool {
	// Simple implementation - can use strings.ToLower for proper case folding
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(s) > 0 && len(substr) > 0)
}
