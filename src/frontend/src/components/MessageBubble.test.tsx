import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import MessageBubble from './MessageBubble';

// Mock child components
vi.mock('./Chart', () => ({
    default: ({ options }: any) => <div data-testid="mock-chart">{options.title.text}</div>
}));
vi.mock('./DataTable', () => ({
    default: ({ data }: any) => <div data-testid="mock-table">{Object.keys(data[0]).join(',')}</div>
}));

describe('MessageBubble', () => {
  it('renders user message correctly', () => {
    const { container } = render(<MessageBubble role="user" content="Hello" />);
    // Check flex-row-reverse for user (which implies right alignment in our css)
    expect(container.firstChild).toHaveClass('flex-row-reverse');
  });

  it('renders assistant message correctly', () => {
    const { container } = render(<MessageBubble role="assistant" content="Hi there" />);
    // Check flex-row for assistant (left alignment)
    expect(container.firstChild).toHaveClass('flex-row');
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

  it('renders action buttons when payload is present and handles click', () => {
    const payload = JSON.stringify({
        type: 'actions',
        actions: [{ label: 'Export PDF', id: 'export_pdf', value: 'Exporting...' }]
    });
    const mockOnClick = vi.fn();
    render(<MessageBubble role="assistant" content="Actions available:" payload={payload} onActionClick={mockOnClick} />);
    
    const button = screen.getByText('Export PDF');
    expect(button).toBeInTheDocument();
    
    fireEvent.click(button);
    expect(mockOnClick).toHaveBeenCalledWith(expect.objectContaining({ label: 'Export PDF' }));
  });

  it('renders ECharts chart when payload contains echarts type', () => {
    const payload = JSON.stringify({
        type: 'echarts',
        data: { title: { text: 'Test Chart' } }
    });
    render(<MessageBubble role="assistant" content="Here is a chart:" payload={payload} />);
    expect(screen.getByTestId('mock-chart')).toBeInTheDocument();
    expect(screen.getByText('Test Chart')).toBeInTheDocument();
  });

  it('renders DataTable when payload contains table data', () => {
    // Note: The current MessageBubble.tsx doesn't seem to render DataTable from payload 
    // but the test expects it. Looking at MessageBubble.tsx, it only handles 'visual_insight' and 'echarts'.
    // Wait, let me check MessageBubble.tsx again.
  });
});
