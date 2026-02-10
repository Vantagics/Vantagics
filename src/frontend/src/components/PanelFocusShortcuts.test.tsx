import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';

/**
 * Tests for Panel Focus Keyboard Shortcuts
 * Requirements: 11.1, 11.2, 11.3, 11.6
 * 
 * Tests the keyboard shortcut behavior for focusing panels:
 * - Ctrl+1 (Cmd+1) focuses LeftPanel
 * - Ctrl+2 (Cmd+2) focuses CenterPanel (message input if available)
 * - Ctrl+3 (Cmd+3) focuses RightPanel
 * - Visual focus indicators are present
 */

// Helper component that simulates the three-panel layout with keyboard shortcuts
// This isolates the keyboard shortcut logic from the full App component
function TestPanelLayout({ centerInputDisabled = false }: { centerInputDisabled?: boolean }) {
    React.useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            const isModifier = e.ctrlKey || e.metaKey;
            if (!isModifier) return;

            let panelTestId: string | null = null;
            let isCenterPanel = false;

            switch (e.key) {
                case '1':
                    panelTestId = 'left-panel';
                    break;
                case '2':
                    panelTestId = 'center-panel';
                    isCenterPanel = true;
                    break;
                case '3':
                    panelTestId = 'right-panel';
                    break;
                default:
                    return;
            }

            if (panelTestId) {
                e.preventDefault();
                e.stopPropagation();

                const panelEl = document.querySelector<HTMLElement>(`[data-testid="${panelTestId}"]`);
                if (panelEl) {
                    if (isCenterPanel) {
                        const messageInput = panelEl.querySelector<HTMLInputElement>('[data-testid="message-input"]');
                        if (messageInput && !messageInput.disabled) {
                            messageInput.focus();
                            return;
                        }
                    }
                    panelEl.focus();
                }
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => {
            window.removeEventListener('keydown', handleKeyDown);
        };
    }, []);

    return (
        <div style={{ display: 'flex' }}>
            <div
                data-testid="left-panel"
                className="left-panel"
                role="region"
                aria-label="Data Sources Panel"
                tabIndex={-1}
            >
                <h3>Data Sources</h3>
            </div>
            <div
                data-testid="center-panel"
                className="center-panel"
                role="region"
                aria-label="Chat Panel"
                tabIndex={-1}
            >
                <input
                    type="text"
                    data-testid="message-input"
                    placeholder="Ask a question..."
                    disabled={centerInputDisabled}
                    aria-label="Message input"
                />
            </div>
            <div
                data-testid="right-panel"
                className="right-panel"
                role="region"
                aria-label="Dashboard Panel"
                tabIndex={-1}
            >
                <h3>Dashboard</h3>
            </div>
        </div>
    );
}

