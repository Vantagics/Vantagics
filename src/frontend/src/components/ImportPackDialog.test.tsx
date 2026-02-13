import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ImportPackDialog from './ImportPackDialog';

// Mock Wails bindings — must include ALL APIs used by the refactored component
vi.mock('../../wailsjs/go/main/App', () => ({
    LoadQuickAnalysisPack: vi.fn(),
    LoadQuickAnalysisPackWithPassword: vi.fn(),
    ExecuteQuickAnalysisPack: vi.fn(),
    ListLocalQuickAnalysisPacks: vi.fn(),
    LoadQuickAnalysisPackByPath: vi.fn(),
}));

// Mock i18n — returns the key as-is
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
    ListLocalQuickAnalysisPacks,
    LoadQuickAnalysisPackByPath,
} from '../../wailsjs/go/main/App';

const mockListPacks = vi.mocked(ListLocalQuickAnalysisPacks);
const mockLoadByPath = vi.mocked(LoadQuickAnalysisPackByPath);
const mockLoadBrowse = vi.mocked(LoadQuickAnalysisPack);
const mockLoadWithPassword = vi.mocked(LoadQuickAnalysisPackWithPassword);
const mockExecute = vi.mocked(ExecuteQuickAnalysisPack);

const mockPackInfo = {
    file_name: 'test_pack.qap',
    file_path: '/tmp/qap/test_pack.qap',
    pack_name: 'Test Analysis Pack',
    description: 'A test pack',
    source_name: 'test_db',
    author: 'Alice',
    created_at: '2024-01-15T10:30:00Z',
    is_encrypted: false,
};

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
    file_path: '/tmp/qap/test_pack.qap',
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

    // 1. Basic render test
    it('should not render when isOpen is false', () => {
        mockListPacks.mockResolvedValue([]);
        render(<ImportPackDialog {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('import_pack_title')).not.toBeInTheDocument();
    });

    // 2. Loading state while fetching pack list
    it('should show loading state while fetching pack list', () => {
        mockListPacks.mockReturnValue(new Promise(() => {})); // never resolves
        render(<ImportPackDialog {...defaultProps} />);
        expect(screen.getByText('import_pack_loading')).toBeInTheDocument();
    });

    // 3. Show pack list after loading
    it('should show pack list after loading', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
            expect(screen.getByText('A test pack')).toBeInTheDocument();
        });
    });

    // 4. Empty state when no packs
    it('should show empty state when no packs', async () => {
        mockListPacks.mockResolvedValue([]);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_empty_hint')).toBeInTheDocument();
        });
    });

    // 5. Click pack item → preview state
    it('should show preview when clicking a pack item', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult() as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(mockLoadByPath).toHaveBeenCalledWith('/tmp/qap/test_pack.qap', 'ds-123');
            expect(screen.getByText('Alice')).toBeInTheDocument();
            expect(screen.getByText('test_db')).toBeInTheDocument();
        });
    });

    // 6. Click pack → needs_password=true → password input
    it('should show password input when pack needs password', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
            file_path: '/tmp/qap/test_pack.qap',
        }) as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });
    });

    // 7. Browse file button → LoadQuickAnalysisPack → preview
    it('should handle browse file button', async () => {
        mockListPacks.mockResolvedValue([]);
        mockLoadBrowse.mockResolvedValue(makePackResult() as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('import_pack_browse_file')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('import_pack_browse_file'));

        await waitFor(() => {
            expect(mockLoadBrowse).toHaveBeenCalledWith('ds-123');
            expect(screen.getByText('Alice')).toBeInTheDocument();
        });
    });

    // 8. Password flow → submit → preview
    it('should submit password and show preview', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
            file_path: '/tmp/qap/test_pack.qap',
        }) as any);
        mockLoadWithPassword.mockResolvedValue(makePackResult() as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });

        fireEvent.change(screen.getByPlaceholderText('import_pack_password_placeholder'), {
            target: { value: 'mypassword' },
        });
        fireEvent.click(screen.getByText('confirm'));

        await waitFor(() => {
            expect(mockLoadWithPassword).toHaveBeenCalledWith('/tmp/qap/test_pack.qap', 'ds-123', 'mypassword');
            expect(screen.getByText('Alice')).toBeInTheDocument();
        });
    });

    // 9. Wrong password → error
    it('should show error on wrong password', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
            file_path: '/tmp/qap/test_pack.qap',
        }) as any);
        mockLoadWithPassword.mockRejectedValue(new Error('wrong password'));

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });

        fireEvent.change(screen.getByPlaceholderText('import_pack_password_placeholder'), {
            target: { value: 'badpass' },
        });
        fireEvent.click(screen.getByText('confirm'));

        await waitFor(() => {
            expect(screen.getByText('wrong password')).toBeInTheDocument();
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });
    });

    // 10. Execute pack on confirm
    it('should execute pack on confirm', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult() as any);
        mockExecute.mockResolvedValue(undefined as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByText('import_pack_confirm')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('import_pack_confirm'));

        await waitFor(() => {
            expect(mockExecute).toHaveBeenCalledWith('/tmp/qap/test_pack.qap', 'ds-123', '');
        });
    });

    // 11. Successful execution → onConfirm + onClose
    it('should call onConfirm and onClose on successful execution', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult() as any);
        mockExecute.mockResolvedValue(undefined as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByText('import_pack_confirm')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('import_pack_confirm'));

        await waitFor(() => {
            expect(defaultProps.onConfirm).toHaveBeenCalled();
            expect(defaultProps.onClose).toHaveBeenCalled();
        });
    });

    // 12. Execution failure → error shown
    it('should show error when execution fails', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult() as any);
        mockExecute.mockRejectedValue(new Error('Execution failed'));

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByText('import_pack_confirm')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('import_pack_confirm'));

        await waitFor(() => {
            expect(screen.getByText('Execution failed')).toBeInTheDocument();
        });
    });

    // 13. Back button from preview → pack-list
    it('should go back to pack-list from preview', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult() as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByText('Alice')).toBeInTheDocument();
        });

        // Click back button
        fireEvent.click(screen.getByText('import_pack_back'));

        await waitFor(() => {
            // Should be back on pack-list showing the pack list again
            expect(screen.getByText('import_pack_browse_file')).toBeInTheDocument();
        });
    });

    // 14. Back button from password → pack-list
    it('should go back to pack-list from password', async () => {
        mockListPacks.mockResolvedValue([mockPackInfo] as any);
        mockLoadByPath.mockResolvedValue(makePackResult({
            needs_password: true,
            is_encrypted: true,
            pack: null,
            validation: null,
            file_path: '/tmp/qap/test_pack.qap',
        }) as any);

        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('Test Analysis Pack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Test Analysis Pack'));

        await waitFor(() => {
            expect(screen.getByPlaceholderText('import_pack_password_placeholder')).toBeInTheDocument();
        });

        // Click back button
        fireEvent.click(screen.getByText('import_pack_back'));

        await waitFor(() => {
            expect(screen.getByText('import_pack_browse_file')).toBeInTheDocument();
        });
    });

    // 15. Cancel button → onClose
    it('should call onClose when cancel is clicked', async () => {
        mockListPacks.mockResolvedValue([]);
        render(<ImportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('cancel')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('cancel'));
        expect(defaultProps.onClose).toHaveBeenCalled();
    });
});
