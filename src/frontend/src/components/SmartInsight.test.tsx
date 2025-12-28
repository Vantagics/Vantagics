import React from 'react';
import { render, screen } from '@testing-library/react';
import SmartInsight from './SmartInsight';

describe('SmartInsight', () => {
  it('renders insight text correctly', () => {
    render(<SmartInsight text="Sales increased by 15%!" icon="trending-up" />);
    expect(screen.getByText('Sales increased by 15%!')).toBeInTheDocument();
  });

  it('renders an icon container', () => {
    const { container } = render(<SmartInsight text="Test" icon="star" />);
    // Check if there is an icon container (likely a div or span with specific class)
    const iconElement = container.querySelector('.insight-icon');
    expect(iconElement).toBeInTheDocument();
  });
});
