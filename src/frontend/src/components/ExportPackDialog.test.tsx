import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ExportPackDialog from './ExportPackDialog';

// Mock Wails bindings
vi.mock('../../wailsjs/go/main/App', () => ({
    ExportQuickAnalysisPack: vi.fn(),
}));

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        language: 'English',
        t: (key: string) => key,
    }),
}));

import { ExportQuickAnalysisPack } from '../../wailsjs/go/main/App';

const mockExport = vi.mocked(ExportQuickAnalysisPack);

describe('ExportPackDialog', () => {
    const defaultProps = {
        isOpen: true,
        onClose: vi.fn(),
        onConfirm: vi.fn(),
        threadId: 'test-thread-123',
    };

    beforeEach(() => {
        vi.clearAllMocks();
        mockExport.mockResolvedValue('/path/to/qap/analysis_20250101_120000.qap');
    });

    it('should not render when isOpen is false', () => {
        render(<ExportPackDialog {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('export_pack_title')).not.toBeInTheDocument();
    });

    it('should render dialog with title when isOpen is true', () => {
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.getByText('export_pack_title')).toBeInTheDocument();
    });

    it('should render author input field', () => {
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.getByPlaceholderText('export_pack_author_placeholder')).toBeInTheDocument();
    });

    it('should not render password input field', () => {
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.queryByPlaceholderText('export_pack_password_placeholder')).not.toBeInTheDocument();
    });

    it('should disable export button when author is empty', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const exportBtn = screen.getByText('export');
        expect(exportBtn.closest('button')).toBeDisabled();
    });

    it('should enable export button when pack name and author are filled', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const packNameInput = screen.getByPlaceholderText('export_pack_name_placeholder');
        fireEvent.change(packNameInput, { target: { value: 'Test Pack' } });
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Test Author' } });
        const exportBtn = screen.getByText('export');
        expect(exportBtn.closest('button')).not.toBeDisabled();
    });

    it('should call ExportQuickAnalysisPack with empty password on confirm', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        const packNameInput = screen.getByPlaceholderText('export_pack_name_placeholder');
        fireEvent.change(packNameInput, { target: { value: 'My Analysis' } });
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Alice' } });

        const exportBtn = screen.getByText('export');
        fireEvent.click(exportBtn);

        await waitFor(() => {
            expect(mockExport).toHaveBeenCalledWith('test-thread-123', 'My Analysis', 'Alice', '');
        });
    });

    it('should call onConfirm and show success path on successful export', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        const packNameInput = screen.getByPlaceholderText('export_pack_name_placeholder');
        fireEvent.change(packNameInput, { target: { value: 'Test Pack' } });
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Bob' } });

        const exportBtn = screen.getByText('export');
        fireEvent.click(exportBtn);

        await waitFor(() => {
            expect(defaultProps.onConfirm).toHaveBeenCalledWith('Bob');
            expect(screen.getByText(/export_pack_success/)).toBeInTheDocument();
            expect(screen.getByText(/analysis_20250101_120000\.qap/)).toBeInTheDocument();
        });
    });

    it('should show error message when export fails', async () => {
        mockExport.mockRejectedValue(new Error('Network error'));
        render(<ExportPackDialog {...defaultProps} />);
        const packNameInput = screen.getByPlaceholderText('export_pack_name_placeholder');
        fireEvent.change(packNameInput, { target: { value: 'Test Pack' } });
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Charlie' } });

        const exportBtn = screen.getByText('export');
        fireEvent.click(exportBtn);

        await waitFor(() => {
            expect(screen.getByText('Network error')).toBeInTheDocument();
        });
    });

    it('should call onClose when cancel button is clicked', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const cancelBtn = screen.getByText('cancel');
        fireEvent.click(cancelBtn);
        expect(defaultProps.onClose).toHaveBeenCalled();
    });
});
