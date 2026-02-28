/**
 * Free Mode Activation Preservation Test
 *
 * **Validates: Requirement 3.1 from bugfix.md**
 *
 * This test verifies that the free mode activation flow remains unchanged
 * after the commercial mode bugfix. The free mode should continue to work
 * exactly as before: user enters email → system calls RequestFreeSN → auto-activates.
 *
 * Test Coverage:
 * 1. Free mode UI shows email input and register button
 * 2. Register button is enabled when valid email is entered
 * 3. Complete activation flow: email → RequestFreeSN → ActivateLicense → SaveConfig → Complete
 * 4. Error handling for invalid email and server errors
 * 5. No changes to the free mode behavior
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import StartupModeModal from '../StartupModeModal';
import * as WailsRuntime from '../../../wailsjs/runtime/runtime';
import * as AppBindings from '../../../wailsjs/go/main/App';

// Mock Wails runtime and bindings
vi.mock('../../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
  EventsEmit: vi.fn(),
}));

vi.mock('../../../wailsjs/go/main/App', () => ({
  GetConfig: vi.fn(),
  GetEffectiveConfig: vi.fn(),
  SaveConfig: vi.fn(),
  GetActivationStatus: vi.fn(),
  ActivateLicense: vi.fn(),
  TestLLMConnection: vi.fn(),
  RequestSN: vi.fn(),
  RequestFreeSN: vi.fn(),
  RequestOpenSourceSN: vi.fn(),
  LoadSavedActivation: vi.fn(),
}));

// Mock i18n
vi.mock('../../i18n', () => ({
  useLanguage: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'welcome_to_vantagics': 'Welcome to Vantagics',
        'free_registration': 'Free Registration',
        'free_registration_subtitle': 'Register with email for free',
        'free_registration_email_hint': 'Enter your email to register',
        'email_address': 'Email Address',
        'registering': 'Registering...',
        'please_enter_valid_email': 'Please enter a valid email',
        'license_error_rate_limit': 'Rate limit exceeded',
        'permanent_free': 'Permanent Free',
        'free_mode_limitation_note': 'Free mode has usage limitations',
      };
      return translations[key] || key;
    },
  }),
}));

// Mock logger
vi.mock('../../utils/systemLog', () => ({
  createLogger: () => ({
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
  }),
}));

describe('StartupModeModal - Free Mode Preservation Test', () => {
  const mockOnComplete = vi.fn();
  const mockOnOpenSettings = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetAllMocks();
    
    // Default mock implementations
    (AppBindings.GetActivationStatus as any).mockResolvedValue({ activated: false });
    (AppBindings.GetConfig as any).mockResolvedValue({});
    (AppBindings.LoadSavedActivation as any).mockResolvedValue({ success: false });
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.resetAllMocks();
  });

  describe('Free Mode UI Verification (Requirement 3.1)', () => {
    it('should show email input and register button in free mode', async () => {
      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      // Wait for component to render - use heading role to avoid ambiguity
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      // Should show email input
      const emailInput = screen.getByPlaceholderText('your@email.com');
      expect(emailInput).toBeDefined();
      expect(emailInput.getAttribute('type')).toBe('email');

      // Should show register button
      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      expect(registerButton).toBeDefined();

      // Should show hint text
      expect(screen.getByText('Enter your email to register')).toBeDefined();
    });

    it('should disable register button when email is empty', async () => {
      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      expect(registerButton.hasAttribute('disabled')).toBe(true);
    });

    it('should enable register button when valid email is entered', async () => {
      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      const registerButton = screen.getByRole('button', { name: /Free Registration/i });

      // Initially disabled
      expect(registerButton.hasAttribute('disabled')).toBe(true);

      // Enter valid email
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } });

      // Should be enabled
      expect(registerButton.hasAttribute('disabled')).toBe(false);
    });
  });

  describe('Free Mode Complete Activation Flow (Requirement 3.1)', () => {
    it('should complete full free mode activation: RequestFreeSN → ActivateLicense → SaveConfig → Complete', async () => {
      const testEmail = 'freeuser@example.com';
      const testSN = 'FREE-SN-12345';
      const serverURL = 'https://license.vantagics.com';

      // Mock successful RequestFreeSN
      (AppBindings.RequestFreeSN as any).mockResolvedValue({
        success: true,
        sn: testSN,
      });

      // Mock successful ActivateLicense
      (AppBindings.ActivateLicense as any).mockResolvedValue({
        success: true,
      });

      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      // Enter email
      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: testEmail } });

      // Click register
      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      // Wait for activation flow to complete
      await waitFor(() => {
        expect(mockOnComplete).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Verify RequestFreeSN was called with email
      expect(AppBindings.RequestFreeSN).toHaveBeenCalledWith(serverURL, testEmail);

      // Verify ActivateLicense was called with the SN from RequestFreeSN
      expect(AppBindings.ActivateLicense).toHaveBeenCalledWith(serverURL, testSN);

      // Verify config was saved
      expect(AppBindings.SaveConfig).toHaveBeenCalledWith(
        expect.objectContaining({
          licenseSN: testSN,
          licenseServerURL: serverURL,
          licenseEmail: testEmail,
        })
      );

      // Verify activation-status-changed event was emitted
      expect(WailsRuntime.EventsEmit).toHaveBeenCalledWith('activation-status-changed');

      // Verify onComplete was called (free mode doesn't verify LLM)
      expect(mockOnComplete).toHaveBeenCalled();
    });

    it('should show registering state during activation', async () => {
      const testEmail = 'freeuser@example.com';

      // Mock slow RequestFreeSN to observe loading state
      (AppBindings.RequestFreeSN as any).mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve({ success: true, sn: 'FREE-SN' }), 100))
      );
      (AppBindings.ActivateLicense as any).mockResolvedValue({ success: true });

      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: testEmail } });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      // Should show registering state
      await waitFor(() => {
        expect(screen.getByText('Registering...')).toBeDefined();
      });
    });
  });

  describe('Free Mode Error Handling (Requirement 3.1)', () => {
    it('should show error for invalid email format', async () => {
      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: 'invalid-email' } });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      await waitFor(() => {
        expect(screen.getByText('Please enter a valid email')).toBeDefined();
      });

      // Should not call RequestFreeSN
      expect(AppBindings.RequestFreeSN).not.toHaveBeenCalled();
    });

    it('should handle RequestFreeSN failure', async () => {
      const testEmail = 'test@example.com';

      // Mock RequestFreeSN failure
      (AppBindings.RequestFreeSN as any).mockResolvedValue({
        success: false,
        code: 'rate_limit',
        message: 'Rate limit exceeded',
      });

      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: testEmail } });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      // Should show error
      await waitFor(() => {
        expect(screen.getByText('Rate limit exceeded')).toBeDefined();
      });

      // Should not call ActivateLicense
      expect(AppBindings.ActivateLicense).not.toHaveBeenCalled();
      expect(mockOnComplete).not.toHaveBeenCalled();
    });

    it('should handle ActivateLicense failure in free mode', async () => {
      const testEmail = 'test@example.com';
      const testSN = 'FREE-SN-12345';

      // Mock successful RequestFreeSN but failed ActivateLicense
      (AppBindings.RequestFreeSN as any).mockResolvedValue({
        success: true,
        sn: testSN,
      });
      (AppBindings.ActivateLicense as any).mockResolvedValue({
        success: false,
        message: 'License activation failed',
      });

      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: testEmail } });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      // Should show error
      await waitFor(() => {
        expect(screen.getByText('License activation failed')).toBeDefined();
      });

      // Should not call SaveConfig or complete
      expect(AppBindings.SaveConfig).not.toHaveBeenCalled();
      expect(mockOnComplete).not.toHaveBeenCalled();
    });
  });

  describe('Free Mode Behavior Consistency', () => {
    it('should not call RequestSN (commercial mode function) in free mode', async () => {
      const testEmail = 'freeuser@example.com';

      (AppBindings.RequestFreeSN as any).mockResolvedValue({
        success: true,
        sn: 'FREE-SN-12345',
      });
      (AppBindings.ActivateLicense as any).mockResolvedValue({ success: true });

      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: testEmail } });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      await waitFor(() => {
        expect(mockOnComplete).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Verify RequestSN (commercial) was NOT called
      expect(AppBindings.RequestSN).not.toHaveBeenCalled();

      // Verify RequestFreeSN (free mode) WAS called
      expect(AppBindings.RequestFreeSN).toHaveBeenCalled();
    });

    it('should not verify LLM connection in free mode (unlike commercial mode)', async () => {
      const testEmail = 'freeuser@example.com';

      (AppBindings.RequestFreeSN as any).mockResolvedValue({
        success: true,
        sn: 'FREE-SN-12345',
      });
      (AppBindings.ActivateLicense as any).mockResolvedValue({ success: true });

      render(
        <StartupModeModal
          isOpen={true}
          onComplete={mockOnComplete}
          onOpenSettings={mockOnOpenSettings}
          initialMode="free"
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Free Registration' })).toBeDefined();
      });

      const emailInput = screen.getByPlaceholderText('your@email.com');
      fireEvent.change(emailInput, { target: { value: testEmail } });

      const registerButton = screen.getByRole('button', { name: /Free Registration/i });
      fireEvent.click(registerButton);

      await waitFor(() => {
        expect(mockOnComplete).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Verify TestLLMConnection was NOT called (free mode doesn't verify LLM)
      expect(AppBindings.TestLLMConnection).not.toHaveBeenCalled();

      // Verify GetEffectiveConfig was NOT called (only used for LLM verification)
      expect(AppBindings.GetEffectiveConfig).not.toHaveBeenCalled();
    });
  });
});
