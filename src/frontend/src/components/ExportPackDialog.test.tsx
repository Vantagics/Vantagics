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
        mockExport.mockResolvedValue(undefined as any);
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

    it('should render password input field', () => {
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.getByPlaceholderText('export_pack_password_placeholder')).toBeInTheDocument();
    });

    it('should not show confirm password field when password is empty', () => {
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.queryByPlaceholderText('export_pack_confirm_password_placeholder')).not.toBeInTheDocument();
    });

    it('should show confirm password field when password is entered', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const passwordInput = screen.getByPlaceholderText('export_pack_password_placeholder');
        fireEvent.change(passwordInput, { target: { value: 'secret' } });
        expect(screen.getByPlaceholderText('export_pack_confirm_password_placeholder')).toBeInTheDocument();
    });

    it('should show password mismatch error when passwords differ', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const passwordInput = screen.getByPlaceholderText('export_pack_password_placeholder');
        fireEvent.change(passwordInput, { target: { value: 'secret1' } });
        const confirmInput = screen.getByPlaceholderText('export_pack_confirm_password_placeholder');
        fireEvent.change(confirmInput, { target: { value: 'secret2' } });
        expect(screen.getByText('export_pack_password_mismatch')).toBeInTheDocument();
    });

    it('should not show mismatch error when passwords match', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const passwordInput = screen.getByPlaceholderText('export_pack_password_placeholder');
        fireEvent.change(passwordInput, { target: { value: 'secret' } });
        const confirmInput = screen.getByPlaceholderText('export_pack_confirm_password_placeholder');
        fireEvent.change(confirmInput, { target: { value: 'secret' } });
        expect(screen.queryByText('export_pack_password_mismatch')).not.toBeInTheDocument();
    });

    it('should disable export button when author is empty', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const exportBtn = screen.getByText('export');
        expect(exportBtn.closest('button')).toBeDisabled();
    });

    it('should enable export button when author is filled and no password mismatch', () => {
        render(<ExportPackDialog {...defaultProps} />);
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Test Author' } });
        const exportBtn = screen.getByText('export');
        expect(exportBtn.closest('button')).not.toBeDisabled();
    });

    it('should call ExportQuickAnalysisPack with correct args on confirm', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Alice' } });
        const passwordInput = screen.getByPlaceholderText('export_pack_password_placeholder');
        fireEvent.change(passwordInput, { target: { value: 'pass123' } });
        const confirmInput = screen.getByPlaceholderText('export_pack_confirm_password_placeholder');
        fireEvent.change(confirmInput, { target: { value: 'pass123' } });

        const exportBtn = screen.getByText('export');
        fireEvent.click(exportBtn);

        await waitFor(() => {
            expect(mockExport).toHaveBeenCalledWith('test-thread-123', 'Alice', 'pass123');
        });
    });

    it('should call onConfirm and onClose on successful export', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Bob' } });

        const exportBtn = screen.getByText('export');
        fireEvent.click(exportBtn);

        await waitFor(() => {
            expect(defaultProps.onConfirm).toHaveBeenCalledWith('Bob', '');
            expect(defaultProps.onClose).toHaveBeenCalled();
        });
    });

    it('should show error message when export fails', async () => {
        mockExport.mockRejectedValue(new Error('Network error'));
        render(<ExportPackDialog {...defaultProps} />);
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
