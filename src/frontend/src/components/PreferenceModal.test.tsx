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
});
