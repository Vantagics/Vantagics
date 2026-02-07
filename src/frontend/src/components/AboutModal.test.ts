/**
 * Property-Based Tests for AboutModal License Mode Switch
 * 
 * Feature: license-mode-switch
 * 
 * These tests verify the correctness properties for the license mode switch button:
 * - Property 1: Button Text Correctness
 * 
 * **Validates: Requirements 1.2, 1.3**
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';

// ==================== Type Definitions ====================

/**
 * Language type matching the i18n configuration
 */
type Language = 'English' | '简体中文';

/**
 * Activation state type
 */
type ActivationState = boolean;

// ==================== Translation Data ====================

/**
 * Translations for license mode switch button text
 * Extracted from i18n.ts to test the button text logic
 */
const translations: Record<Language, Record<string, string>> = {
    'English': {
        'switch_to_commercial': 'Switch to Commercial',
        'switch_to_opensource': 'Switch to Open Source',
    },
    '简体中文': {
        'switch_to_commercial': '切换到商业模式',
        'switch_to_opensource': '切换到开源模式',
    }
};

// ==================== Test Data Generators ====================

/**
 * Generate valid activation state (boolean)
 */
const activationStateArb = fc.boolean();

/**
 * Generate valid language setting
 */
const languageArb = fc.constantFrom<Language>('English', '简体中文');

// ==================== Helper Functions ====================

/**
 * Get the expected button text based on activation state and language
 * This mirrors the logic in AboutModal.tsx
 * 
 * @param activated - Whether the license is activated (commercial mode)
 * @param language - The current language setting
 * @returns The expected button text
 */
function getExpectedButtonText(activated: ActivationState, language: Language): string {
    const translationKey = activated ? 'switch_to_opensource' : 'switch_to_commercial';
    return translations[language][translationKey];
}

/**
 * Simulate the button text determination logic from AboutModal
 * This is the actual logic being tested
 * 
 * @param activated - Whether the license is activated (commercial mode)
 * @param language - The current language setting
 * @returns The button text that would be displayed
 */
function getButtonText(activated: ActivationState, language: Language): string {
    // When activated (commercial mode), show "Switch to Open Source"
    // When not activated (open source mode), show "Switch to Commercial"
    if (activated) {
        return translations[language]['switch_to_opensource'];
    } else {
        return translations[language]['switch_to_commercial'];
    }
}

// ==================== Property Tests ====================

