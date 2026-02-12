import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ImportPackDialog from './ImportPackDialog';

// Mock Wails bindings
vi.mock('../../wailsjs/go/main/App', () => ({
    LoadQuickAnalysisPack: vi.fn(),
    LoadQuickAnalysisPackWithPassword: vi.fn(),
    ExecuteQuickAnalysisPack: vi.fn(),
}));

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        language: 'English',
        t: (key: string) => key,
    }),
}));

import {
    LoadQuickAnalysisPack,
    LoadQuickAnalysisPackWithPassword,
    ExecuteQuickAnalysisPack,
} from '../../wailsjs/go/main/App';

const mockLoad = vi.mocked(LoadQuickAnalysisPack);
const mockLoadWithPassword = vi.mocked(LoadQuickAnalysisPackWithPassword);
const mockExecute = vi.mocked(ExecuteQuickAnalysisPack);

const makePackResult = (overrides: any = {}) => ({
    pack: {
        file_type: 'VantageData_QuickAnalysisPack',
        format_version: '1.0',
        metadata: {
            author: 'Alice',
            created_at: '2024-01-15T10:30:00Z',
            source_name: 'test_db',
            description: '',
        },
        schema_requirements: [],
        executable_steps: [
            { step_id: 1, step_type: 'sql_query', code: 'SELECT 1', description: 'test' },
        ],
    },
    validation: {
        compatible: true,
        table_count_match: true,
        source_table_count: 2,
        target_table_count: 2,
        missing_tables: [],
        missing_columns: [],
        extra_tables: [],
    },
    is_encrypted: false,
    needs_password: false,
    file_path: '/tmp/test.qap',
    ...overrides,
});

describe('ImportPackDialog', () => {
    const defaultProps = {
        isOpen: true,
        onClose: vi.fn(),
        onConfirm: vi.fn(),
        dataSourceId: 'ds-123',
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should not render when isOpen is false', () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        render(<ImportPackDialog {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('import_pack_title')).not.toBeInTheDocument();
    });

    it('should show loading state initially', () => {
        mockLoad.mockReturnValue(new Promise(() => {})); // never resolves
        render(<ImportPackDialog {...defaultProps} />);
        expect(screen.getByText('import_pack_loading')).toBeInTheDocument();
    });

    it('should call LoadQuickAnalysisPack with dataSourceId on open', () => {
        mockLoad.mockReturnValue(new Promise(() => {}));
        render(<ImportPackDialog {...defaultProps} />);
        expect(mockLoad).toHaveBeenCalledWith('ds-123');
    });

    it('should show password input when pack needs password', async () => {
        mockLoad.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
        }) as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });
    });

    it('should show preview with metadata when pack loads successfully', async () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Alice')).toBeInTheDocument();
            expect(screen.getByText('test_db')).toBeInTheDocument();
            expect(screen.getByText('1')).toBeInTheDocument(); // steps count
        });
    });

    it('should show schema compatible message when no issues', async () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_schema_compatible')).toBeInTheDocument();
        });
    });

    it('should show missing tables error and disable import', async () => {
        mockLoad.mockResolvedValue(makePackResult({
            validation: {
                compatible: false,
                table_count_match: false,
                source_table_count: 3,
                target_table_count: 1,
                missing_tables: ['orders', 'customers'],
                missing_columns: [],
                extra_tables: [],
            },
        }) as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_missing_tables')).toBeInTheDocument();
            expect(screen.getByText('orders, customers')).toBeInTheDocument();
            const confirmBtn = screen.getByText('import_pack_confirm');
            expect(confirmBtn.closest('button')).toBeDisabled();
        });
    });

    it('should show missing columns warning but allow import', async () => {
        mockLoad.mockResolvedValue(makePackResult({
            validation: {
                compatible: true,
                table_count_match: true,
                source_table_count: 2,
                target_table_count: 2,
                missing_tables: [],
                missing_columns: [{ table_name: 'orders', column_name: 'discount' }],
                extra_tables: [],
            },
        }) as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_missing_columns')).toBeInTheDocument();
            expect(screen.getByText('orders.discount')).toBeInTheDocument();
            const confirmBtn = screen.getByText('import_pack_confirm');
            expect(confirmBtn.closest('button')).not.toBeDisabled();
        });
    });

    it('should call ExecuteQuickAnalysisPack on confirm', async () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        mockExecute.mockResolvedValue(undefined as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_confirm')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('import_pack_confirm'));
        await waitFor(() => {
            expect(mockExecute).toHaveBeenCalledWith('/tmp/test.qap', 'ds-123', '');
        });
    });

    it('should call onConfirm and onClose on successful import', async () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        mockExecute.mockResolvedValue(undefined as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_confirm')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('import_pack_confirm'));
        await waitFor(() => {
            expect(defaultProps.onConfirm).toHaveBeenCalled();
            expect(defaultProps.onClose).toHaveBeenCalled();
        });
    });

    it('should show error when import execution fails', async () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        mockExecute.mockRejectedValue(new Error('Execution failed'));
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_confirm')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('import_pack_confirm'));
        await waitFor(() => {
            expect(screen.getByText('Execution failed')).toBeInTheDocument();
        });
    });

    it('should submit password and show preview on success', async () => {
        mockLoad.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
            file_path: '/tmp/encrypted.qap',
        }) as any);
        mockLoadWithPassword.mockResolvedValue(makePackResult() as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });

        fireEvent.change(screen.getByPlaceholderText('import_pack_password_placeholder'), {
            target: { value: 'mypassword' },
        });
        fireEvent.click(screen.getByText('confirm'));

        await waitFor(() => {
            expect(mockLoadWithPassword).toHaveBeenCalledWith('/tmp/encrypted.qap', 'ds-123', 'mypassword');
            expect(screen.getByText('Alice')).toBeInTheDocument();
        });
    });

    it('should show error on wrong password and allow retry', async () => {
        mockLoad.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
            file_path: '/tmp/encrypted.qap',
        }) as any);
        mockLoadWithPassword.mockRejectedValue(new Error('wrong password'));

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });

        fireEvent.change(screen.getByPlaceholderText('import_pack_password_placeholder'), {
            target: { value: 'badpass' },
        });
        fireEvent.click(screen.getByText('confirm'));

        await waitFor(() => {
            expect(screen.getByText('wrong password')).toBeInTheDocument();
            // Should still be on password screen for retry
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });
    });

    it('should call onClose when cancel is clicked', async () => {
        mockLoad.mockResolvedValue(makePackResult() as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('cancel')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('cancel'));
        expect(defaultProps.onClose).toHaveBeenCalled();
    });
});
