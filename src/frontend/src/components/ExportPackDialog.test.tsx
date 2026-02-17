import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ExportPackDialog from './ExportPackDialog';

// Mock Wails bindings
vi.mock('../../wailsjs/go/main/App', () => ({
    ExportQuickAnalysisPackSelected: vi.fn(),
    GetConfig: vi.fn().mockResolvedValue({ authorSignature: '' }),
    GetThreadExportableRequests: vi.fn(),
}));

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        language: 'English',
        t: (key: string) => key,
    }),
}));

import { ExportQuickAnalysisPackSelected, GetThreadExportableRequests } from '../../wailsjs/go/main/App';

const mockExport = vi.mocked(ExportQuickAnalysisPackSelected);
const mockGetRequests = vi.mocked(GetThreadExportableRequests);

const sampleRequests = [
    { request_id: 'msg1', user_request: 'Give me some analysis suggestions for this data source.', step_count: 3, timestamp: 1000, is_auto_suggestion: true },
    { request_id: 'msg2', user_request: '分析销售趋势', step_count: 2, timestamp: 2000, is_auto_suggestion: false },
    { request_id: 'msg3', user_request: '客户分群分析', step_count: 4, timestamp: 3000, is_auto_suggestion: false },
];

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
        mockGetRequests.mockResolvedValue(sampleRequests as any);
    });

    it('should not render when isOpen is false', () => {
        render(<ExportPackDialog {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('export_pack_title')).not.toBeInTheDocument();
    });

    it('should render dialog with title when isOpen is true', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.getByText('export_pack_title')).toBeInTheDocument();
        // Wait for async effects to settle to avoid act() warnings
        await waitFor(() => {
            expect(screen.getByText('export_pack_select_hint')).toBeInTheDocument();
        });
    });

    it('should show loading state initially', () => {
        mockGetRequests.mockReturnValue(new Promise(() => {})); // never resolves
        render(<ExportPackDialog {...defaultProps} />);
        expect(screen.getByText('export_pack_loading_requests')).toBeInTheDocument();
    });

    it('should show request selection after loading', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('export_pack_select_hint')).toBeInTheDocument();
        });
        // All requests should be visible
        expect(screen.getByText('分析销售趋势')).toBeInTheDocument();
        expect(screen.getByText('客户分群分析')).toBeInTheDocument();
    });

    it('should not select auto-suggestion by default', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('export_pack_select_hint')).toBeInTheDocument();
        });
        // Auto suggestion badge should be visible
        expect(screen.getByText('export_pack_auto_suggestion')).toBeInTheDocument();
    });

    it('should navigate to form step when Next is clicked', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('export_pack_next')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('export_pack_next'));
        await waitFor(() => {
            expect(screen.getByPlaceholderText('export_pack_name_placeholder')).toBeInTheDocument();
            expect(screen.getByPlaceholderText('export_pack_author_placeholder')).toBeInTheDocument();
        });
    });

    it('should call ExportQuickAnalysisPackSelected with selected IDs on confirm', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('export_pack_next')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('export_pack_next'));

        await waitFor(() => {
            expect(screen.getByPlaceholderText('export_pack_name_placeholder')).toBeInTheDocument();
        });

        const packNameInput = screen.getByPlaceholderText('export_pack_name_placeholder');
        fireEvent.change(packNameInput, { target: { value: 'My Analysis' } });
        const authorInput = screen.getByPlaceholderText('export_pack_author_placeholder');
        fireEvent.change(authorInput, { target: { value: 'Alice' } });

        const exportBtn = screen.getByText('export');
        fireEvent.click(exportBtn);

        await waitFor(() => {
            expect(mockExport).toHaveBeenCalledWith(
                'test-thread-123', 'My Analysis', 'Alice', '',
                expect.arrayContaining(['msg2', 'msg3'])
            );
            // Auto-suggestion (msg1) should NOT be in the selected IDs
            const selectedIds = mockExport.mock.calls[0][4];
            expect(selectedIds).not.toContain('msg1');
        });
    });

    it('should call onClose when cancel button is clicked', async () => {
        render(<ExportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('cancel')).toBeInTheDocument();
        });
        fireEvent.click(screen.getByText('cancel'));
        expect(defaultProps.onClose).toHaveBeenCalled();
    });

    it('should show error when loading requests fails', async () => {
        mockGetRequests.mockRejectedValue(new Error('No data'));
        render(<ExportPackDialog {...defaultProps} />);
        await waitFor(() => {
            expect(screen.getByText('No data')).toBeInTheDocument();
        });
    });
});
