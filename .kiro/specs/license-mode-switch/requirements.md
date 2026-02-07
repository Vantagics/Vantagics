# Requirements Document

## Introduction

本功能在关于对话框（AboutModal）中添加授权类型切换按钮，允许用户在开源模式和商业模式之间切换。开源模式下用户可以切换到商业模式（打开激活对话框），商业模式下用户可以切换到开源模式（取消激活）。切换操作需要确认对话框以避免误操作。

## Glossary

- **AboutModal**: 关于对话框组件，显示应用版本、授权状态等信息
- **ActivationModal**: 激活对话框组件，用于输入序列号激活商业授权
- **License_Mode_Switch**: 授权模式切换按钮，用于在开源模式和商业模式之间切换
- **Open_Source_Mode**: 开源模式，`activated: false`，用户需自行配置 LLM
- **Commercial_Mode**: 商业模式，`activated: true`，包括试用版和正式版
- **Confirmation_Dialog**: 确认对话框，用于防止用户误操作

## Requirements

### Requirement 1: 显示授权模式切换按钮

**User Story:** As a user, I want to see a mode switch button in the AboutModal, so that I can easily switch between open source and commercial modes.

#### Acceptance Criteria

1. WHEN the AboutModal is opened, THE License_Mode_Switch SHALL display a switch button next to the working mode label
2. WHEN the current mode is open source (activated: false), THE License_Mode_Switch SHALL display the button text as "切换到商业模式" in Chinese or "Switch to Commercial" in English
3. WHEN the current mode is commercial (activated: true), THE License_Mode_Switch SHALL display the button text as "切换到开源模式" in Chinese or "Switch to Open Source" in English
4. THE License_Mode_Switch SHALL use consistent styling with other buttons in the AboutModal

### Requirement 2: 切换到商业模式

**User Story:** As a user in open source mode, I want to switch to commercial mode, so that I can use cloud LLM services without manual configuration.

#### Acceptance Criteria

1. WHEN a user clicks the "Switch to Commercial" button, THE License_Mode_Switch SHALL display a confirmation dialog
2. WHEN the user confirms the switch to commercial mode, THE License_Mode_Switch SHALL close the AboutModal and open the ActivationModal
3. WHEN the user cancels the confirmation dialog, THE License_Mode_Switch SHALL keep the AboutModal open and maintain the current state
4. WHEN the ActivationModal is closed after successful activation, THE AboutModal SHALL refresh the activation status to reflect the new commercial mode

### Requirement 3: 切换到开源模式

**User Story:** As a user in commercial mode, I want to switch to open source mode, so that I can use my own LLM configuration.

#### Acceptance Criteria

1. WHEN a user clicks the "Switch to Open Source" button, THE License_Mode_Switch SHALL display a confirmation dialog with a warning message
2. WHEN the user confirms the switch to open source mode, THE License_Mode_Switch SHALL call the DeactivateLicense() backend method
3. WHEN the DeactivateLicense() call succeeds, THE License_Mode_Switch SHALL refresh the activation status in the AboutModal
4. WHEN the user cancels the confirmation dialog, THE License_Mode_Switch SHALL keep the current commercial mode state unchanged
5. IF the DeactivateLicense() call fails, THEN THE License_Mode_Switch SHALL display an error message to the user

### Requirement 4: 确认对话框

**User Story:** As a user, I want to see a confirmation dialog before switching modes, so that I can avoid accidental mode changes.

#### Acceptance Criteria

1. THE Confirmation_Dialog SHALL display a clear title indicating the action (e.g., "切换到商业模式" or "切换到开源模式")
2. THE Confirmation_Dialog SHALL display a description explaining the consequences of the switch
3. WHEN switching to open source mode, THE Confirmation_Dialog SHALL warn that the current license will be deactivated
4. THE Confirmation_Dialog SHALL provide "Confirm" and "Cancel" buttons
5. THE Confirmation_Dialog SHALL support both Chinese and English languages based on the current language setting

### Requirement 5: 国际化支持

**User Story:** As a user, I want the mode switch feature to support both Chinese and English, so that I can use it in my preferred language.

#### Acceptance Criteria

1. THE License_Mode_Switch SHALL use the existing useLanguage hook for translations
2. THE License_Mode_Switch SHALL add all new text strings to the i18n.ts translations file
3. WHEN the language is Chinese, THE License_Mode_Switch SHALL display all text in Chinese
4. WHEN the language is English, THE License_Mode_Switch SHALL display all text in English
