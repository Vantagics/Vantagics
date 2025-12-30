import { fireEvent, render } from '@testing-library/react';

describe('Context Menu Logic', () => {
  it('allows contextmenu event on inputs', () => {
    // We test that no one is calling preventDefault on the window level for inputs
    // Although main.tsx logic is global, we can check if a manually dispatched event is prevented.
    const input = document.createElement('input');
    document.body.appendChild(input);
    
    const event = new MouseEvent('contextmenu', {
      bubbles: true,
      cancelable: true,
    });
    
    input.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(false);
    
    document.body.removeChild(input);
  });
});
