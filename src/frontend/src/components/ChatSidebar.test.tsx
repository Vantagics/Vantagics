import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import ChatSidebar from './ChatSidebar';

// Mock Wails bindings
vi.mock('../../wailsjs/go/main/App', () => ({
  GetChatHistory: vi.fn(() => Promise.resolve([])),
  SaveChatHistory: vi.fn(() => Promise.resolve()),
  SendMessage: vi.fn(() => Promise.resolve("Mock response")),
  DeleteThread: vi.fn(() => Promise.resolve()),
  ClearHistory: vi.fn(() => Promise.resolve()),
  GetConfig: vi.fn(() => Promise.resolve({ language: 'English' })),
  GetDataSources: vi.fn(() => Promise.resolve([])),
}));

// Mock window.runtime
(window as any).runtime = {
    EventsOnMultiple: vi.fn().mockReturnValue(() => {}),
};

describe('ChatSidebar', () => {
  it('renders closed by default', () => {
    render(<ChatSidebar isOpen={false} onClose={() => {}} />);
    const sidebar = screen.getByTestId('chat-sidebar');
    expect(sidebar).toHaveClass('translate-x-full'); // Assuming hidden off-screen
  });

  it('renders open when isOpen prop is true', () => {
    render(<ChatSidebar isOpen={true} onClose={() => {}} />);
    const sidebar = screen.getByTestId('chat-sidebar');
    expect(sidebar).toHaveClass('translate-x-0');
  });

  it('calls onClose when close button is clicked', () => {
    const handleClose = vi.fn();
    render(<ChatSidebar isOpen={true} onClose={handleClose} />);
    const closeButton = screen.getByLabelText('Close sidebar');
    fireEvent.click(closeButton);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('shows confirmation modal when Clear History is clicked', () => {
    render(<ChatSidebar isOpen={true} onClose={() => {}} />);
    const clearButton = screen.getByText('CLEAR HISTORY');
    fireEvent.click(clearButton);
    expect(screen.getByText('Clear All History?')).toBeInTheDocument();
  });
});