import React from 'react';
import { render, screen } from '@testing-library/react';
import DashboardLayout from './DashboardLayout';

describe('DashboardLayout', () => {
  it('renders children correctly', () => {
    render(
      <DashboardLayout>
        <div data-testid="test-child">Child Content</div>
      </DashboardLayout>
    );
    expect(screen.getByTestId('test-child')).toBeInTheDocument();
  });

  it('has a grid layout', () => {
    const { container } = render(
        <DashboardLayout>
            <div>Content</div>
        </DashboardLayout>
    );
    // Check for grid class in the container's first child
    expect(container.firstChild).toHaveClass('grid');
  });
});