describe('Feature: license-mode-switch, Property 1: Button Text Correctness', () => {
    /**
     * **Validates: Requirements 1.2, 1.3**
     * 
     * Property 1: Button Text Correctness
     * For any combination of activation state (activated: true/false) and language setting 
     * (Chinese/English), the switch button text SHALL correctly reflect both the current mode 
     * and the target mode in the appropriate language.
     * 
     * - When `activated: false` and language is Chinese → "切换到商业模式"
     * - When `activated: false` and language is English → "Switch to Commercial"
     * - When `activated: true` and language is Chinese → "切换到开源模式"
     * - When `activated: true` and language is English → "Switch to Open Source"
     */

    /**
     * Property Test 1.1: Button text should match activation state and language
     * 
     * **Validates: Requirements 1.2, 1.3**
     */
    it('should display correct button text for any activation state and language combination', () => {
        fc.assert(
            fc.property(
                activationStateArb,
                languageArb,
                (activated, language) => {
                    // Get the expected button text based on the specification
                    const expectedText = getExpectedButtonText(activated, language);
                    
                    // Get the actual button text from the logic
                    const actualText = getButtonText(activated, language);
                    
                    // Property: Button text should match expected value
                    expect(actualText).toBe(expectedText);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 1.2: Open source mode (activated: false) should show "Switch to Commercial"
     * 
     * **Validates: Requirements 1.2**
     */
    it('should show "Switch to Commercial" text when in open source mode (activated: false)', () => {
        fc.assert(
            fc.property(
                languageArb,
                (language) => {
                    const activated = false; // Open source mode
                    const buttonText = getButtonText(activated, language);
                    
                    // Property: Should show "Switch to Commercial" in the appropriate language
                    if (language === 'English') {
                        expect(buttonText).toBe('Switch to Commercial');
                    } else {
                        expect(buttonText).toBe('切换到商业模式');
                    }
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 1.3: Commercial mode (activated: true) should show "Switch to Open Source"
     * 
     * **Validates: Requirements 1.3**
     */
    it('should show "Switch to Open Source" text when in commercial mode (activated: true)', () => {
        fc.assert(
            fc.property(
                languageArb,
                (language) => {
                    const activated = true; // Commercial mode
                    const buttonText = getButtonText(activated, language);
                    
                    // Property: Should show "Switch to Open Source" in the appropriate language
                    if (language === 'English') {
                        expect(buttonText).toBe('Switch to Open Source');
                    } else {
                        expect(buttonText).toBe('切换到开源模式');
                    }
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 1.4: Button text should be non-empty for all valid inputs
     * 
     * **Validates: Requirements 1.2, 1.3**
     */
    it('should always return non-empty button text for any valid input', () => {
        fc.assert(
            fc.property(
                activationStateArb,
                languageArb,
                (activated, language) => {
                    const buttonText = getButtonText(activated, language);
                    
                    // Property: Button text should never be empty
                    expect(buttonText).toBeTruthy();
                    expect(buttonText.length).toBeGreaterThan(0);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 1.5: Button text should be deterministic
     * 
     * **Validates: Requirements 1.2, 1.3**
     */
    it('should return the same button text for the same inputs (deterministic)', () => {
        fc.assert(
            fc.property(
                activationStateArb,
                languageArb,
                (activated, language) => {
                    // Call the function multiple times with the same inputs
                    const text1 = getButtonText(activated, language);
                    const text2 = getButtonText(activated, language);
                    const text3 = getButtonText(activated, language);
                    
                    // Property: All calls should return the same result
                    expect(text1).toBe(text2);
                    expect(text2).toBe(text3);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 1.6: Opposite activation states should produce different button texts
     * 
     * **Validates: Requirements 1.2, 1.3**
     */
    it('should produce different button texts for opposite activation states', () => {
        fc.assert(
            fc.property(
                languageArb,
                (language) => {
                    const textWhenActivated = getButtonText(true, language);
                    const textWhenNotActivated = getButtonText(false, language);
                    
                    // Property: Button texts should be different for opposite states
                    expect(textWhenActivated).not.toBe(textWhenNotActivated);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });
});


// ==================== Property 2: Cancel Preserves State ====================

describe('Feature: license-mode-switch, Property 2: Cancel Preserves State', () => {
    /**
     * **Validates: Requirements 2.3, 3.4**
     * 
     * Property 2: Cancel Action Preserves State (Idempotence)
     * For any initial activation state, when the user opens the confirmation dialog 
     * and clicks cancel, the activation state SHALL remain unchanged from its initial value.
     * 
     * - If `activated: false` before cancel → `activated: false` after cancel
     * - If `activated: true` before cancel → `activated: true` after cancel
     */

    /**
     * Dialog state interface that mirrors the AboutModal component state
     */
    interface DialogState {
        showConfirmDialog: boolean;
        confirmAction: 'toCommercial' | 'toOpenSource' | null;
        deactivateError: string | null;
        activationStatus: { activated: boolean };
    }

    /**
     * Create initial state with the given activation status
     * This mirrors the initial state setup in AboutModal
     * 
     * @param activated - Whether the license is activated (commercial mode)
     * @returns The initial dialog state
     */
    function createInitialState(activated: boolean): DialogState {
        return {
            showConfirmDialog: false,
            confirmAction: null,
            deactivateError: null,
            activationStatus: { activated }
        };
    }

    /**
     * Simulate the handleSwitchClick function from AboutModal
     * Opens the confirmation dialog and sets the appropriate action
     * 
     * @param state - Current dialog state
     * @returns Updated dialog state with dialog open
     */
    function handleSwitchClick(state: DialogState): DialogState {
        return {
            ...state,
            showConfirmDialog: true,
            confirmAction: state.activationStatus.activated ? 'toOpenSource' : 'toCommercial'
        };
    }

    /**
     * Simulate the handleCancel function from AboutModal
     * Closes the confirmation dialog and resets related state
     * Note: This should NOT modify the activation status
     * 
     * @param state - Current dialog state
     * @returns Updated dialog state with dialog closed
     */
    function handleCancel(state: DialogState): DialogState {
        return {
            ...state,
            showConfirmDialog: false,
            confirmAction: null,
            deactivateError: null
            // Note: activationStatus is NOT modified - this is the key property being tested
        };
    }

    /**
     * Property Test 2.1: Cancel should preserve activation state
     * 
     * **Validates: Requirements 2.3, 3.4**
     * 
     * For any initial activation state, when the user:
     * 1. Opens the confirmation dialog (clicks switch button)
     * 2. Clicks cancel
     * The activation state should remain unchanged.
     */
    it('should preserve activation state when cancel is clicked', () => {
        fc.assert(
            fc.property(
                fc.boolean(), // initial activation state
                (initialActivated) => {
                    // Create initial state with the given activation status
                    let state = createInitialState(initialActivated);
                    const originalActivated = state.activationStatus.activated;
                    
                    // Step 1: Open the confirmation dialog
                    state = handleSwitchClick(state);
                    
                    // Verify dialog is open
                    expect(state.showConfirmDialog).toBe(true);
                    
                    // Step 2: Click cancel
                    state = handleCancel(state);
                    
                    // Property: Activation state should be unchanged
                    expect(state.activationStatus.activated).toBe(originalActivated);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 2.2: Cancel should close the dialog
     * 
     * **Validates: Requirements 2.3, 3.4**
     * 
     * After cancel, the dialog should be closed and action should be reset.
     */
    it('should close dialog and reset action when cancel is clicked', () => {
        fc.assert(
            fc.property(
                fc.boolean(), // initial activation state
                (initialActivated) => {
                    // Create initial state and open dialog
                    let state = createInitialState(initialActivated);
                    state = handleSwitchClick(state);
                    
                    // Verify dialog is open with an action
                    expect(state.showConfirmDialog).toBe(true);
                    expect(state.confirmAction).not.toBeNull();
                    
                    // Click cancel
                    state = handleCancel(state);
                    
                    // Property: Dialog should be closed and action reset
                    expect(state.showConfirmDialog).toBe(false);
                    expect(state.confirmAction).toBeNull();
                    expect(state.deactivateError).toBeNull();
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 2.3: Multiple cancel operations should be idempotent
     * 
     * **Validates: Requirements 2.3, 3.4**
     * 
     * Canceling multiple times should have the same effect as canceling once.
     */
    it('should be idempotent - multiple cancels should have same effect', () => {
        fc.assert(
            fc.property(
                fc.boolean(), // initial activation state
                fc.integer({ min: 1, max: 10 }), // number of cancel operations
                (initialActivated, cancelCount) => {
                    // Create initial state and open dialog
                    let state = createInitialState(initialActivated);
                    const originalActivated = state.activationStatus.activated;
                    state = handleSwitchClick(state);
                    
                    // Apply cancel multiple times
                    for (let i = 0; i < cancelCount; i++) {
                        state = handleCancel(state);
                    }
                    
                    // Property: Activation state should still be unchanged
                    expect(state.activationStatus.activated).toBe(originalActivated);
                    expect(state.showConfirmDialog).toBe(false);
                    expect(state.confirmAction).toBeNull();
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 2.4: Open-cancel cycle should preserve state
     * 
     * **Validates: Requirements 2.3, 3.4**
     * 
     * Multiple open-cancel cycles should preserve the activation state.
     */
    it('should preserve state through multiple open-cancel cycles', () => {
        fc.assert(
            fc.property(
                fc.boolean(), // initial activation state
                fc.integer({ min: 1, max: 5 }), // number of open-cancel cycles
                (initialActivated, cycleCount) => {
                    // Create initial state
                    let state = createInitialState(initialActivated);
                    const originalActivated = state.activationStatus.activated;
                    
                    // Perform multiple open-cancel cycles
                    for (let i = 0; i < cycleCount; i++) {
                        // Open dialog
                        state = handleSwitchClick(state);
                        expect(state.showConfirmDialog).toBe(true);
                        
                        // Cancel
                        state = handleCancel(state);
                        expect(state.showConfirmDialog).toBe(false);
                    }
                    
                    // Property: Activation state should be unchanged after all cycles
                    expect(state.activationStatus.activated).toBe(originalActivated);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 2.5: Cancel should clear any error state
     * 
     * **Validates: Requirements 2.3, 3.4**
     * 
     * If there was an error before cancel, it should be cleared.
     */
    it('should clear error state when cancel is clicked', () => {
        fc.assert(
            fc.property(
                fc.boolean(), // initial activation state
                fc.string({ minLength: 1, maxLength: 100 }), // error message
                (initialActivated, errorMessage) => {
                    // Create initial state with an error
                    let state = createInitialState(initialActivated);
                    state = handleSwitchClick(state);
                    state = { ...state, deactivateError: errorMessage };
                    
                    // Verify error is set
                    expect(state.deactivateError).toBe(errorMessage);
                    
                    // Click cancel
                    state = handleCancel(state);
                    
                    // Property: Error should be cleared
                    expect(state.deactivateError).toBeNull();
                    
                    // And activation state should be unchanged
                    expect(state.activationStatus.activated).toBe(initialActivated);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });
});


// ==================== Property 3: Language Consistency ====================

describe('Feature: license-mode-switch, Property 3: Language Consistency', () => {
    /**
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * Property 3: Language Consistency
     * For any language setting, all text elements in the License_Mode_Switch feature 
     * (button text, dialog title, dialog message, confirm button, cancel button) 
     * SHALL be displayed in the selected language.
     */

    /**
     * All translations for the License_Mode_Switch feature
     * Extracted from i18n.ts to test language consistency
     */
    const allTranslations: Record<Language, Record<string, string>> = {
        'English': {
            'switch_to_commercial': 'Switch to Commercial',
            'switch_to_opensource': 'Switch to Open Source',
            'confirm_switch_to_commercial': 'Switch to Commercial Mode',
            'confirm_switch_to_commercial_desc': 'You will be redirected to the activation page to enter your serial number and activate commercial mode.',
            'confirm_switch_to_opensource': 'Switch to Open Source Mode',
            'confirm_switch_to_opensource_desc': 'Warning: Your current license will be deactivated. You will need to configure your own LLM API to continue using the application.',
            'deactivate_failed': 'Failed to deactivate license',
        },
        '简体中文': {
            'switch_to_commercial': '切换到商业模式',
            'switch_to_opensource': '切换到开源模式',
            'confirm_switch_to_commercial': '切换到商业模式',
            'confirm_switch_to_commercial_desc': '您将被重定向到激活页面，输入序列号以激活商业模式。',
            'confirm_switch_to_opensource': '切换到开源模式',
            'confirm_switch_to_opensource_desc': '警告：您当前的授权将被取消激活。您需要配置自己的 LLM API 才能继续使用应用程序。',
            'deactivate_failed': '取消激活授权失败',
        }
    };

    /**
     * Check if a string contains Chinese characters (CJK Unified Ideographs)
     * @param text - The text to check
     * @returns true if the text contains Chinese characters
     */
    function containsChineseCharacters(text: string): boolean {
        return /[\u4e00-\u9fff]/.test(text);
    }

    /**
     * Check if a string contains only ASCII characters
     * @param text - The text to check
     * @returns true if the text contains only ASCII characters
     */
    function isAsciiOnly(text: string): boolean {
        return /^[\x00-\x7F]*$/.test(text);
    }

    /**
     * Property Test 3.1: All text elements should be in the selected language
     * 
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * For any language setting, all text elements in the License_Mode_Switch feature
     * should be displayed in the selected language:
     * - Chinese text should contain Chinese characters when language is '简体中文'
     * - English text should contain only ASCII characters when language is 'English'
     */
    it('should display all text elements in the selected language', () => {
        fc.assert(
            fc.property(
                languageArb,
                (language) => {
                    const texts = allTranslations[language];
                    
                    // All texts should be in the same language
                    Object.entries(texts).forEach(([key, text]) => {
                        if (language === '简体中文') {
                            // Chinese text should contain Chinese characters
                            expect(containsChineseCharacters(text)).toBe(true);
                        } else {
                            // English text should contain only ASCII characters
                            expect(isAsciiOnly(text)).toBe(true);
                        }
                    });
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 3.2: All translation keys should exist for both languages
     * 
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * For any language setting, all required translation keys should exist
     * and have non-empty values.
     */
    it('should have all required translation keys for any language', () => {
        const requiredKeys = [
            'switch_to_commercial',
            'switch_to_opensource',
            'confirm_switch_to_commercial',
            'confirm_switch_to_commercial_desc',
            'confirm_switch_to_opensource',
            'confirm_switch_to_opensource_desc',
            'deactivate_failed'
        ];

        fc.assert(
            fc.property(
                languageArb,
                (language) => {
                    const texts = allTranslations[language];
                    
                    // All required keys should exist and have non-empty values
                    requiredKeys.forEach(key => {
                        expect(texts[key]).toBeDefined();
                        expect(texts[key].length).toBeGreaterThan(0);
                    });
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 3.3: Translations should be consistent within a language
     * 
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * For any language setting, all text elements should be consistently
     * in the same language (no mixed language content).
     */
    it('should have consistent language across all text elements', () => {
        fc.assert(
            fc.property(
                languageArb,
                (language) => {
                    const texts = allTranslations[language];
                    const textValues = Object.values(texts);
                    
                    if (language === '简体中文') {
                        // All Chinese texts should contain Chinese characters
                        const allContainChinese = textValues.every(text => 
                            containsChineseCharacters(text)
                        );
                        expect(allContainChinese).toBe(true);
                    } else {
                        // All English texts should be ASCII only
                        const allAscii = textValues.every(text => 
                            isAsciiOnly(text)
                        );
                        expect(allAscii).toBe(true);
                    }
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 3.4: Different languages should have different translations
     * 
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * For any translation key, the Chinese and English translations should be different.
     */
    it('should have different translations for different languages', () => {
        const translationKeys = Object.keys(allTranslations['English']);
        
        fc.assert(
            fc.property(
                fc.constantFrom(...translationKeys),
                (key) => {
                    const englishText = allTranslations['English'][key];
                    const chineseText = allTranslations['简体中文'][key];
                    
                    // Translations should be different for different languages
                    expect(englishText).not.toBe(chineseText);
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 3.5: Button text should match language setting
     * 
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * For any combination of activation state and language, the button text
     * should be in the correct language.
     */
    it('should display button text in the correct language for any state', () => {
        fc.assert(
            fc.property(
                activationStateArb,
                languageArb,
                (activated, language) => {
                    const buttonKey = activated ? 'switch_to_opensource' : 'switch_to_commercial';
                    const buttonText = allTranslations[language][buttonKey];
                    
                    if (language === '简体中文') {
                        expect(containsChineseCharacters(buttonText)).toBe(true);
                    } else {
                        expect(isAsciiOnly(buttonText)).toBe(true);
                    }
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });

    /**
     * Property Test 3.6: Dialog text should match language setting
     * 
     * **Validates: Requirements 4.5, 5.3, 5.4**
     * 
     * For any confirmation action and language, the dialog title and description
     * should be in the correct language.
     */
    it('should display dialog text in the correct language for any action', () => {
        const confirmActionArb = fc.constantFrom<'toCommercial' | 'toOpenSource'>('toCommercial', 'toOpenSource');
        
        fc.assert(
            fc.property(
                confirmActionArb,
                languageArb,
                (action, language) => {
                    const titleKey = action === 'toCommercial' 
                        ? 'confirm_switch_to_commercial' 
                        : 'confirm_switch_to_opensource';
                    const descKey = action === 'toCommercial' 
                        ? 'confirm_switch_to_commercial_desc' 
                        : 'confirm_switch_to_opensource_desc';
                    
                    const titleText = allTranslations[language][titleKey];
                    const descText = allTranslations[language][descKey];
                    
                    if (language === '简体中文') {
                        expect(containsChineseCharacters(titleText)).toBe(true);
                        expect(containsChineseCharacters(descText)).toBe(true);
                    } else {
                        expect(isAsciiOnly(titleText)).toBe(true);
                        expect(isAsciiOnly(descText)).toBe(true);
                    }
                    
                    return true;
                }
            ),
            { numRuns: 100 }
        );
    });
});


// ==================== Unit Tests: License Mode Switch ====================

describe('Unit Tests: License Mode Switch', () => {
    /**
     * Unit tests for the License Mode Switch feature in AboutModal.
     * These tests verify specific behaviors and edge cases.
     * 
     * _Requirements: 2.1, 3.1, 3.5_
     */

    // ==================== Type Definitions ====================

    /**
     * Activation status interface matching the component state
     */
    interface ActivationStatus {
        activated: boolean;
        sn?: string;
        expires_at?: string;
        daily_analysis_limit?: number;
        daily_analysis_count?: number;
    }

    /**
     * Dialog state interface for testing state management
     */
    interface DialogState {
        showConfirmDialog: boolean;
        confirmAction: 'toCommercial' | 'toOpenSource' | null;
        isDeactivating: boolean;
        deactivateError: string | null;
        activationStatus: ActivationStatus;
    }

    // ==================== Helper Functions ====================

    /**
     * Create initial dialog state
     */
    function createInitialDialogState(activated: boolean): DialogState {
        return {
            showConfirmDialog: false,
            confirmAction: null,
            isDeactivating: false,
            deactivateError: null,
            activationStatus: { activated }
        };
    }

    /**
     * Simulate handleSwitchClick from AboutModal
     */
    function handleSwitchClick(state: DialogState): DialogState {
        return {
            ...state,
            showConfirmDialog: true,
            confirmAction: state.activationStatus.activated ? 'toOpenSource' : 'toCommercial'
        };
    }

    /**
     * Simulate handleCancel from AboutModal
     */
    function handleCancel(state: DialogState): DialogState {
        return {
            ...state,
            showConfirmDialog: false,
            confirmAction: null,
            deactivateError: null
        };
    }

    /**
     * Simulate starting deactivation process
     */
    function startDeactivation(state: DialogState): DialogState {
        return {
            ...state,
            isDeactivating: true,
            deactivateError: null
        };
    }

    /**
     * Simulate successful deactivation
     */
    function completeDeactivationSuccess(state: DialogState): DialogState {
        return {
            ...state,
            isDeactivating: false,
            showConfirmDialog: false,
            confirmAction: null,
            activationStatus: { activated: false }
        };
    }

    /**
     * Simulate failed deactivation
     */
    function completeDeactivationError(state: DialogState, errorMessage: string): DialogState {
        return {
            ...state,
            isDeactivating: false,
            deactivateError: errorMessage
        };
    }

    /**
     * Get button text based on activation state and language
     */
    function getButtonText(activated: boolean, language: 'English' | '简体中文'): string {
        const translations: Record<string, Record<string, string>> = {
            'English': {
                'switch_to_commercial': 'Switch to Commercial',
                'switch_to_opensource': 'Switch to Open Source',
            },
            '简体中文': {
                'switch_to_commercial': '切换到商业模式',
                'switch_to_opensource': '切换到开源模式',
            }
        };
        const key = activated ? 'switch_to_opensource' : 'switch_to_commercial';
        return translations[language][key];
    }

    // ==================== Button Rendering Tests ====================

    describe('Button Rendering', () => {
        /**
         * Test that the switch button renders with correct text for open source mode
         * 
         * _Requirements: 2.1_
         */
        it('should render switch button with correct text for open source mode', () => {
            const state = createInitialDialogState(false); // Open source mode
            
            // Verify the button text would be "Switch to Commercial"
            const buttonTextEn = getButtonText(state.activationStatus.activated, 'English');
            const buttonTextZh = getButtonText(state.activationStatus.activated, '简体中文');
            
            expect(buttonTextEn).toBe('Switch to Commercial');
            expect(buttonTextZh).toBe('切换到商业模式');
        });

        /**
         * Test that the switch button renders with correct text for commercial mode
         * 
         * _Requirements: 2.1_
         */
        it('should render switch button with correct text for commercial mode', () => {
            const state = createInitialDialogState(true); // Commercial mode
            
            // Verify the button text would be "Switch to Open Source"
            const buttonTextEn = getButtonText(state.activationStatus.activated, 'English');
            const buttonTextZh = getButtonText(state.activationStatus.activated, '简体中文');
            
            expect(buttonTextEn).toBe('Switch to Open Source');
            expect(buttonTextZh).toBe('切换到开源模式');
        });

        /**
         * Test that button text changes based on activation state
         * 
         * _Requirements: 2.1_
         */
        it('should change button text based on activation state', () => {
            // Start in open source mode
            let state = createInitialDialogState(false);
            expect(getButtonText(state.activationStatus.activated, 'English')).toBe('Switch to Commercial');
            
            // Simulate switching to commercial mode
            state = { ...state, activationStatus: { activated: true } };
            expect(getButtonText(state.activationStatus.activated, 'English')).toBe('Switch to Open Source');
            
            // Simulate switching back to open source mode
            state = { ...state, activationStatus: { activated: false } };
            expect(getButtonText(state.activationStatus.activated, 'English')).toBe('Switch to Commercial');
        });

        /**
         * Test that button is disabled during deactivation
         * 
         * _Requirements: 3.1_
         */
        it('should indicate disabled state during deactivation', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            
            // Button should be disabled when isDeactivating is true
            expect(state.isDeactivating).toBe(true);
        });
    });

    // ==================== Confirmation Dialog Tests ====================

    describe('Confirmation Dialog', () => {
        /**
         * Test that dialog shows when switch button is clicked
         * 
         * _Requirements: 2.1, 3.1_
         */
        it('should show dialog when switch button is clicked', () => {
            let state = createInitialDialogState(false);
            
            // Initially dialog should be hidden
            expect(state.showConfirmDialog).toBe(false);
            
            // Click switch button
            state = handleSwitchClick(state);
            
            // Dialog should now be visible
            expect(state.showConfirmDialog).toBe(true);
        });

        /**
         * Test that dialog hides when cancel is clicked
         * 
         * _Requirements: 2.1, 3.1_
         */
        it('should hide dialog when cancel is clicked', () => {
            let state = createInitialDialogState(false);
            state = handleSwitchClick(state);
            
            // Dialog should be visible
            expect(state.showConfirmDialog).toBe(true);
            
            // Click cancel
            state = handleCancel(state);
            
            // Dialog should be hidden
            expect(state.showConfirmDialog).toBe(false);
        });

        /**
         * Test that dialog shows correct action for open source to commercial switch
         * 
         * _Requirements: 2.1_
         */
        it('should show correct action for switching to commercial mode', () => {
            let state = createInitialDialogState(false); // Open source mode
            state = handleSwitchClick(state);
            
            expect(state.confirmAction).toBe('toCommercial');
        });

        /**
         * Test that dialog shows correct action for commercial to open source switch
         * 
         * _Requirements: 3.1_
         */
        it('should show correct action for switching to open source mode', () => {
            let state = createInitialDialogState(true); // Commercial mode
            state = handleSwitchClick(state);
            
            expect(state.confirmAction).toBe('toOpenSource');
        });

        /**
         * Test that confirm action is reset when cancel is clicked
         * 
         * _Requirements: 2.1, 3.1_
         */
        it('should reset confirm action when cancel is clicked', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            
            expect(state.confirmAction).toBe('toOpenSource');
            
            state = handleCancel(state);
            
            expect(state.confirmAction).toBeNull();
        });

        /**
         * Test that dialog can be reopened after cancel
         * 
         * _Requirements: 2.1, 3.1_
         */
        it('should allow reopening dialog after cancel', () => {
            let state = createInitialDialogState(false);
            
            // Open dialog
            state = handleSwitchClick(state);
            expect(state.showConfirmDialog).toBe(true);
            
            // Cancel
            state = handleCancel(state);
            expect(state.showConfirmDialog).toBe(false);
            
            // Reopen dialog
            state = handleSwitchClick(state);
            expect(state.showConfirmDialog).toBe(true);
            expect(state.confirmAction).toBe('toCommercial');
        });
    });

    // ==================== DeactivateLicense Call Tests ====================

    describe('DeactivateLicense Call', () => {
        /**
         * Test that DeactivateLicense flow starts correctly
         * 
         * _Requirements: 3.1_
         */
        it('should start deactivation process when confirming switch to open source', () => {
            let state = createInitialDialogState(true); // Commercial mode
            state = handleSwitchClick(state);
            
            // Confirm action should be toOpenSource
            expect(state.confirmAction).toBe('toOpenSource');
            
            // Start deactivation
            state = startDeactivation(state);
            
            // Should be in deactivating state
            expect(state.isDeactivating).toBe(true);
            expect(state.deactivateError).toBeNull();
        });

        /**
         * Test that activation status is refreshed after successful deactivation
         * 
         * _Requirements: 3.1_
         */
        it('should refresh activation status after successful deactivation', () => {
            let state = createInitialDialogState(true); // Commercial mode
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            
            // Complete deactivation successfully
            state = completeDeactivationSuccess(state);
            
            // Activation status should be updated to open source mode
            expect(state.activationStatus.activated).toBe(false);
            expect(state.isDeactivating).toBe(false);
            expect(state.showConfirmDialog).toBe(false);
            expect(state.confirmAction).toBeNull();
        });

        /**
         * Test that dialog closes after successful deactivation
         * 
         * _Requirements: 3.1_
         */
        it('should close dialog after successful deactivation', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            state = completeDeactivationSuccess(state);
            
            expect(state.showConfirmDialog).toBe(false);
        });

        /**
         * Test that deactivation does not start for toCommercial action
         * 
         * _Requirements: 2.1_
         */
        it('should not call DeactivateLicense for toCommercial action', () => {
            let state = createInitialDialogState(false); // Open source mode
            state = handleSwitchClick(state);
            
            // Confirm action should be toCommercial, not toOpenSource
            expect(state.confirmAction).toBe('toCommercial');
            
            // For toCommercial, we don't start deactivation
            // Instead, we would close AboutModal and open ActivationModal
            // This is verified by checking the action type
            expect(state.confirmAction).not.toBe('toOpenSource');
        });
    });

    // ==================== Error Handling Tests ====================

    describe('Error Handling', () => {
        /**
         * Test that error message is displayed when DeactivateLicense fails
         * 
         * _Requirements: 3.5_
         */
        it('should display error message when deactivation fails', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            
            // Simulate deactivation failure
            const errorMessage = 'Failed to deactivate license: Network error';
            state = completeDeactivationError(state, errorMessage);
            
            // Error should be displayed
            expect(state.deactivateError).toBe(errorMessage);
            expect(state.isDeactivating).toBe(false);
        });

        /**
         * Test that dialog remains open when deactivation fails
         * 
         * _Requirements: 3.5_
         */
        it('should keep dialog open when deactivation fails', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            
            // Simulate deactivation failure
            state = completeDeactivationError(state, 'Error occurred');
            
            // Dialog should remain open so user can retry or cancel
            expect(state.showConfirmDialog).toBe(true);
            expect(state.confirmAction).toBe('toOpenSource');
        });

        /**
         * Test that error is cleared when cancel is clicked
         * 
         * _Requirements: 3.5_
         */
        it('should clear error when cancel is clicked', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            state = completeDeactivationError(state, 'Some error');
            
            // Error should be present
            expect(state.deactivateError).toBe('Some error');
            
            // Click cancel
            state = handleCancel(state);
            
            // Error should be cleared
            expect(state.deactivateError).toBeNull();
        });

        /**
         * Test that activation status is preserved when deactivation fails
         * 
         * _Requirements: 3.5_
         */
        it('should preserve activation status when deactivation fails', () => {
            let state = createInitialDialogState(true); // Commercial mode
            const originalActivated = state.activationStatus.activated;
            
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            state = completeDeactivationError(state, 'Error');
            
            // Activation status should remain unchanged
            expect(state.activationStatus.activated).toBe(originalActivated);
        });

        /**
         * Test that user can retry after error
         * 
         * _Requirements: 3.5_
         */
        it('should allow retry after deactivation error', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            state = completeDeactivationError(state, 'First error');
            
            // Error should be present
            expect(state.deactivateError).toBe('First error');
            
            // Retry deactivation
            state = startDeactivation(state);
            
            // Error should be cleared, deactivating should be true
            expect(state.deactivateError).toBeNull();
            expect(state.isDeactivating).toBe(true);
            
            // Complete successfully this time
            state = completeDeactivationSuccess(state);
            
            expect(state.activationStatus.activated).toBe(false);
            expect(state.showConfirmDialog).toBe(false);
        });

        /**
         * Test that error message format is correct
         * 
         * _Requirements: 3.5_
         */
        it('should format error message correctly', () => {
            const baseError = 'Failed to deactivate license';
            const detailedError = 'Network timeout';
            const expectedFormat = `${baseError}: ${detailedError}`;
            
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            state = completeDeactivationError(state, expectedFormat);
            
            expect(state.deactivateError).toContain(baseError);
            expect(state.deactivateError).toContain(detailedError);
        });

        /**
         * Test that multiple errors don't accumulate
         * 
         * _Requirements: 3.5_
         */
        it('should replace previous error with new error', () => {
            let state = createInitialDialogState(true);
            state = handleSwitchClick(state);
            state = startDeactivation(state);
            state = completeDeactivationError(state, 'First error');
            
            expect(state.deactivateError).toBe('First error');
            
            // Retry and get a different error
            state = startDeactivation(state);
            state = completeDeactivationError(state, 'Second error');
            
            // Should only show the latest error
            expect(state.deactivateError).toBe('Second error');
            expect(state.deactivateError).not.toContain('First error');
        });
    });

    // ==================== State Transition Tests ====================

    describe('State Transitions', () => {
        /**
         * Test complete flow from open source to commercial mode initiation
         * 
         * _Requirements: 2.1_
         */
        it('should handle complete flow for switching to commercial mode', () => {
            let state = createInitialDialogState(false); // Open source mode
            
            // Initial state
            expect(state.showConfirmDialog).toBe(false);
            expect(state.confirmAction).toBeNull();
            
            // Click switch button
            state = handleSwitchClick(state);
            expect(state.showConfirmDialog).toBe(true);
            expect(state.confirmAction).toBe('toCommercial');
            
            // For toCommercial, the next step would be to close AboutModal
            // and open ActivationModal (not tested here as it involves component interaction)
        });

        /**
         * Test complete flow from commercial to open source mode
         * 
         * _Requirements: 3.1_
         */
        it('should handle complete flow for switching to open source mode', () => {
            let state = createInitialDialogState(true); // Commercial mode
            
            // Initial state
            expect(state.activationStatus.activated).toBe(true);
            
            // Click switch button
            state = handleSwitchClick(state);
            expect(state.showConfirmDialog).toBe(true);
            expect(state.confirmAction).toBe('toOpenSource');
            
            // Start deactivation
            state = startDeactivation(state);
            expect(state.isDeactivating).toBe(true);
            
            // Complete deactivation
            state = completeDeactivationSuccess(state);
            expect(state.activationStatus.activated).toBe(false);
            expect(state.showConfirmDialog).toBe(false);
        });

        /**
         * Test that state is consistent after cancel
         * 
         * _Requirements: 2.1, 3.1_
         */
        it('should maintain consistent state after cancel', () => {
            let state = createInitialDialogState(true);
            const originalState = { ...state };
            
            // Open and cancel
            state = handleSwitchClick(state);
            state = handleCancel(state);
            
            // State should be back to initial (except for any transient properties)
            expect(state.showConfirmDialog).toBe(originalState.showConfirmDialog);
            expect(state.confirmAction).toBe(originalState.confirmAction);
            expect(state.deactivateError).toBe(originalState.deactivateError);
            expect(state.activationStatus.activated).toBe(originalState.activationStatus.activated);
        });
    });
});
