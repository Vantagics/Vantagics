import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ChatSidebar from './ChatSidebar';

describe('ChatSidebar', () => {
  it('renders closed by default', () => {
    render(<ChatSidebar />);
    const sidebar = screen.getByTestId('chat-sidebar');
    expect(sidebar).toHaveClass('translate-x-full'); // Assuming hidden off-screen
  });

  it('renders open when isOpen prop is true', () => {
    render(<ChatSidebar isOpen={true} />);
    const sidebar = screen.getByTestId('chat-sidebar');
    expect(sidebar).not.toHaveClass('translate-x-full');
  });

  it('calls onClose when close button is clicked', () => {
    const handleClose = vi.fn();
    render(<ChatSidebar isOpen={true} onClose={handleClose} />);
    const closeButton = screen.getByLabelText('Close sidebar');
    fireEvent.click(closeButton);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });
});
