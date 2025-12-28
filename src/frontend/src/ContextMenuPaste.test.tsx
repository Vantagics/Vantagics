import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ContextMenu from './components/ContextMenu';
import { vi } from 'vitest';
import * as WailsRuntime from '../wailsjs/runtime/runtime';

vi.mock('../wailsjs/runtime/runtime', () => ({
    ClipboardGetText: vi.fn(),
    ClipboardSetText: vi.fn(),
}));

describe('ContextMenu Paste with Wails', () => {
  it('calls ClipboardGetText and pastes content', async () => {
    const handleClose = vi.fn();
    const input = document.createElement('input');
    input.value = 'Initial';
    document.body.appendChild(input);

    (WailsRuntime.ClipboardGetText as any).mockResolvedValue('Pasted Text');

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
        // Since we modify the input value manually in the component, testing the value is tricky if we don't trigger React events or if input is uncontrolled. 
        // But let's check the input value property.
        // Wait, ContextMenu implementation uses input.value manipulation directly.
        // So checking the element value should work.
        expect(input.value).toContain('Pasted Text');
    });

    document.body.removeChild(input);
  });
});
