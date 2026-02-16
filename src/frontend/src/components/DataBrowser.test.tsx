import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import DataBrowser, { DataBrowserProps, inferColumnType, formatCellValue, ROWS_PER_PAGE } from './DataBrowser';

// Mock the Wails backend functions
vi.mock('../../wailsjs/go/main/App', () => ({
    GetDataSourceTables: vi.fn(),
    GetDataSourceTableData: vi.fn(),
    GetDataSourceTableCount: vi.fn(),
    GetDataSourceTableDataWithCount: vi.fn(),
}));

import { GetDataSourceTables, GetDataSourceTableData, GetDataSourceTableCount, GetDataSourceTableDataWithCount } from '../../wailsjs/go/main/App';

const mockGetDataSourceTables = vi.mocked(GetDataSourceTables);
const mockGetDataSourceTableData = vi.mocked(GetDataSourceTableData);
const mockGetDataSourceTableCount = vi.mocked(GetDataSourceTableCount);
const mockGetDataSourceTableDataWithCount = vi.mocked(GetDataSourceTableDataWithCount);

const defaultProps: DataBrowserProps = {
    isOpen: false,
    sourceId: null,
    sourceName: null,
    onClose: vi.fn(),
    width: 500,
    onWidthChange: vi.fn(),
};

const sampleTables = ['users', 'orders', 'products'];
const sampleTableData = [
    { id: 1, name: 'Alice', email: 'alice@example.com', age: 30 },
    { id: 2, name: 'Bob', email: 'bob@example.com', age: 25 },
    { id: 3, name: 'Charlie', email: 'charlie@example.com', age: 35 },
];