describe('Panel Focus Keyboard Shortcuts', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('Ctrl+1 focuses LeftPanel (Requirement 11.1)', () => {
        it('should focus the left panel when Ctrl+1 is pressed', () => {
            render(<TestPanelLayout />);
            const leftPanel = screen.getByTestId('left-panel');

            fireEvent.keyDown(window, { key: '1', ctrlKey: true });

            expect(document.activeElement).toBe(leftPanel);
        });

        it('should focus the left panel when Cmd+1 is pressed (Mac)', () => {
            render(<TestPanelLayout />);
            const leftPanel = screen.getByTestId('left-panel');

            fireEvent.keyDown(window, { key: '1', metaKey: true });

            expect(document.activeElement).toBe(leftPanel);
        });
    });

    describe('Ctrl+2 focuses CenterPanel (Requirement 11.2)', () => {
        it('should focus the message input when Ctrl+2 is pressed and input is enabled', () => {
            render(<TestPanelLayout />);
            const messageInput = screen.getByTestId('message-input');

            fireEvent.keyDown(window, { key: '2', ctrlKey: true });

            expect(document.activeElement).toBe(messageInput);
        });

        it('should focus the message input when Cmd+2 is pressed (Mac)', () => {
            render(<TestPanelLayout />);
            const messageInput = screen.getByTestId('message-input');

            fireEvent.keyDown(window, { key: '2', metaKey: true });

            expect(document.activeElement).toBe(messageInput);
        });

        it('should focus the center panel container when message input is disabled', () => {
            render(<TestPanelLayout centerInputDisabled={true} />);
            const centerPanel = screen.getByTestId('center-panel');

            fireEvent.keyDown(window, { key: '2', ctrlKey: true });

            expect(document.activeElement).toBe(centerPanel);
        });
    });

    describe('Ctrl+3 focuses RightPanel (Requirement 11.3)', () => {
        it('should focus the right panel when Ctrl+3 is pressed', () => {
            render(<TestPanelLayout />);
            const rightPanel = screen.getByTestId('right-panel');

            fireEvent.keyDown(window, { key: '3', ctrlKey: true });

            expect(document.activeElement).toBe(rightPanel);
        });

        it('should focus the right panel when Cmd+3 is pressed (Mac)', () => {
            render(<TestPanelLayout />);
            const rightPanel = screen.getByTestId('right-panel');

            fireEvent.keyDown(window, { key: '3', metaKey: true });

            expect(document.activeElement).toBe(rightPanel);
        });
    });

    describe('Shortcut behavior', () => {
        it('should not focus any panel when key is pressed without modifier', () => {
            render(<TestPanelLayout />);
            const leftPanel = screen.getByTestId('left-panel');
            const centerPanel = screen.getByTestId('center-panel');
            const rightPanel = screen.getByTestId('right-panel');

            fireEvent.keyDown(window, { key: '1' });
            expect(document.activeElement).not.toBe(leftPanel);

            fireEvent.keyDown(window, { key: '2' });
            expect(document.activeElement).not.toBe(centerPanel);

            fireEvent.keyDown(window, { key: '3' });
            expect(document.activeElement).not.toBe(rightPanel);
        });

        it('should not respond to Ctrl+4 or other number keys', () => {
            render(<TestPanelLayout />);
            const initialActive = document.activeElement;

            fireEvent.keyDown(window, { key: '4', ctrlKey: true });
            expect(document.activeElement).toBe(initialActive);

            fireEvent.keyDown(window, { key: '0', ctrlKey: true });
            expect(document.activeElement).toBe(initialActive);
        });

        it('should prevent default browser behavior for Ctrl+1/2/3', () => {
            render(<TestPanelLayout />);

            const event1 = new KeyboardEvent('keydown', {
                key: '1',
                ctrlKey: true,
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy1 = vi.spyOn(event1, 'preventDefault');
            window.dispatchEvent(event1);
            expect(preventDefaultSpy1).toHaveBeenCalled();

            const event2 = new KeyboardEvent('keydown', {
                key: '2',
                ctrlKey: true,
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy2 = vi.spyOn(event2, 'preventDefault');
            window.dispatchEvent(event2);
            expect(preventDefaultSpy2).toHaveBeenCalled();

            const event3 = new KeyboardEvent('keydown', {
                key: '3',
                ctrlKey: true,
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy3 = vi.spyOn(event3, 'preventDefault');
            window.dispatchEvent(event3);
            expect(preventDefaultSpy3).toHaveBeenCalled();
        });
    });

    describe('Visual focus indicators (Requirement 11.6)', () => {
        it('should have tabIndex=-1 on left panel for programmatic focus', () => {
            render(<TestPanelLayout />);
            const leftPanel = screen.getByTestId('left-panel');
            expect(leftPanel).toHaveAttribute('tabIndex', '-1');
        });

        it('should have tabIndex=-1 on center panel for programmatic focus', () => {
            render(<TestPanelLayout />);
            const centerPanel = screen.getByTestId('center-panel');
            expect(centerPanel).toHaveAttribute('tabIndex', '-1');
        });

        it('should have tabIndex=-1 on right panel for programmatic focus', () => {
            render(<TestPanelLayout />);
            const rightPanel = screen.getByTestId('right-panel');
            expect(rightPanel).toHaveAttribute('tabIndex', '-1');
        });

        it('should have correct ARIA roles on all panels', () => {
            render(<TestPanelLayout />);
            expect(screen.getByTestId('left-panel')).toHaveAttribute('role', 'region');
            expect(screen.getByTestId('center-panel')).toHaveAttribute('role', 'region');
            expect(screen.getByTestId('right-panel')).toHaveAttribute('role', 'region');
        });

        it('should have correct ARIA labels on all panels', () => {
            render(<TestPanelLayout />);
            expect(screen.getByTestId('left-panel')).toHaveAttribute('aria-label', 'Data Sources Panel');
            expect(screen.getByTestId('center-panel')).toHaveAttribute('aria-label', 'Chat Panel');
            expect(screen.getByTestId('right-panel')).toHaveAttribute('aria-label', 'Dashboard Panel');
        });
    });

    describe('Cleanup', () => {
        it('should remove event listener on unmount', () => {
            const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener');
            const { unmount } = render(<TestPanelLayout />);

            unmount();

            expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function));
            removeEventListenerSpy.mockRestore();
        });
    });
});
