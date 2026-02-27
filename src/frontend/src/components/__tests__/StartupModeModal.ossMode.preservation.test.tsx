/**
 * Open Source Mode Preservation Test
 * 
 * Purpose: Verify that the open source mode activation flow remains unchanged
 * after the commercial mode bugfix (email-only activation).
 * 
 * Validates: Requirement 3.2 from bugfix.md
 * 
 * This test ensures:
 * 1. Open source mode UI displays only email input and register button
 * 2. Open source mode calls RequestOpenSourceSN (not RequestSN or RequestFreeSN)
 * 3. Open source mode does NOT call TestLLMConnection or GetEffectiveConfig
 * 4. Open source mode calls onOpenSettings() after activation (not onComplete())
 * 5. Configuration is saved correctly
 * 6. activation-status-changed event is emitted
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import StartupModeModal from '../StartupModeModal';
import * as WailsRuntime from '../../../wailsjs/runtime/runtime';
import * as App from '../../../wailsjs/go/main/App';

// Mock Wails runtime
vi.mock('../../../wailsjs/runtime/runtime', () => ({
    EventsEmit: vi.fn(),
    BrowserOpenURL: vi.fn(),
}));

// Mock Wails Go bindings
vi.mock('../../../wailsjs/go/main/App', () => ({
    GetConfig: vi.fn(),
    SaveConfig: vi.fn(),
    GetActivationStatus: vi.fn(),
    ActivateLicense: vi.fn(),
    RequestOpenSourceSN: vi.fn(),
    RequestSN: vi.fn(),
    RequestFreeSN: vi.fn(),
    TestLLMConnection: vi.fn(),
    GetEffectiveConfig: vi.fn(),
    LoadSavedActivation: vi.fn(),
}));

// Mock i18n
vi.mock('../../i18n', () => ({
    useLanguage: () => ({
        t: (key: string) => {
            const translations: Record<string, string> = {
                'welcome_to_vantagics': 'Welcome to Vantagics',
                'select_usage_mode': 'Select Usage Mode',
                'opensource_mode': 'Open Source Mode',
                'oss_registration_subtitle': 'Register with email',
                'oss_registration_email_hint': 'Enter your email to register',
                'email_address': 'Email Address',
                'oss_register_and_activate': 'Register and Activate',
                'registering': 'Registering...',
                'oss_mode_limitation_note': 'Open source mode requires LLM configuration',
                'please_enter_valid_email': 'Please enter a valid email address',
                'activation_success': 'Activation successful',
            };
            return translations[key] || key;
        },
        language: 'en',
    }),
}));

describe('StartupModeModal - Open Source Mode Preservation', () => {
    const mockOnComplete = vi.fn();
    const mockOnOpenSettings = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        
        // Default mock implementations
        vi.mocked(App.GetActivationStatus).mockResolvedValue({ activated: false });
        vi.mocked(App.GetConfig).mockResolvedValue({});
        vi.mocked(App.LoadSavedActivation).mockResolvedValue({ success: false });
    });

    describe('UI Preservation', () => {
        it('should display only email input and register button (no SN input, no request trial link)', async () => {
            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            // Should display email input
            expect(screen.getByLabelText('Email Address')).toBeInTheDocument();
            
            // Should display register button
            expect(screen.getByRole('button', { name: /Register and Activate/i })).toBeInTheDocument();

            // Should NOT display serial number input
            expect(screen.queryByLabelText(/Serial Number/i)).not.toBeInTheDocument();
            expect(screen.queryByLabelText(/序列号/i)).not.toBeInTheDocument();
            
            // Should NOT display "request trial" link
            expect(screen.queryByText(/Request Trial/i)).not.toBeInTheDocument();
            expect(screen.queryByText(/申请试用/i)).not.toBeInTheDocument();
            expect(screen.queryByText(/没有序列号/i)).not.toBeInTheDocument();
        });

        it('should enable register button only when email is provided', async () => {
            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });
            const emailInput = screen.getByLabelText('Email Address');

            // Initially disabled (no email)
            expect(registerButton).toBeDisabled();

            // Enter email
            fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

            // Should be enabled
            expect(registerButton).not.toBeDisabled();

            // Clear email
            fireEvent.change(emailInput, { target: { value: '' } });

            // Should be disabled again
            expect(registerButton).toBeDisabled();
        });
    });

    describe('Activation Flow Preservation', () => {
        it('should call RequestOpenSourceSN (not RequestSN or RequestFreeSN) and activate successfully', async () => {
            const testEmail = 'test@example.com';
            const testSN = 'OSS-TEST-SN-12345';

            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: true,
                sn: testSN,
            });
            vi.mocked(App.ActivateLicense).mockResolvedValue({
                success: true,
            });
            vi.mocked(App.SaveConfig).mockResolvedValue(undefined);

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            // Enter email and click register
            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                // Should call RequestOpenSourceSN with correct parameters
                expect(App.RequestOpenSourceSN).toHaveBeenCalledWith(
                    'https://license.vantagics.com',
                    testEmail
                );
            });

            // Should NOT call RequestSN (commercial mode function)
            expect(App.RequestSN).not.toHaveBeenCalled();

            // Should NOT call RequestFreeSN (free mode function)
            expect(App.RequestFreeSN).not.toHaveBeenCalled();

            // Should call ActivateLicense with the returned SN
            expect(App.ActivateLicense).toHaveBeenCalledWith(
                'https://license.vantagics.com',
                testSN
            );

            // Should save config
            expect(App.SaveConfig).toHaveBeenCalledWith(
                expect.objectContaining({
                    licenseSN: testSN,
                    licenseServerURL: 'https://license.vantagics.com',
                    licenseEmail: testEmail,
                })
            );

            // Should emit activation-status-changed event
            expect(WailsRuntime.EventsEmit).toHaveBeenCalledWith('activation-status-changed');

            // Should call onOpenSettings (not onComplete)
            expect(mockOnOpenSettings).toHaveBeenCalled();
            expect(mockOnComplete).not.toHaveBeenCalled();
        });

        it('should NOT call TestLLMConnection or GetEffectiveConfig (unlike commercial mode)', async () => {
            const testEmail = 'test@example.com';
            const testSN = 'OSS-TEST-SN-12345';

            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: true,
                sn: testSN,
            });
            vi.mocked(App.ActivateLicense).mockResolvedValue({
                success: true,
            });
            vi.mocked(App.SaveConfig).mockResolvedValue(undefined);

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                expect(mockOnOpenSettings).toHaveBeenCalled();
            });

            // Should NOT call TestLLMConnection (commercial mode only)
            expect(App.TestLLMConnection).not.toHaveBeenCalled();

            // Should NOT call GetEffectiveConfig (commercial mode only)
            expect(App.GetEffectiveConfig).not.toHaveBeenCalled();
        });

        it('should call onOpenSettings() after activation (not onComplete())', async () => {
            const testEmail = 'test@example.com';
            const testSN = 'OSS-TEST-SN-12345';

            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: true,
                sn: testSN,
            });
            vi.mocked(App.ActivateLicense).mockResolvedValue({
                success: true,
            });
            vi.mocked(App.SaveConfig).mockResolvedValue(undefined);

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                // Should call onOpenSettings (to open settings for LLM configuration)
                expect(mockOnOpenSettings).toHaveBeenCalled();
            });

            // Should NOT call onComplete (that's for commercial/free modes)
            expect(mockOnComplete).not.toHaveBeenCalled();
        });
    });

    describe('Error Handling Preservation', () => {
        it('should display error message when email is invalid', async () => {
            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            // Enter invalid email (no @)
            fireEvent.change(emailInput, { target: { value: 'invalid-email' } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                expect(screen.getByText('Please enter a valid email address')).toBeInTheDocument();
            });

            // Should not call any backend functions
            expect(App.RequestOpenSourceSN).not.toHaveBeenCalled();
            expect(App.ActivateLicense).not.toHaveBeenCalled();
        });

        it('should display error message when RequestOpenSourceSN fails', async () => {
            const testEmail = 'test@example.com';
            const errorMessage = 'Server error';

            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: false,
                message: errorMessage,
            });

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                expect(screen.getByText(errorMessage)).toBeInTheDocument();
            });

            // Should not proceed to activation
            expect(App.ActivateLicense).not.toHaveBeenCalled();
            expect(mockOnOpenSettings).not.toHaveBeenCalled();
        });

        it('should display error message when ActivateLicense fails', async () => {
            const testEmail = 'test@example.com';
            const testSN = 'OSS-TEST-SN-12345';
            const errorMessage = 'Activation failed';

            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: true,
                sn: testSN,
            });
            vi.mocked(App.ActivateLicense).mockResolvedValue({
                success: false,
                message: errorMessage,
            });

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                expect(screen.getByText(errorMessage)).toBeInTheDocument();
            });

            // Should not save config or complete
            expect(App.SaveConfig).not.toHaveBeenCalled();
            expect(mockOnOpenSettings).not.toHaveBeenCalled();
        });
    });

    describe('Configuration Preservation', () => {
        it('should save correct configuration after successful activation', async () => {
            const testEmail = 'test@example.com';
            const testSN = 'OSS-TEST-SN-12345';
            const mockConfig = { existingKey: 'existingValue' };

            vi.mocked(App.GetConfig).mockResolvedValue(mockConfig);
            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: true,
                sn: testSN,
            });
            vi.mocked(App.ActivateLicense).mockResolvedValue({
                success: true,
            });
            vi.mocked(App.SaveConfig).mockResolvedValue(undefined);

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                expect(App.SaveConfig).toHaveBeenCalledWith({
                    existingKey: 'existingValue',
                    licenseSN: testSN,
                    licenseServerURL: 'https://license.vantagics.com',
                    licenseEmail: testEmail,
                });
            });
        });
    });

    describe('Event Emission Preservation', () => {
        it('should emit activation-status-changed event after successful activation', async () => {
            const testEmail = 'test@example.com';
            const testSN = 'OSS-TEST-SN-12345';

            vi.mocked(App.RequestOpenSourceSN).mockResolvedValue({
                success: true,
                sn: testSN,
            });
            vi.mocked(App.ActivateLicense).mockResolvedValue({
                success: true,
            });
            vi.mocked(App.SaveConfig).mockResolvedValue(undefined);

            render(
                <StartupModeModal
                    isOpen={true}
                    onComplete={mockOnComplete}
                    onOpenSettings={mockOnOpenSettings}
                    initialMode="opensource"
                />
            );

            await waitFor(() => {
                expect(screen.getByText('Open Source Mode')).toBeInTheDocument();
            });

            const emailInput = screen.getByLabelText('Email Address');
            const registerButton = screen.getByRole('button', { name: /Register and Activate/i });

            fireEvent.change(emailInput, { target: { value: testEmail } });
            fireEvent.click(registerButton);

            await waitFor(() => {
                expect(WailsRuntime.EventsEmit).toHaveBeenCalledWith('activation-status-changed');
            });
        });
    });
});