describe('DataBrowser Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockGetDataSourceTables.mockResolvedValue([]);
        mockGetDataSourceTableData.mockResolvedValue([]);
        mockGetDataSourceTableCount.mockResolvedValue(0);
        mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: [], rowCount: 0 });
    });

    afterEach(() => {
        document.body.style.userSelect = '';
        document.body.style.cursor = '';
    });

    describe('Rendering', () => {
        it('should render the data browser panel', () => {
            render(<DataBrowser {...defaultProps} />);
            expect(screen.getByTestId('data-browser')).toBeInTheDocument();
        });

        it('should render the backdrop element', () => {
            render(<DataBrowser {...defaultProps} />);
            expect(screen.getByTestId('data-browser-backdrop')).toBeInTheDocument();
        });

        it('should render the header with close button', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />);
            expect(screen.getByTestId('data-browser-header')).toBeInTheDocument();
            expect(screen.getByTestId('data-browser-close-button')).toBeInTheDocument();
        });

        it('should render the resize handle on left edge', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            expect(screen.getByTestId('data-browser-resize-handle')).toBeInTheDocument();
        });

        it('should render the content area', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />);
            expect(screen.getByTestId('data-browser-content')).toBeInTheDocument();
        });

        it('should apply the specified width', () => {
            render(<DataBrowser {...defaultProps} width={600} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel).toHaveStyle({ width: '600px' });
        });
    });

    describe('ARIA and Accessibility', () => {
        it('should have dialog role', () => {
            render(<DataBrowser {...defaultProps} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel).toHaveAttribute('role', 'dialog');
        });

        it('should have aria-label', () => {
            render(<DataBrowser {...defaultProps} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel).toHaveAttribute('aria-label', 'Data Browser');
        });

        it('should set aria-modal to true when open', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel).toHaveAttribute('aria-modal', 'true');
        });

        it('should set aria-hidden to true when closed', () => {
            render(<DataBrowser {...defaultProps} isOpen={false} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel).toHaveAttribute('aria-hidden', 'true');
        });

        it('should set aria-hidden to false when open', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel).toHaveAttribute('aria-hidden', 'false');
        });

        it('should have aria-label on close button', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const closeBtn = screen.getByTestId('data-browser-close-button');
            expect(closeBtn).toHaveAttribute('aria-label', 'Close data browser');
        });

        it('should have separator role on resize handle', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const handle = screen.getByTestId('data-browser-resize-handle');
            expect(handle).toHaveAttribute('role', 'separator');
            expect(handle).toHaveAttribute('aria-orientation', 'vertical');
        });
    });

    describe('Slide-in/out Animation (Requirements 7.1, 7.6)', () => {
        it('should not have open class when closed', () => {
            render(<DataBrowser {...defaultProps} isOpen={false} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel.className).not.toContain('open');
        });

        it('should have open class when open', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel.className).toContain('open');
        });

        it('should transition from closed to open', () => {
            const { rerender } = render(<DataBrowser {...defaultProps} isOpen={false} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel.className).not.toContain('open');

            rerender(<DataBrowser {...defaultProps} isOpen={true} />);
            expect(panel.className).toContain('open');
        });

        it('should transition from open to closed', () => {
            const { rerender } = render(<DataBrowser {...defaultProps} isOpen={true} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel.className).toContain('open');

            rerender(<DataBrowser {...defaultProps} isOpen={false} />);
            expect(panel.className).not.toContain('open');
        });
    });

    describe('Close Button (Requirement 7.5)', () => {
        it('should call onClose when close button is clicked', () => {
            const onClose = vi.fn();
            render(<DataBrowser {...defaultProps} isOpen={true} onClose={onClose} />);

            fireEvent.click(screen.getByTestId('data-browser-close-button'));
            expect(onClose).toHaveBeenCalledTimes(1);
        });
    });

    describe('Escape Key (Requirement 7.6)', () => {
        it('should call onClose when Escape is pressed while open', () => {
            const onClose = vi.fn();
            render(<DataBrowser {...defaultProps} isOpen={true} onClose={onClose} />);

            fireEvent.keyDown(document, { key: 'Escape' });
            expect(onClose).toHaveBeenCalledTimes(1);
        });

        it('should not call onClose when Escape is pressed while closed', () => {
            const onClose = vi.fn();
            render(<DataBrowser {...defaultProps} isOpen={false} onClose={onClose} />);

            fireEvent.keyDown(document, { key: 'Escape' });
            expect(onClose).not.toHaveBeenCalled();
        });

        it('should not call onClose for non-Escape keys', () => {
            const onClose = vi.fn();
            render(<DataBrowser {...defaultProps} isOpen={true} onClose={onClose} />);

            fireEvent.keyDown(document, { key: 'Enter' });
            expect(onClose).not.toHaveBeenCalled();
        });
    });

    describe('Backdrop (Requirement 7.9)', () => {
        it('should show visible backdrop when open', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const backdrop = screen.getByTestId('data-browser-backdrop');
            expect(backdrop.className).toContain('visible');
        });

        it('should not show visible backdrop when closed', () => {
            render(<DataBrowser {...defaultProps} isOpen={false} />);
            const backdrop = screen.getByTestId('data-browser-backdrop');
            expect(backdrop.className).not.toContain('visible');
        });

        it('should call onClose when backdrop is clicked', () => {
            const onClose = vi.fn();
            render(<DataBrowser {...defaultProps} isOpen={true} onClose={onClose} />);

            fireEvent.click(screen.getByTestId('data-browser-backdrop'));
            expect(onClose).toHaveBeenCalledTimes(1);
        });

        it('should not call onClose when backdrop is clicked while closed', () => {
            const onClose = vi.fn();
            render(<DataBrowser {...defaultProps} isOpen={false} onClose={onClose} />);

            fireEvent.click(screen.getByTestId('data-browser-backdrop'));
            expect(onClose).not.toHaveBeenCalled();
        });
    });

    describe('Resize Handle (Requirement 7.10)', () => {
        it('should start resizing on mouse down', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} width={500} />);
            const handle = screen.getByTestId('data-browser-resize-handle');

            fireEvent.mouseDown(handle, { clientX: 200, clientY: 100 });

            expect(handle.className).toContain('dragging');
            expect(document.body.style.userSelect).toBe('none');
            expect(document.body.style.cursor).toBe('col-resize');
        });

        it('should call onWidthChange during resize drag', () => {
            const onWidthChange = vi.fn();
            render(
                <DataBrowser
                    {...defaultProps}
                    isOpen={true}
                    width={500}
                    onWidthChange={onWidthChange}
                />
            );
            const handle = screen.getByTestId('data-browser-resize-handle');

            fireEvent.mouseDown(handle, { clientX: 200, clientY: 100 });
            fireEvent.mouseMove(window, { clientX: 150, clientY: 100 });

            expect(onWidthChange).toHaveBeenCalledWith(550);
        });

        it('should enforce minimum width during resize', () => {
            const onWidthChange = vi.fn();
            render(
                <DataBrowser
                    {...defaultProps}
                    isOpen={true}
                    width={350}
                    onWidthChange={onWidthChange}
                />
            );
            const handle = screen.getByTestId('data-browser-resize-handle');

            fireEvent.mouseDown(handle, { clientX: 200, clientY: 100 });
            fireEvent.mouseMove(window, { clientX: 400, clientY: 100 });

            expect(onWidthChange).toHaveBeenCalledWith(300);
        });

        it('should stop resizing on mouse up', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} width={500} />);
            const handle = screen.getByTestId('data-browser-resize-handle');

            fireEvent.mouseDown(handle, { clientX: 200, clientY: 100 });
            expect(handle.className).toContain('dragging');

            fireEvent.mouseUp(window);
            expect(handle.className).not.toContain('dragging');
            expect(document.body.style.userSelect).toBe('');
            expect(document.body.style.cursor).toBe('');
        });

        it('should not respond to mouse move when not dragging', () => {
            const onWidthChange = vi.fn();
            render(
                <DataBrowser
                    {...defaultProps}
                    isOpen={true}
                    width={500}
                    onWidthChange={onWidthChange}
                />
            );

            fireEvent.mouseMove(window, { clientX: 150, clientY: 100 });
            expect(onWidthChange).not.toHaveBeenCalled();
        });
    });

    describe('Header Display (Requirement 8.1)', () => {
        it('should display source name in header when provided', () => {
            render(
                <DataBrowser
                    {...defaultProps}
                    isOpen={true}
                    sourceId="src-1"
                    sourceName="My Sales Data"
                />
            );
            expect(screen.getByTestId('data-browser-title')).toHaveTextContent('My Sales Data');
        });

        it('should display "Data Browser" when sourceId is provided but no sourceName', () => {
            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" sourceName={null} />
            );
            expect(screen.getByTestId('data-browser-title')).toHaveTextContent('Data Browser');
        });

        it('should display "No Data Source" when sourceId is null', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} sourceId={null} />);
            expect(screen.getByTestId('data-browser-title')).toHaveTextContent('No Data Source');
        });

        it('should display selected table name in header when a table is selected', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser
                    {...defaultProps}
                    isOpen={true}
                    sourceId="src-1"
                    sourceName="My Data"
                />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('data-browser-title')).toHaveTextContent('users');
            });
        });
    });

    describe('No Source Selected', () => {
        it('should show no source message when sourceId is null', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} sourceId={null} />);
            expect(screen.getByTestId('data-browser-no-source')).toBeInTheDocument();
            expect(screen.getByText('No data source selected')).toBeInTheDocument();
        });

        it('should not show search bar when sourceId is null', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} sourceId={null} />);
            expect(screen.queryByTestId('db-search-bar')).not.toBeInTheDocument();
        });
    });

    describe('Table List Loading (Requirement 8.2)', () => {
        it('should load tables when opened with a sourceId', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(mockGetDataSourceTables).toHaveBeenCalledWith('src-1');
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            expect(screen.getByTestId('db-table-item-users')).toBeInTheDocument();
            expect(screen.getByTestId('db-table-item-orders')).toBeInTheDocument();
            expect(screen.getByTestId('db-table-item-products')).toBeInTheDocument();
        });

        it('should show loading state while tables are loading', async () => {
            let resolvePromise: (value: string[]) => void;
            mockGetDataSourceTables.mockReturnValue(
                new Promise((resolve) => { resolvePromise = resolve; })
            );

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            expect(screen.getByTestId('db-loading-tables')).toBeInTheDocument();

            await act(async () => {
                resolvePromise!(sampleTables);
            });

            await waitFor(() => {
                expect(screen.queryByTestId('db-loading-tables')).not.toBeInTheDocument();
            });
        });

        it('should show empty state when no tables exist', async () => {
            mockGetDataSourceTables.mockResolvedValue([]);

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-no-tables')).toBeInTheDocument();
            });
        });

        it('should reset state when closed', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);

            const { rerender } = render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            rerender(<DataBrowser {...defaultProps} isOpen={false} sourceId="src-1" />);

            // When reopened, it should load fresh
            rerender(<DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />);
            expect(mockGetDataSourceTables).toHaveBeenCalledTimes(2);
        });
    });

    describe('Table Selection (Requirement 8.3)', () => {
        it('should load table data when a table is selected', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(mockGetDataSourceTableDataWithCount).toHaveBeenCalledWith('src-1', 'users');
            });
        });

        it('should show back button when a table is selected', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-back-button')).toBeInTheDocument();
            });
        });

        it('should go back to table list when back button is clicked', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-back-button')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-back-button'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });
        });
    });

    describe('Column and Data Type Display (Requirement 8.3)', () => {
        it('should display columns with data types', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-columns-list')).toBeInTheDocument();
            });

            expect(screen.getByTestId('db-column-id')).toBeInTheDocument();
            expect(screen.getByTestId('db-column-name')).toBeInTheDocument();
            expect(screen.getByTestId('db-column-email')).toBeInTheDocument();
            expect(screen.getByTestId('db-column-age')).toBeInTheDocument();

            // Check data types
            expect(screen.getByTestId('db-column-type-id')).toHaveTextContent('integer');
            expect(screen.getByTestId('db-column-type-name')).toHaveTextContent('text');
            expect(screen.getByTestId('db-column-type-age')).toHaveTextContent('integer');
        });
    });

    describe('Sample Data Display (Requirement 8.4)', () => {
        it('should display sample data rows in a table', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-data-table')).toBeInTheDocument();
            });

            expect(screen.getByTestId('db-data-row-0')).toBeInTheDocument();
            expect(screen.getByTestId('db-data-row-1')).toBeInTheDocument();
            expect(screen.getByTestId('db-data-row-2')).toBeInTheDocument();
        });

        it('should show no data message when table has no rows', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: [], rowCount: 0 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-no-data')).toBeInTheDocument();
            });
        });
    });

    describe('Pagination (Requirement 8.5)', () => {
        it('should show pagination when data exceeds page size', async () => {
            const manyRows = Array.from({ length: 30 }, (_, i) => ({
                id: i + 1,
                name: `User ${i + 1}`,
            }));
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: manyRows, rowCount: 30 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-pagination')).toBeInTheDocument();
            });

            expect(screen.getByTestId('db-pagination-info')).toHaveTextContent('Page 1 of 2');
        });

        it('should navigate to next page', async () => {
            const manyRows = Array.from({ length: 30 }, (_, i) => ({
                id: i + 1,
                name: `User ${i + 1}`,
            }));
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: manyRows, rowCount: 30 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-pagination')).toBeInTheDocument();
            });

            fireEvent.click(screen.getByTestId('db-pagination-next'));
            expect(screen.getByTestId('db-pagination-info')).toHaveTextContent('Page 2 of 2');
        });

        it('should disable prev button on first page', async () => {
            const manyRows = Array.from({ length: 30 }, (_, i) => ({
                id: i + 1,
                name: `User ${i + 1}`,
            }));
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: manyRows, rowCount: 30 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-pagination')).toBeInTheDocument();
            });

            expect(screen.getByTestId('db-pagination-prev')).toBeDisabled();
        });

        it('should disable next button on last page', async () => {
            const manyRows = Array.from({ length: 30 }, (_, i) => ({
                id: i + 1,
                name: `User ${i + 1}`,
            }));
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: manyRows, rowCount: 30 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-pagination')).toBeInTheDocument();
            });

            fireEvent.click(screen.getByTestId('db-pagination-next'));
            expect(screen.getByTestId('db-pagination-next')).toBeDisabled();
        });

        it('should not show pagination when data fits in one page', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 3 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-data-table')).toBeInTheDocument();
            });

            expect(screen.queryByTestId('db-pagination')).not.toBeInTheDocument();
        });
    });

    describe('Row Count and Column Statistics (Requirement 8.6)', () => {
        it('should display row count and column count', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 1500 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-stats')).toBeInTheDocument();
            });

            expect(screen.getByText('1,500 rows')).toBeInTheDocument();
            expect(screen.getByText('4 columns')).toBeInTheDocument();
        });
    });

    describe('Error State (Requirement 8.7)', () => {
        it('should display error when table loading fails', async () => {
            mockGetDataSourceTables.mockRejectedValue(new Error('Connection failed'));

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-error')).toBeInTheDocument();
            });

            expect(screen.getByText('Unable to load data source tables. Please try again.')).toBeInTheDocument();
        });

        it('should display error when table data loading fails', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockRejectedValue(new Error('Query failed'));

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-error')).toBeInTheDocument();
            });

            expect(screen.getByText('Unable to load data for table "users". Please try again.')).toBeInTheDocument();
        });
    });

    describe('Search/Filter (Requirement 8.8)', () => {
        it('should render search bar when sourceId is provided', () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            expect(screen.getByTestId('db-search-bar')).toBeInTheDocument();
            expect(screen.getByTestId('db-search-input')).toBeInTheDocument();
        });

        it('should filter tables based on search query', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            fireEvent.change(screen.getByTestId('db-search-input'), {
                target: { value: 'user' },
            });

            expect(screen.getByTestId('db-table-item-users')).toBeInTheDocument();
            expect(screen.queryByTestId('db-table-item-orders')).not.toBeInTheDocument();
            expect(screen.queryByTestId('db-table-item-products')).not.toBeInTheDocument();
        });

        it('should show no results message when search matches nothing', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            fireEvent.change(screen.getByTestId('db-search-input'), {
                target: { value: 'nonexistent' },
            });

            expect(screen.getByTestId('db-no-search-results')).toBeInTheDocument();
        });

        it('should filter columns when a table is selected', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-columns-list')).toBeInTheDocument();
            });

            // All 4 columns should be visible
            expect(screen.getByTestId('db-column-id')).toBeInTheDocument();
            expect(screen.getByTestId('db-column-name')).toBeInTheDocument();
            expect(screen.getByTestId('db-column-email')).toBeInTheDocument();
            expect(screen.getByTestId('db-column-age')).toBeInTheDocument();

            // Filter to only show 'name'
            fireEvent.change(screen.getByTestId('db-search-input'), {
                target: { value: 'name' },
            });

            expect(screen.getByTestId('db-column-name')).toBeInTheDocument();
            expect(screen.queryByTestId('db-column-email')).not.toBeInTheDocument();
        });

        it('should have correct placeholder text for table search vs column filter', async () => {
            mockGetDataSourceTables.mockResolvedValue(sampleTables);
            mockGetDataSourceTableDataWithCount.mockResolvedValue({ data: sampleTableData, rowCount: 100 });

            render(
                <DataBrowser {...defaultProps} isOpen={true} sourceId="src-1" />
            );

            // Table list view
            expect(screen.getByTestId('db-search-input')).toHaveAttribute(
                'placeholder',
                'Search tables...'
            );

            await waitFor(() => {
                expect(screen.getByTestId('db-table-list')).toBeInTheDocument();
            });

            // Select a table
            await act(async () => {
                fireEvent.click(screen.getByTestId('db-table-item-users'));
            });

            await waitFor(() => {
                expect(screen.getByTestId('db-search-input')).toHaveAttribute(
                    'placeholder',
                    'Filter columns...'
                );
            });
        });
    });

    describe('Absolute Positioning (Requirements 7.2, 7.3)', () => {
        it('should use absolute positioning within center panel', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel.className).toContain('data-browser');
        });

        it('should be positioned at right edge', () => {
            render(<DataBrowser {...defaultProps} isOpen={true} />);
            const panel = screen.getByTestId('data-browser');
            expect(panel.className).toContain('data-browser');
        });
    });

    describe('Utility Functions', () => {
        describe('inferColumnType', () => {
            it('should return "integer" for integer numbers', () => {
                expect(inferColumnType(42)).toBe('integer');
                expect(inferColumnType(0)).toBe('integer');
                expect(inferColumnType(-5)).toBe('integer');
            });

            it('should return "float" for decimal numbers', () => {
                expect(inferColumnType(3.14)).toBe('float');
                expect(inferColumnType(-0.5)).toBe('float');
            });

            it('should return "boolean" for booleans', () => {
                expect(inferColumnType(true)).toBe('boolean');
                expect(inferColumnType(false)).toBe('boolean');
            });

            it('should return "text" for regular strings', () => {
                expect(inferColumnType('hello')).toBe('text');
                expect(inferColumnType('abc123')).toBe('text');
            });

            it('should return "date" for date-like strings', () => {
                expect(inferColumnType('2024-01-15')).toBe('date');
                expect(inferColumnType('2024-01-15T10:30:00')).toBe('date');
            });

            it('should return "numeric" for numeric strings', () => {
                expect(inferColumnType('42')).toBe('numeric');
                expect(inferColumnType('-3.14')).toBe('numeric');
            });

            it('should return "unknown" for null/undefined', () => {
                expect(inferColumnType(null)).toBe('unknown');
                expect(inferColumnType(undefined)).toBe('unknown');
            });

            it('should return "json" for objects', () => {
                expect(inferColumnType({ key: 'value' })).toBe('json');
                expect(inferColumnType([1, 2, 3])).toBe('json');
            });
        });

        describe('formatCellValue', () => {
            it('should return dash for null/undefined', () => {
                expect(formatCellValue(null)).toBe('—');
                expect(formatCellValue(undefined)).toBe('—');
            });

            it('should stringify objects', () => {
                expect(formatCellValue({ a: 1 })).toBe('{"a":1}');
            });

            it('should convert values to string', () => {
                expect(formatCellValue(42)).toBe('42');
                expect(formatCellValue('hello')).toBe('hello');
                expect(formatCellValue(true)).toBe('true');
            });
        });
    });
});
