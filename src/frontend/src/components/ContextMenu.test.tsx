import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ContextMenu from './ContextMenu';

describe('ContextMenu', () => {
  it('renders at the specified position', () => {
    render(
      <ContextMenu 
        position={{ x: 100, y: 200 }} 
        onClose={() => {}} 
        target={document.createElement('input')} 
      />
    );
    const menu = screen.getByRole('menu');
    expect(menu).toHaveStyle('top: 200px');
    expect(menu).toHaveStyle('left: 100px');
  });

  it('shows Copy, Paste, Cut, and Select All options', () => {
    render(
      <ContextMenu 
        position={{ x: 0, y: 0 }} 
        onClose={() => {}} 
        target={document.createElement('input')} 
      />
    );
    expect(screen.getByText('Copy')).toBeInTheDocument();
    expect(screen.getByText('Paste')).toBeInTheDocument();
    expect(screen.getByText('Cut')).toBeInTheDocument();
    expect(screen.getByText('Select All')).toBeInTheDocument();
  });

  it('calls onClose when clicking outside', () => {
    const handleClose = vi.fn();
    render(
        <>
            <div data-testid="outside">Outside</div>
            <ContextMenu 
                position={{ x: 0, y: 0 }} 
                onClose={handleClose} 
                target={document.createElement('input')} 
            />
        </>
    );
    fireEvent.mouseDown(screen.getByTestId('outside'));
    expect(handleClose).toHaveBeenCalled();
  });
});
