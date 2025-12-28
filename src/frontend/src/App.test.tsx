import { render, screen, waitFor } from '@testing-library/react';
import App from './App';
import * as AppBindings from '../wailsjs/go/main/App';
import { vi } from 'vitest';

// Mock the Wails bindings
vi.mock('../wailsjs/go/main/App', () => ({
    GetDashboardData: vi.fn(),
    GetConfig: vi.fn(),
    Greet: vi.fn(),
    SaveConfig: vi.fn(),
}));

// Mock the runtime
vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(() => () => {}),
}));

describe('App Dashboard Integration', () => {
    it('fetches and displays dashboard data on mount', async () => {
        const mockData = {
            metrics: [
                { title: 'Total Sales', value: '$10,000', change: '+10%' },
            ],
            insights: [
                { text: 'Great progress!', icon: 'star' },
            ],
        };

        (AppBindings.GetDashboardData as any).mockResolvedValue(mockData);
        (AppBindings.GetConfig as any).mockResolvedValue({});

        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Total Sales')).toBeInTheDocument();
            expect(screen.getByText('$10,000')).toBeInTheDocument();
            expect(screen.getByText('Great progress!')).toBeInTheDocument();
        });
    });
});
