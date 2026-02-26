package main

import (
	"context"
	"fmt"

	"vantagics/agent"
	"vantagics/agent/templates"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SkillManager 定义技能管理接�
type SkillManager interface {
	GetSkills() ([]SkillInfo, error)
	GetEnabledSkills() ([]SkillInfo, error)
	GetSkillCategories() ([]string, error)
	EnableSkill(skillID string) error
	DisableSkill(skillID string) error
	DeleteSkill(skillID string) error
	ReloadSkills() error
	ListSkills() ([]agent.Skill, error)
	InstallSkillsFromZip() ([]string, error)
}

// SkillFacadeService 技能服务门面，封装所有技能相关的业务逻辑
type SkillFacadeService struct {
	ctx              context.Context
	skillService     *agent.SkillService
	einoService      *agent.EinoService
	chatFacadeService *ChatFacadeService
	logger           func(string)
}

// NewSkillFacadeService 创建新的 SkillFacadeService 实例
func NewSkillFacadeService(
	skillService *agent.SkillService,
	einoService *agent.EinoService,
	chatFacadeService *ChatFacadeService,
	logger func(string),
) *SkillFacadeService {
	return &SkillFacadeService{
		skillService:      skillService,
		einoService:       einoService,
		chatFacadeService: chatFacadeService,
		logger:            logger,
	}
}

// Name 返回服务名称
func (s *SkillFacadeService) Name() string {
	return "skill"
}

// Initialize 初始化技能门面服�
func (s *SkillFacadeService) Initialize(ctx context.Context) error {
	s.ctx = ctx
	s.log("SkillFacadeService initialized")
	return nil
}

// Shutdown 关闭技能门面服�
func (s *SkillFacadeService) Shutdown() error {
	return nil
}

// SetContext 设置 Wails 上下�
func (s *SkillFacadeService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// SetEinoService 设置 EinoService（用于延迟注入或重新初始化）
func (s *SkillFacadeService) SetEinoService(es *agent.EinoService) {
	s.einoService = es
}

// SetSkillService 设置 SkillService（用于延迟注入）
func (s *SkillFacadeService) SetSkillService(ss *agent.SkillService) {
	s.skillService = ss
}

// SetChatFacadeService 设置 ChatFacadeService（用于延迟注入）
func (s *SkillFacadeService) SetChatFacadeService(cfs *ChatFacadeService) {
	s.chatFacadeService = cfs
}

// log 记录日志
func (s *SkillFacadeService) log(msg string) {
	if s.logger != nil {
		s.logger(msg)
	}
}

// getSkillManager 获取 SkillManager 实例
func (s *SkillFacadeService) getSkillManager() *templates.SkillManager {
	if s.einoService == nil {
		return nil
	}
	return s.einoService.GetSkillManager()
}

// --- Skill Query Methods ---

// GetSkills 返回所有已加载的技�
func (s *SkillFacadeService) GetSkills() ([]SkillInfo, error) {
	skillManager := s.getSkillManager()
	if skillManager == nil {
		return nil, WrapError("skill", "GetSkills", fmt.Errorf("eino service or skill manager not available"))
	}

	skills := skillManager.ListSkills()
	result := make([]SkillInfo, 0, len(skills))

	for _, skill := range skills {
		result = append(result, SkillInfo{
			ID:              skill.Manifest.ID,
			Name:            skill.Manifest.Name,
			Description:     skill.Manifest.Description,
			Version:         skill.Manifest.Version,
			Author:          skill.Manifest.Author,
			Category:        skill.Manifest.Category,
			Keywords:        skill.Manifest.Keywords,
			RequiredColumns: skill.Manifest.RequiredColumns,
			Tools:           skill.Manifest.Tools,
			Enabled:         skill.Manifest.Enabled,
			Icon:            skill.Manifest.Icon,
			Tags:            skill.Manifest.Tags,
		})
	}

	return result, nil
}

// GetEnabledSkills 返回仅启用的技�
func (s *SkillFacadeService) GetEnabledSkills() ([]SkillInfo, error) {
	skillManager := s.getSkillManager()
	if skillManager == nil {
		return nil, WrapError("skill", "GetEnabledSkills", fmt.Errorf("eino service or skill manager not available"))
	}

	skills := skillManager.ListEnabledSkills()
	result := make([]SkillInfo, 0, len(skills))

	for _, skill := range skills {
		result = append(result, SkillInfo{
			ID:              skill.Manifest.ID,
			Name:            skill.Manifest.Name,
			Description:     skill.Manifest.Description,
			Version:         skill.Manifest.Version,
			Author:          skill.Manifest.Author,
			Category:        skill.Manifest.Category,
			Keywords:        skill.Manifest.Keywords,
			RequiredColumns: skill.Manifest.RequiredColumns,
			Tools:           skill.Manifest.Tools,
			Enabled:         skill.Manifest.Enabled,
			Icon:            skill.Manifest.Icon,
			Tags:            skill.Manifest.Tags,
		})
	}

	return result, nil
}

// GetSkillCategories 返回所有技能分�
func (s *SkillFacadeService) GetSkillCategories() ([]string, error) {
	skillManager := s.getSkillManager()
	if skillManager == nil {
		return nil, WrapError("skill", "GetSkillCategories", fmt.Errorf("eino service or skill manager not available"))
	}

	return skillManager.GetCategories(), nil
}

// --- Skill Mutation Methods ---

// EnableSkill 启用指定技�
func (s *SkillFacadeService) EnableSkill(skillID string) error {
	// Check if analysis is in progress
	if s.chatFacadeService != nil && s.chatFacadeService.HasActiveAnalysis() {
		return fmt.Errorf("cannot enable skill while analysis is in progress")
	}

	// Use skillService if available (for new skill management)
	if s.skillService != nil {
		if err := s.skillService.EnableSkill(skillID); err != nil {
			return err
		}
		// Reload skills in agent after enabling
		return s.ReloadSkills()
	}

	// Fallback to einoService for backward compatibility
	skillManager := s.getSkillManager()
	if skillManager == nil {
		return WrapError("skill", "EnableSkill", fmt.Errorf("skill service not initialized"))
	}

	return skillManager.EnableSkill(skillID)
}

// DisableSkill 禁用指定技�
func (s *SkillFacadeService) DisableSkill(skillID string) error {
	// Check if analysis is in progress
	if s.chatFacadeService != nil && s.chatFacadeService.HasActiveAnalysis() {
		return fmt.Errorf("cannot disable skill while analysis is in progress")
	}

	// Use skillService if available (for new skill management)
	if s.skillService != nil {
		if err := s.skillService.DisableSkill(skillID); err != nil {
			return err
		}
		// Reload skills in agent after disabling
		return s.ReloadSkills()
	}

	// Fallback to einoService for backward compatibility
	skillManager := s.getSkillManager()
	if skillManager == nil {
		return WrapError("skill", "DisableSkill", fmt.Errorf("skill service not initialized"))
	}

	return skillManager.DisableSkill(skillID)
}

// DeleteSkill 删除指定技能（移除目录和配置）
func (s *SkillFacadeService) DeleteSkill(skillID string) error {
	// Check if analysis is in progress
	if s.chatFacadeService != nil && s.chatFacadeService.HasActiveAnalysis() {
		return fmt.Errorf("cannot delete skill while analysis is in progress")
	}

	// Use skillService if available (for new skill management)
	if s.skillService != nil {
		if err := s.skillService.DeleteSkill(skillID); err != nil {
			return err
		}
		// Try to reload skills in agent after deleting, but don't fail if it errors
		if err := s.ReloadSkills(); err != nil {
			s.log(fmt.Sprintf("[SKILLS] Warning: Failed to reload skills after deletion: %v", err))
		}
		return nil
	}

	return WrapError("skill", "DeleteSkill", fmt.Errorf("skill service not initialized"))
}

// ReloadSkills 从磁盘重新加载所有技�
func (s *SkillFacadeService) ReloadSkills() error {
	skillManager := s.getSkillManager()
	if skillManager == nil {
		return WrapError("skill", "ReloadSkills", fmt.Errorf("eino service or skill manager not available"))
	}

	return skillManager.ReloadSkills()
}

// --- Skill Service Methods ---

// ListSkills 返回所有已安装的技能（通过 agent.SkillService�
func (s *SkillFacadeService) ListSkills() ([]agent.Skill, error) {
	if s.skillService == nil {
		return nil, WrapError("skill", "ListSkills", fmt.Errorf("skill service not initialized"))
	}
	return s.skillService.ListSkills()
}

// InstallSkillsFromZip �ZIP 文件安装技能，打开文件对话框让用户选择 ZIP 文件
func (s *SkillFacadeService) InstallSkillsFromZip() ([]string, error) {
	if s.skillService == nil {
		return nil, WrapError("skill", "InstallSkillsFromZip", fmt.Errorf("skill service not initialized"))
	}

	// Open file dialog to select ZIP file
	zipPath, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select Skills ZIP Package",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "ZIP Files (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open file dialog: %v", err)
	}

	if zipPath == "" {
		return nil, fmt.Errorf("no file selected")
	}

	s.log(fmt.Sprintf("[SKILLS] Installing from: %s", zipPath))

	// Install skills from ZIP
	installed, err := s.skillService.InstallFromZip(zipPath)
	if err != nil {
		s.log(fmt.Sprintf("[SKILLS] Installation failed: %v", err))
		return nil, err
	}

	s.log(fmt.Sprintf("[SKILLS] Successfully installed: %v", installed))
	return installed, nil
}
