import { useEffect, RefObject } from 'react';

/**
 * Selector for all focusable elements within a container.
 * Includes buttons, inputs, selects, textareas, links with href,
 * and elements with a non-negative tabIndex.
 */
const FOCUSABLE_SELECTOR = [
    'a[href]:not([disabled]):not([tabindex="-1"])',
    'button:not([disabled]):not([tabindex="-1"])',
    'input:not([disabled]):not([tabindex="-1"])',
    'select:not([disabled]):not([tabindex="-1"])',
    'textarea:not([disabled]):not([tabindex="-1"])',
    '[tabindex]:not([tabindex="-1"]):not([disabled])',
].join(', ');

/**
 * Returns all focusable elements within a container, excluding the container itself.
 */
export function getFocusableElements(container: HTMLElement): HTMLElement[] {
    const elements = Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));
    // Filter out the container itself (which may have tabIndex=-1 but matches the selector)
    // and filter out elements that are not visible
    return elements.filter(el => {
        if (el === container) return false;
        // Check if element is visible (not hidden or display:none)
        const style = window.getComputedStyle(el);
        return style.display !== 'none' && style.visibility !== 'hidden';
    });
}

/**
 * Custom hook that traps Tab/Shift+Tab focus within a panel container.
 *
 * When the panel (or any element within it) has focus, pressing Tab will cycle
 * through focusable elements within the panel. After the last element, focus
 * wraps to the first. Shift+Tab at the first element wraps to the last.
 *
 * The trap is only active when focus is within the container.
 *
 * Requirements: 11.7 - Tab navigation within each panel
 * Validates: Property 19 (Tab navigation containment)
 *
 * @param containerRef - React ref to the panel container element
 * @param enabled - Whether the focus trap is active (default: true)
 */
export function useTrapFocus(
    containerRef: RefObject<HTMLElement | null>,
    enabled: boolean = true
): void {
    useEffect(() => {
        if (!enabled) return;

        const container = containerRef.current;
        if (!container) return;

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key !== 'Tab') return;

            const focusableElements = getFocusableElements(container);
            if (focusableElements.length === 0) return;

            const firstElement = focusableElements[0];
            const lastElement = focusableElements[focusableElements.length - 1];

            if (e.shiftKey) {
                // Shift+Tab: if on first element (or the container itself), wrap to last
                if (document.activeElement === firstElement || document.activeElement === container) {
                    e.preventDefault();
                    lastElement.focus();
                }
            } else {
                // Tab: if on last element, wrap to first
                if (document.activeElement === lastElement) {
                    e.preventDefault();
                    firstElement.focus();
                }
            }
        };

        container.addEventListener('keydown', handleKeyDown);
        return () => {
            container.removeEventListener('keydown', handleKeyDown);
        };
    }, [containerRef, enabled]);
}

export default useTrapFocus;
