import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        t: (key: string) => {
            const translations: Record<string, string> = {
                'insufficient_balance': 'Insufficient Balance',
                'current_balance_label': 'Current Balance',
                'balance_needed': 'Total Cost',
                'balance_diff': 'Shortfall',
                'market_browse_credits': 'Credits',
                'cancel': 'Cancel',
                'go_topup': 'Go Top Up',
            };
            return translations[key] || key;
        },
        currentLanguage: 'English',
        setLanguage: vi.fn(),
    }),
}));

import React from 'react';

describe('InsufficientBalanceDialog', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
    });

    afterEach(() => {
        document.body.innerHTML = '';
    });

    async function renderDialog(props: {
        currentBalance: number;
        totalCost: number;
        onTopUp: () => void;
        onClose: () => void;
    }) {
        const { default: InsufficientBalanceDialog } = await import('./InsufficientBalanceDialog');
        const { createRoot } = await import('react-dom/client');
        const root = createRoot(container);

        await new Promise<void>((resolve) => {
            root.render(React.createElement(InsufficientBalanceDialog, props));
            setTimeout(resolve, 0);
        });

        return root;
    }

    it('displays current balance, total cost, and deficit', async () => {
        const onTopUp = vi.fn();
        const onClose = vi.fn();

        await renderDialog({ currentBalance: 50, totalCost: 120, onTopUp, onClose });

        const dialog = document.querySelector('[role="dialog"]');
        expect(dialog).not.toBeNull();

        const textContent = dialog!.textContent || '';
        // Current balance: 50
        expect(textContent).toContain('50');
        // Total cost: 120
        expect(textContent).toContain('120');
        // Deficit: 70
        expect(textContent).toContain('70');
        // Labels
        expect(textContent).toContain('Current Balance');
        expect(textContent).toContain('Total Cost');
        expect(textContent).toContain('Shortfall');
    });

    it('displays the title', async () => {
        await renderDialog({ currentBalance: 10, totalCost: 100, onTopUp: vi.fn(), onClose: vi.fn() });

        const title = document.getElementById('insufficient-balance-dialog-title');
        expect(title).not.toBeNull();
        expect(title!.textContent).toBe('Insufficient Balance');
    });

    it('calls onTopUp when "Go Top Up" button is clicked', async () => {
        const onTopUp = vi.fn();
        const onClose = vi.fn();

        await renderDialog({ currentBalance: 10, totalCost: 100, onTopUp, onClose });

        const buttons = document.querySelectorAll('button');
        const topUpButton = Array.from(buttons).find(b => b.textContent === 'Go Top Up');
        expect(topUpButton).not.toBeUndefined();

        topUpButton!.click();
        expect(onTopUp).toHaveBeenCalledTimes(1);
    });

    it('calls onClose when Cancel button is clicked', async () => {
        const onTopUp = vi.fn();
        const onClose = vi.fn();

        await renderDialog({ currentBalance: 10, totalCost: 100, onTopUp, onClose });

        const buttons = document.querySelectorAll('button');
        const cancelButton = Array.from(buttons).find(b => b.textContent === 'Cancel');
        expect(cancelButton).not.toBeUndefined();

        cancelButton!.click();
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('calculates deficit correctly for various values', async () => {
        await renderDialog({ currentBalance: 0, totalCost: 500, onTopUp: vi.fn(), onClose: vi.fn() });

        const dialog = document.querySelector('[role="dialog"]');
        const textContent = dialog!.textContent || '';
        expect(textContent).toContain('500');
    });

    it('has proper aria attributes for accessibility', async () => {
        await renderDialog({ currentBalance: 10, totalCost: 100, onTopUp: vi.fn(), onClose: vi.fn() });

        const dialog = document.querySelector('[role="dialog"]');
        expect(dialog).not.toBeNull();
        expect(dialog!.getAttribute('aria-modal')).toBe('true');
        expect(dialog!.getAttribute('aria-labelledby')).toBe('insufficient-balance-dialog-title');
    });
});
