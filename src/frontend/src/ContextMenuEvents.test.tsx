import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react';
import ContextMenu from './components/ContextMenu';

describe('ContextMenu Event Handling', () => {
  it('prevents native context menu on the custom menu itself', () => {
    const handleClose = vi.fn();
    render(
      <ContextMenu 
        position={{ x: 100, y: 100 }} 
        target={document.createElement('input')}
        onClose={handleClose} 
      />
    );

    const menu = screen.getByRole('menu');
    
    // Simulate a right click on the menu
    const contextMenuEvent = new MouseEvent('contextmenu', {
      bubbles: true,
      cancelable: true,
    });
    
    // We need to spy on preventDefault/stopPropagation or check defaultPrevented
    // But since we are testing React component logic, checking defaultPrevented on the event after dispatch is correct if the handler works.
    
    fireEvent(menu, contextMenuEvent);
    
    expect(contextMenuEvent.defaultPrevented).toBe(true);
  });
});
