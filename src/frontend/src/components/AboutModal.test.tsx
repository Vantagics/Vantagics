import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import AboutModal from './AboutModal';

// Mock the Wails functions
vi.mock('../../wailsjs/go/main/App', () => ({
    GetActivationStatus: vi.fn(),
    DeactivateLicense: vi.fn(),
}));

vi.mock('../../wailsjs/runtime/runtime', () => ({
    BrowserOpenURL: vi.fn(),
    EventsEmit: vi.fn(),
    EventsOn: vi.fn(() => vi.fn()),
}));

vi.mock('../utils/systemLog', () => ({
    createLogger: () => ({
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    }),
}));

// Mock i18n to return the key as-is for easy assertion
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        language: 'English',
        t: (key: string) => key,
    }),
}));

import { GetActivationStatus } from '../../wailsjs/go/main/App';

describe('AboutModal - Credits vs Daily Limit conditional rendering', () => {
    const defaultProps = {
        isOpen: true,
        onClose: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    // Requirement 6.1: Credits mode shows credits usage area with used/total values
    describe('Credits Mode (credits_mode === true)', () => {
        it('should display credits usage area with used/total values', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 0,
                daily_analysis_count: 0,
                total_credits: 100,
                used_credits: 25,
                credits_mode: true,
            });

            render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                expect(screen.getByText('credits_usage')).toBeInTheDocument();
                expect(screen.getByText('25 / 100')).toBeInTheDocument();
            });
        });

        // Requirement 6.2: Credits mode shows progress bar
        it('should display credits usage progress bar', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 0,
                daily_analysis_count: 0,
                total_credits: 100,
                used_credits: 50,
                credits_mode: true,
            });

            const { container } = render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                // The progress bar is a div with bg-blue-200 class containing a child div
                const progressBarTrack = container.querySelector('.bg-blue-200');
                expect(progressBarTrack).toBeInTheDocument();
                const progressBarFill = progressBarTrack?.querySelector('.bg-blue-500');
                expect(progressBarFill).toBeInTheDocument();
                expect(progressBarFill).toHaveStyle({ width: '50%' });
            });
        });

        // Requirement 6.3: Credits mode hides daily analysis count display
        it('should hide daily analysis section when in credits mode', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 10,
                daily_analysis_count: 3,
                total_credits: 100,
                used_credits: 25,
                credits_mode: true,
            });

            render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                expect(screen.getByText('credits_usage')).toBeInTheDocument();
            });

            // Daily analysis section should NOT be rendered
            expect(screen.queryByText('daily_analysis_usage')).not.toBeInTheDocument();
        });
    });

    // Requirement 6.4: Daily limit mode hides credits display and shows daily analysis count
    describe('Daily Limit Mode (credits_mode === false)', () => {
        it('should hide credits section and show daily analysis count', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 10,
                daily_analysis_count: 3,
                total_credits: 0,
                used_credits: 0,
                credits_mode: false,
            });

            render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                expect(screen.getByText('daily_analysis_usage')).toBeInTheDocument();
                expect(screen.getByText('3 / 10')).toBeInTheDocument();
            });

            // Credits section should NOT be rendered
            expect(screen.queryByText('credits_usage')).not.toBeInTheDocument();
        });

        it('should show daily analysis when credits_mode is not set', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 5,
                daily_analysis_count: 2,
                total_credits: 0,
                used_credits: 0,
                // credits_mode not set at all
            });

            render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                expect(screen.getByText('daily_analysis_usage')).toBeInTheDocument();
                expect(screen.getByText('2 / 5')).toBeInTheDocument();
            });

            expect(screen.queryByText('credits_usage')).not.toBeInTheDocument();
        });
    });

    // Requirement 6.5: When credits fully used, progress bar shows warning color (red)
    describe('Credits Fully Used - Warning Color', () => {
        it('should show progress bar in red when credits are fully used', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 0,
                daily_analysis_count: 0,
                total_credits: 100,
                used_credits: 100,
                credits_mode: true,
            });

            const { container } = render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                const progressBarTrack = container.querySelector('.bg-blue-200');
                expect(progressBarTrack).toBeInTheDocument();
                // When fully used, the fill should be red (bg-red-500)
                const redFill = progressBarTrack?.querySelector('.bg-red-500');
                expect(redFill).toBeInTheDocument();
            });
        });

        it('should show progress bar in red when used_credits exceeds total_credits', async () => {
            (GetActivationStatus as any).mockResolvedValue({
                activated: true,
                sn: 'TEST-SN',
                expires_at: '2027-12-31',
                daily_analysis_limit: 0,
                daily_analysis_count: 0,
                total_credits: 50,
                used_credits: 55,
                credits_mode: true,
            });

            const { container } = render(<AboutModal {...defaultProps} />);

            await waitFor(() => {
                const progressBarTrack = container.querySelector('.bg-blue-200');
                expect(progressBarTrack).toBeInTheDocument();
                const redFill = progressBarTrack?.querySelector('.bg-red-500');
                expect(redFill).toBeInTheDocument();
            });
        });
    });
});
