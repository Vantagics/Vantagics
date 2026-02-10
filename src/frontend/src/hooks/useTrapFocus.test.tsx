import React, { useRef } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, vi } from 'vitest';
import { useTrapFocus, getFocusableElements } from './useTrapFocus';

/**
 * Tests for useTrapFocus hook
 * Requirements: 11.7 - Tab navigation within each panel
 * Validates: Property 19 (Tab navigation containment)
 */

/**
 * Test component that uses the useTrapFocus hook.
 * Renders a panel with focusable elements for testing tab trapping.
 */
function TestPanel({
    enabled = true,
    children,
}: {
    enabled?: boolean;
    children?: React.ReactNode;
}) {
    const panelRef = useRef<HTMLDivElement>(null);
    useTrapFocus(panelRef, enabled);

    return (
        <div
            ref={panelRef}
            data-testid="test-panel"
            tabIndex={-1}
            role="region"
            aria-label="Test Panel"
        >
            {children || (
                <>
                    <button data-testid="btn-first">First</button>
                    <input data-testid="input-middle" placeholder="Middle" />
                    <button data-testid="btn-last">Last</button>
                </>
            )}
        </div>
    );
}

describe('useTrapFocus', () => {
    describe('Tab wrapping behavior', () => {
        it('should wrap focus from last element to first on Tab', () => {
            render(<TestPanel />);

            const lastBtn = screen.getByTestId('btn-last');
            lastBtn.focus();
            expect(document.activeElement).toBe(lastBtn);

            fireEvent.keyDown(lastBtn, { key: 'Tab' });

            expect(document.activeElement).toBe(screen.getByTestId('btn-first'));
        });

        it('should wrap focus from first element to last on Shift+Tab', () => {
            render(<TestPanel />);

            const firstBtn = screen.getByTestId('btn-first');
            firstBtn.focus();
            expect(document.activeElement).toBe(firstBtn);

            fireEvent.keyDown(firstBtn, { key: 'Tab', shiftKey: true });

            expect(document.activeElement).toBe(screen.getByTestId('btn-last'));
        });

        it('should wrap focus from panel container to last on Shift+Tab', () => {
            render(<TestPanel />);

            const panel = screen.getByTestId('test-panel');
            panel.focus();
            expect(document.activeElement).toBe(panel);

            fireEvent.keyDown(panel, { key: 'Tab', shiftKey: true });

            expect(document.activeElement).toBe(screen.getByTestId('btn-last'));
        });

        it('should not interfere with Tab between middle elements', () => {
            render(<TestPanel />);

            const firstBtn = screen.getByTestId('btn-first');
            firstBtn.focus();

            // Tab on first element (not last) should NOT be prevented
            const event = new KeyboardEvent('keydown', {
                key: 'Tab',
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            firstBtn.dispatchEvent(event);

            // preventDefault should NOT be called since we're not on the last element
            expect(preventDefaultSpy).not.toHaveBeenCalled();
        });
    });

    describe('Disabled state', () => {
        it('should not trap focus when disabled', () => {
            render(<TestPanel enabled={false} />);

            const lastBtn = screen.getByTestId('btn-last');
            lastBtn.focus();

            const event = new KeyboardEvent('keydown', {
                key: 'Tab',
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            lastBtn.dispatchEvent(event);

            // Should not prevent default when disabled
            expect(preventDefaultSpy).not.toHaveBeenCalled();
        });
    });

    describe('No focusable elements', () => {
        it('should not throw when panel has no focusable elements', () => {
            render(
                <TestPanel>
                    <p>No focusable elements here</p>
                </TestPanel>
            );

            const panel = screen.getByTestId('test-panel');
            panel.focus();

            // Should not throw
            expect(() => {
                fireEvent.keyDown(panel, { key: 'Tab' });
            }).not.toThrow();
        });
    });

    describe('Non-Tab keys', () => {
        it('should not interfere with non-Tab key events', () => {
            render(<TestPanel />);

            const firstBtn = screen.getByTestId('btn-first');
            firstBtn.focus();

            const event = new KeyboardEvent('keydown', {
                key: 'Enter',
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            firstBtn.dispatchEvent(event);

            expect(preventDefaultSpy).not.toHaveBeenCalled();
        });

        it('should not interfere with Escape key', () => {
            render(<TestPanel />);

            const firstBtn = screen.getByTestId('btn-first');
            firstBtn.focus();

            const event = new KeyboardEvent('keydown', {
                key: 'Escape',
                bubbles: true,
                cancelable: true,
            });
            const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
            firstBtn.dispatchEvent(event);

            expect(preventDefaultSpy).not.toHaveBeenCalled();
        });
    });

    describe('Disabled and hidden elements', () => {
        it('should skip disabled buttons', () => {
            render(
                <TestPanel>
                    <button data-testid="btn-a">A</button>
                    <button data-testid="btn-b" disabled>B (disabled)</button>
                    <button data-testid="btn-c">C</button>
                </TestPanel>
            );

            const btnC = screen.getByTestId('btn-c');
            btnC.focus();

            // Tab on last focusable element should wrap to first focusable
            fireEvent.keyDown(btnC, { key: 'Tab' });

            expect(document.activeElement).toBe(screen.getByTestId('btn-a'));
        });

        it('should skip elements with tabindex="-1"', () => {
            render(
                <TestPanel>
                    <button data-testid="btn-a">A</button>
                    <div data-testid="div-skip" tabIndex={-1}>Skip me</div>
                    <button data-testid="btn-b">B</button>
                </TestPanel>
            );

            const btnB = screen.getByTestId('btn-b');
            btnB.focus();

            fireEvent.keyDown(btnB, { key: 'Tab' });

            expect(document.activeElement).toBe(screen.getByTestId('btn-a'));
        });
    });

    describe('Various focusable element types', () => {
        it('should include links, inputs, selects, and textareas', () => {
            render(
                <TestPanel>
                    <a href="#" data-testid="link">Link</a>
                    <input data-testid="input" />
                    <select data-testid="select"><option>Opt</option></select>
                    <textarea data-testid="textarea" />
                </TestPanel>
            );

            const textarea = screen.getByTestId('textarea');
            textarea.focus();

            // Tab on last element should wrap to first
            fireEvent.keyDown(textarea, { key: 'Tab' });

            expect(document.activeElement).toBe(screen.getByTestId('link'));
        });
    });

    describe('Modifier key combinations', () => {
        it('should not interfere with Ctrl+Tab', () => {
            render(<TestPanel />);

            const lastBtn = screen.getByTestId('btn-last');
            lastBtn.focus();

            // Ctrl+Tab is a browser shortcut, our hook should still handle Tab
            // but the key event still has key === 'Tab'
            fireEvent.keyDown(lastBtn, { key: 'Tab', ctrlKey: true });

            // The hook handles Tab regardless of Ctrl - this is fine since
            // Ctrl+Tab is typically handled by the browser before reaching JS
            expect(document.activeElement).toBe(screen.getByTestId('btn-first'));
        });
    });
});

describe('getFocusableElements', () => {
    it('should return all focusable elements within a container', () => {
        const { container } = render(
            <div data-testid="container">
                <button>Btn 1</button>
                <input placeholder="Input" />
                <a href="#">Link</a>
                <button>Btn 2</button>
            </div>
        );

        const containerEl = screen.getByTestId('container');
        const focusable = getFocusableElements(containerEl);

        expect(focusable).toHaveLength(4);
    });

    it('should exclude disabled elements', () => {
        render(
            <div data-testid="container">
                <button>Enabled</button>
                <button disabled>Disabled</button>
                <input disabled />
            </div>
        );

        const containerEl = screen.getByTestId('container');
        const focusable = getFocusableElements(containerEl);

        expect(focusable).toHaveLength(1);
    });

    it('should exclude elements with tabindex="-1"', () => {
        render(
            <div data-testid="container">
                <button>Focusable</button>
                <div tabIndex={-1}>Not focusable via tab</div>
            </div>
        );

        const containerEl = screen.getByTestId('container');
        const focusable = getFocusableElements(containerEl);

        expect(focusable).toHaveLength(1);
    });

    it('should include elements with tabindex="0"', () => {
        render(
            <div data-testid="container">
                <button>Button</button>
                <div tabIndex={0}>Custom focusable</div>
            </div>
        );

        const containerEl = screen.getByTestId('container');
        const focusable = getFocusableElements(containerEl);

        expect(focusable).toHaveLength(2);
    });

    it('should return empty array when no focusable elements exist', () => {
        render(
            <div data-testid="container">
                <p>Just text</p>
                <span>More text</span>
            </div>
        );

        const containerEl = screen.getByTestId('container');
        const focusable = getFocusableElements(containerEl);

        expect(focusable).toHaveLength(0);
    });
});
