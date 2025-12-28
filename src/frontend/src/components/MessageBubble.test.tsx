import React from 'react';
import { render, screen } from '@testing-library/react';
import MessageBubble from './MessageBubble';

describe('MessageBubble', () => {
  it('renders user message on the right', () => {
    const { container } = render(<MessageBubble role="user" content="Hello" />);
    // Check if the container (or child) has 'justify-end' class
    expect(container.firstChild).toHaveClass('justify-end');
  });

  it('renders assistant message on the left', () => {
    const { container } = render(<MessageBubble role="assistant" content="Hi there" />);
    expect(container.firstChild).toHaveClass('justify-start');
  });

  it('renders Markdown content correctly', () => {
    render(<MessageBubble role="assistant" content="**Bold Text**" />);
    const boldElement = screen.getByText('Bold Text');
    expect(boldElement.tagName).toBe('STRONG');
  });
});
