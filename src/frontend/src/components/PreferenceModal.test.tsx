import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import PreferenceModal from './PreferenceModal';
import * as AppBindings from '../../wailsjs/go/main/App';
import { vi } from 'vitest';

vi.mock('../../wailsjs/go/main/App', () => ({
    GetConfig: vi.fn(),
    SaveConfig: vi.fn(),
}));

describe('PreferenceModal', () => {
    it('loads and saves LLM configuration', async () => {
        const mockConfig = {
            llmProvider: 'OpenAI',
            apiKey: 'old-key',
            baseUrl: '',
            modelName: 'gpt-4',
            maxTokens: 4096,
        };

        (AppBindings.GetConfig as any).mockResolvedValue(mockConfig);
        (AppBindings.SaveConfig as any).mockResolvedValue({});

        const handleClose = vi.fn();
        render(<PreferenceModal isOpen={true} onClose={handleClose} />);

        // Check if config loaded
        await waitFor(() => {
            const apiKeyInput = screen.getByLabelText(/API Key/i) as HTMLInputElement;
            expect(apiKeyInput.value).toBe('old-key');
        });

        // Change provider
        const providerSelect = screen.getByLabelText(/Provider Type/i);
        fireEvent.change(providerSelect, { target: { value: 'Anthropic' } });

        // Save
        const saveButton = screen.getByText(/Save Changes/i);
        fireEvent.click(saveButton);

        await waitFor(() => {
            expect(AppBindings.SaveConfig).toHaveBeenCalledWith(expect.objectContaining({
                llmProvider: 'Anthropic'
            }));
            expect(handleClose).toHaveBeenCalled();
        });
    });

    it('supports Claude-Compatible provider configuration', async () => {
        const mockConfig = {
            llmProvider: 'OpenAI',
            apiKey: '',
            baseUrl: '',
            modelName: '',
            maxTokens: 4096,
            claudeHeaderStyle: 'Anthropic',
        };

        (AppBindings.GetConfig as any).mockResolvedValue(mockConfig);

        render(<PreferenceModal isOpen={true} onClose={() => {}} />);

        await waitFor(() => {
            screen.getByLabelText(/Provider Type/i);
        });

        const providerSelect = screen.getByLabelText(/Provider Type/i);
        
        // Check if option exists (this will fail if not added)
        const claudeOption = screen.getByRole('option', { name: /Claude-Compatible/i });
        expect(claudeOption).toBeInTheDocument();

        // Select Claude-Compatible
        fireEvent.change(providerSelect, { target: { value: 'Claude-Compatible' } });

        // Check for Header Style option (this will fail if not implemented)
        const headerStyleSelect = await screen.findByLabelText(/Header Style/i);
        expect(headerStyleSelect).toBeInTheDocument();
        
        // Change header style
        fireEvent.change(headerStyleSelect, { target: { value: 'OpenAI' } });
        
        // Save
        const saveButton = screen.getByText(/Save Changes/i);
        fireEvent.click(saveButton);
        
        await waitFor(() => {
            expect(AppBindings.SaveConfig).toHaveBeenCalledWith(expect.objectContaining({
                llmProvider: 'Claude-Compatible',
                claudeHeaderStyle: 'OpenAI'
            }));
        });
    });

    it('allows changing Data Cache Directory in System Parameters', async () => {
        const mockConfig = {
            llmProvider: 'OpenAI',
            apiKey: '',
            baseUrl: '',
            modelName: '',
            maxTokens: 4096,
            darkMode: false,
            localCache: true,
            language: 'English',
            claudeHeaderStyle: 'Anthropic',
            dataCacheDir: '~/RapidBI'
        };

        (AppBindings.GetConfig as any).mockResolvedValue(mockConfig);
        (AppBindings.SaveConfig as any).mockResolvedValue({});

        render(<PreferenceModal isOpen={true} onClose={() => {}} />);

        // Switch to System Parameters tab
        const systemTab = await screen.findByText(/System Parameters/i);
        fireEvent.click(systemTab);

        // Check if field exists and has value
        const cacheDirInput = await screen.findByLabelText(/Data Cache Directory/i) as HTMLInputElement;
        expect(cacheDirInput.value).toBe('~/RapidBI');

        // Change value
        fireEvent.change(cacheDirInput, { target: { value: '/tmp/RapidBI' } });

        // Save
        const saveButton = screen.getByText(/Save Changes/i);
        fireEvent.click(saveButton);

        await waitFor(() => {
            expect(AppBindings.SaveConfig).toHaveBeenCalledWith(expect.objectContaining({
                dataCacheDir: '/tmp/RapidBI'
            }));
        });
    });
});
