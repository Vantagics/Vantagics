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

  it('renders visual insight when payload is present', () => {
    const payload = JSON.stringify({
        type: 'visual_insight',
        data: { title: 'Sales Trend', value: '$50k', change: '+20%' }
    });
    render(<MessageBubble role="assistant" content="Here is your insight:" payload={payload} />);
    expect(screen.getByText('Sales Trend')).toBeInTheDocument();
    expect(screen.getByText('$50k')).toBeInTheDocument();
  });

  it('renders action buttons when payload is present', () => {
    const payload = JSON.stringify({
        type: 'actions',
        actions: [{ label: 'Export PDF', id: 'export_pdf' }]
    });
    render(<MessageBubble role="assistant" content="Actions available:" payload={payload} />);
    expect(screen.getByText('Export PDF')).toBeInTheDocument();
  });
});
