import React from 'react';
import { render, screen } from '@testing-library/react';
import MetricCard from './MetricCard';

describe('MetricCard', () => {
  it('renders title and value correctly', () => {
    render(<MetricCard title="Total Sales" value="$12,345" change="+15%" />);
    expect(screen.getByText('Total Sales')).toBeInTheDocument();
    expect(screen.getByText('$12,345')).toBeInTheDocument();
  });

  it('renders change with correct styling', () => {
    const { container } = render(<MetricCard title="Sales" value="$100" change="+5%" />);
    expect(screen.getByText('+5%')).toBeInTheDocument();
    // Ideally check for color based on positive/negative change, but basic rendering first
  });
});
