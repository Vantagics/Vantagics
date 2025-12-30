import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ContextMenu from './components/ContextMenu';
import { vi } from 'vitest';
import * as WailsRuntime from '../wailsjs/runtime/runtime';

vi.mock('../wailsjs/runtime/runtime', () => ({
    ClipboardGetText: vi.fn(),
    ClipboardSetText: vi.fn(),
}));

describe('ContextMenu Paste Regression', () => {
    let mockClipboardReadText: any;
    let mockExecCommand: any;

    beforeEach(() => {
        mockClipboardReadText = vi.fn();
        Object.assign(navigator, {
            clipboard: {
                readText: mockClipboardReadText,
                writeText: vi.fn(),
            },
        });
        mockExecCommand = vi.fn();
        document.execCommand = mockExecCommand;
        vi.clearAllMocks();
    });

    it('inserts text at cursor position in non-empty input', async () => {
        const handleClose = vi.fn();
        const input = document.createElement('input');
        input.value = 'Hello World';
        document.body.appendChild(input);
        
        // Set cursor before "World"
        input.selectionStart = 6;
        input.selectionEnd = 6;

        (WailsRuntime.ClipboardGetText as any).mockResolvedValue('Beautiful ');

        render(
            <ContextMenu 
                position={{ x: 0, y: 0 }} 
                onClose={handleClose} 
                target={input} 
            />
        );

        const pasteButton = screen.getByText('Paste');
        fireEvent.click(pasteButton);

        await waitFor(() => {
            expect(WailsRuntime.ClipboardGetText).toHaveBeenCalled();
            expect(input.value).toBe('Hello Beautiful World');
        });

        document.body.removeChild(input);
    });

    it('pastes correctly into textarea', async () => {
        const handleClose = vi.fn();
        const textarea = document.createElement('textarea');
        textarea.value = '';
        document.body.appendChild(textarea);

        (WailsRuntime.ClipboardGetText as any).mockResolvedValue('Multi\nLine\nText');

        render(
            <ContextMenu 
                position={{ x: 0, y: 0 }} 
                onClose={handleClose} 
                target={textarea} 
            />
        );

        const pasteButton = screen.getByText('Paste');
        fireEvent.click(pasteButton);

        await waitFor(() => {
            expect(textarea.value).toBe('Multi\nLine\nText');
        });

        document.body.removeChild(textarea);
    });

    it('falls back to navigator.clipboard if Wails returns empty string', async () => {
        const handleClose = vi.fn();
        const input = document.createElement('input');
        input.value = '';
        document.body.appendChild(input);

        // Mock Wails returning empty
        (WailsRuntime.ClipboardGetText as any).mockResolvedValue('');
        // Mock Navigator returning text
        mockClipboardReadText.mockResolvedValue('Browser Clipboard');

        render(
            <ContextMenu 
                position={{ x: 0, y: 0 }} 
                onClose={handleClose} 
                target={input} 
            />
        );

        const pasteButton = screen.getByText('Paste');
        fireEvent.click(pasteButton);

        await waitFor(() => {
            expect(WailsRuntime.ClipboardGetText).toHaveBeenCalled();
            expect(mockClipboardReadText).toHaveBeenCalled();
            expect(input.value).toBe('Browser Clipboard');
        });

        document.body.removeChild(input);
    });
});
