import React from 'react';
import { render, screen, fireEvent, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';

/**
 * Tests for Data Browser Keyboard Shortcuts
 * Requirements: 11.4, 11.5
 *
 * Tests the keyboard shortcut behavior for the data browser:
 * - Ctrl+B (Cmd+B) toggles data browser visibility
 * - Escape closes data browser when open
 * - When toggling ON, uses selectedDataSourceId if available
 */

// Helper component that simulates the App-level keyboard handler for data browser shortcuts
function TestDataBrowserShortcuts({
    initialOpen = false,
    selectedDataSourceId = null as string | null,
    initialSourceId = null as string | null,
}: {
    initialOpen?: boolean;
    selectedDataSourceId?: string | null;
    initialSourceId?: string | null;
}) {
    const [dataBrowserOpen, setDataBrowserOpen] = React.useState(initialOpen);
    const [dataBrowserSourceId, setDataBrowserSourceId] = React.useState<string | null>(initialSourceId);

    React.useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            const isModifier = e.ctrlKey || e.metaKey;

            // Escape key: close data browser if open (no modifier needed)
            if (e.key === 'Escape') {
                setDataBrowserOpen((prev) => {
                    if (prev) {
                        e.preventDefault();
                        return false;
                    }
                    return prev;
                });
                return;
            }

            if (!isModifier) return;

            // Ctrl+B / Cmd+B: Toggle data browser
            if (e.key === 'b' || e.key === 'B') {
                e.preventDefault();
                e.stopPropagation();
                setDataBrowserOpen((prev) => {
                    if (!prev) {
                        // Opening: use selectedDataSourceId if available
                        setDataBrowserSourceId((currentSourceId) => {
                            return selectedDataSourceId || currentSourceId;
                        });
                    }
                    return !prev;
                });
                return;
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => {
            window.removeEventListener('keydown', handleKeyDown);
        };
    }, [selectedDataSourceId]);

    return (
        <div>
            <div data-testid="data-browser-status">
                {dataBrowserOpen ? 'open' : 'closed'}
            </div>
            <div data-testid="data-browser-source-id">
                {dataBrowserSourceId || 'none'}
            </div>
            {dataBrowserOpen && (
                <div data-testid="data-browser-panel" role="dialog" aria-label="Data Browser">
                    <span>Data Browser Content</span>
                    <button data-testid="data-browser-close-button">Close</button>
                </div>
            )}
        </div>
    );
}

describe('Data Browser Keyboard Shortcuts', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('Ctrl+B toggles data browser (Requirement 11.4)', () => {
        it('should open data browser when Ctrl+B is pressed and browser is closed', () => {
            render(<TestDataBrowserShortcuts />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');
        });

        it('should close data browser when Ctrl+B is pressed and browser is open', () => {
            render(<TestDataBrowserShortcuts initialOpen={true} />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');
        });

        it('should open data browser when Cmd+B is pressed (Mac)', () => {
            render(<TestDataBrowserShortcuts />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');

            fireEvent.keyDown(window, { key: 'b', metaKey: true });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');
        });

        it('should close data browser when Cmd+B is pressed (Mac) and browser is open', () => {
            render(<TestDataBrowserShortcuts initialOpen={true} />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');

            fireEvent.keyDown(window, { key: 'b', metaKey: true });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');
        });

        it('should handle uppercase B key with Ctrl', () => {
            render(<TestDataBrowserShortcuts />);

            fireEvent.keyDown(window, { key: 'B', ctrlKey: true });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');
        });

        it('should toggle back and forth with repeated Ctrl+B presses', () => {
            render(<TestDataBrowserShortcuts />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });
            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });
            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });
            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');
        });

        it('should not toggle data browser when B is pressed without modifier', () => {
            render(<TestDataBrowserShortcuts />);

            fireEvent.keyDown(window, { key: 'b' });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');
        });

        it('should prevent default browser behavior for Ctrl+B', () => {
            render(<TestDataBrowserShortcuts />);

            const event = new KeyboardEvent('keydown', {
                key: 'b',
                ctrlKey: true,
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            window.dispatchEvent(event);

            expect(preventDefaultSpy).toHaveBeenCalled();
        });
    });

    describe('Ctrl+B uses selectedDataSourceId when opening (Requirement 11.4)', () => {
        it('should set dataBrowserSourceId to selectedDataSourceId when opening', () => {
            render(
                <TestDataBrowserShortcuts
                    selectedDataSourceId="ds-123"
                />
            );

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });

            expect(screen.getByTestId('data-browser-source-id')).toHaveTextContent('ds-123');
        });

        it('should keep existing source ID when no selectedDataSourceId and opening', () => {
            render(
                <TestDataBrowserShortcuts
                    initialSourceId="ds-existing"
                    selectedDataSourceId={null}
                />
            );

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });

            expect(screen.getByTestId('data-browser-source-id')).toHaveTextContent('ds-existing');
        });

        it('should not change source ID when closing via Ctrl+B', () => {
            render(
                <TestDataBrowserShortcuts
                    initialOpen={true}
                    initialSourceId="ds-current"
                    selectedDataSourceId="ds-new"
                />
            );

            expect(screen.getByTestId('data-browser-source-id')).toHaveTextContent('ds-current');

            fireEvent.keyDown(window, { key: 'b', ctrlKey: true });

            // Source ID should remain unchanged when closing
            expect(screen.getByTestId('data-browser-source-id')).toHaveTextContent('ds-current');
        });
    });

    describe('Escape closes data browser (Requirement 11.5)', () => {
        it('should close data browser when Escape is pressed and browser is open', () => {
            render(<TestDataBrowserShortcuts initialOpen={true} />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('open');

            fireEvent.keyDown(window, { key: 'Escape' });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');
        });

        it('should not change state when Escape is pressed and browser is already closed', () => {
            render(<TestDataBrowserShortcuts />);

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');

            fireEvent.keyDown(window, { key: 'Escape' });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');
        });

        it('should prevent default when Escape closes the data browser', () => {
            render(<TestDataBrowserShortcuts initialOpen={true} />);

            const event = new KeyboardEvent('keydown', {
                key: 'Escape',
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            window.dispatchEvent(event);

            expect(preventDefaultSpy).toHaveBeenCalled();
        });

        it('should not prevent default when Escape is pressed and browser is closed', () => {
            render(<TestDataBrowserShortcuts />);

            const event = new KeyboardEvent('keydown', {
                key: 'Escape',
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            window.dispatchEvent(event);

            expect(preventDefaultSpy).not.toHaveBeenCalled();
        });

        it('should not require modifier keys for Escape', () => {
            render(<TestDataBrowserShortcuts initialOpen={true} />);

            // Escape without any modifier should work
            fireEvent.keyDown(window, { key: 'Escape' });

            expect(screen.getByTestId('data-browser-status')).toHaveTextContent('closed');
        });
    });

    describe('Cleanup', () => {
        it('should remove event listener on unmount', () => {
            const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener');
            const { unmount } = render(<TestDataBrowserShortcuts />);

            unmount();

            expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function));
            removeEventListenerSpy.mockRestore();
        });
    });
});
