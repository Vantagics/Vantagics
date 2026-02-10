import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { describe, it, expect, vi } from 'vitest';
import DataSourcesSection, {
    DataSourcesSectionProps,
    DataSourceItem,
    getDataSourceIcon,
    getDataSourceTypeClass,
} from './DataSourcesSection';

describe('DataSourcesSection Component', () => {
    const defaultProps: DataSourcesSectionProps = {
        dataSources: [],
        selectedId: null,
        onSelect: vi.fn(),
        onContextMenu: vi.fn(),
        onAdd: vi.fn(),
    };

    const mockDataSources: DataSourceItem[] = [
        { id: 'ds1', name: 'Sales Database', type: 'mysql' },
        { id: 'ds2', name: 'Revenue Report', type: 'excel' },
        { id: 'ds3', name: 'User Data', type: 'postgresql' },
        { id: 'ds4', name: 'Config File', type: 'json' },
    ];

    describe('Header rendering', () => {
        it('should render the "Data Sources" title', () => {
            render(<DataSourcesSection {...defaultProps} />);
            expect(screen.getByText('Data Sources')).toBeInTheDocument();
        });

        it('should render the "Add Data Source" button with + text', () => {
            render(<DataSourcesSection {...defaultProps} />);
            const addButton = screen.getByRole('button', { name: 'Add Data Source' });
            expect(addButton).toBeInTheDocument();
            expect(addButton).toHaveTextContent('+');
        });

        it('should call onAdd when the add button is clicked', () => {
            const onAdd = vi.fn();
            render(<DataSourcesSection {...defaultProps} onAdd={onAdd} />);
            fireEvent.click(screen.getByRole('button', { name: 'Add Data Source' }));
            expect(onAdd).toHaveBeenCalledTimes(1);
        });
    });

    describe('Empty state', () => {
        it('should display empty state when no data sources exist', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={[]} />);
            expect(screen.getByText(/No data sources available/)).toBeInTheDocument();
        });

        it('should not render a list when no data sources exist', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={[]} />);
            expect(screen.queryByRole('list')).not.toBeInTheDocument();
        });
    });

    describe('Data sources list rendering', () => {
        it('should render all data sources as list items', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            expect(screen.getByText('Sales Database')).toBeInTheDocument();
            expect(screen.getByText('Revenue Report')).toBeInTheDocument();
            expect(screen.getByText('User Data')).toBeInTheDocument();
            expect(screen.getByText('Config File')).toBeInTheDocument();
        });

        it('should render exactly one list item per data source', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            const listItems = screen.getAllByRole('listitem');
            expect(listItems).toHaveLength(mockDataSources.length);
        });

        it('should render a list container with role="list"', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            expect(screen.getByRole('list')).toBeInTheDocument();
        });

        it('should not display empty state when data sources exist', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            expect(screen.queryByText(/No data sources available/)).not.toBeInTheDocument();
        });
    });

    describe('Data source selection', () => {
        it('should call onSelect with the correct id when a data source is clicked', () => {
            const onSelect = vi.fn();
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    onSelect={onSelect}
                />
            );
            fireEvent.click(screen.getByText('Sales Database'));
            expect(onSelect).toHaveBeenCalledWith('ds1');
        });

        it('should highlight the selected data source with "selected" class', () => {
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    selectedId="ds2"
                />
            );
            const selectedItem = screen.getByText('Revenue Report').closest('.data-source-item');
            expect(selectedItem).toHaveClass('selected');
        });

        it('should not highlight unselected data sources', () => {
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    selectedId="ds2"
                />
            );
            const unselectedItem = screen.getByText('Sales Database').closest('.data-source-item');
            expect(unselectedItem).not.toHaveClass('selected');
        });

        it('should set aria-selected on the selected item', () => {
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    selectedId="ds1"
                />
            );
            const selectedItem = screen.getByText('Sales Database').closest('[role="listitem"]');
            expect(selectedItem).toHaveAttribute('aria-selected', 'true');
        });

        it('should set aria-selected=false on unselected items', () => {
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    selectedId="ds1"
                />
            );
            const unselectedItem = screen.getByText('Revenue Report').closest('[role="listitem"]');
            expect(unselectedItem).toHaveAttribute('aria-selected', 'false');
        });
    });

    describe('Context menu', () => {
        it('should call onContextMenu with event and id on right-click', () => {
            const onContextMenu = vi.fn();
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    onContextMenu={onContextMenu}
                />
            );
            fireEvent.contextMenu(screen.getByText('Sales Database'));
            expect(onContextMenu).toHaveBeenCalledTimes(1);
            expect(onContextMenu).toHaveBeenCalledWith(expect.any(Object), 'ds1');
        });

        it('should pass the correct source id for each data source on right-click', () => {
            const onContextMenu = vi.fn();
            render(
                <DataSourcesSection
                    {...defaultProps}
                    dataSources={mockDataSources}
                    onContextMenu={onContextMenu}
                />
            );
            fireEvent.contextMenu(screen.getByText('User Data'));
            expect(onContextMenu).toHaveBeenCalledWith(expect.any(Object), 'ds3');
        });
    });

    describe('Type indicators', () => {
        it('should display type indicator icons for each data source', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            // Each data source item should have a source-icon span
            const icons = document.querySelectorAll('.source-icon');
            expect(icons).toHaveLength(mockDataSources.length);
        });

        it('should add type-specific CSS class to each data source item', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            const mysqlItem = screen.getByText('Sales Database').closest('.data-source-item');
            expect(mysqlItem).toHaveClass('source-type-mysql');

            const excelItem = screen.getByText('Revenue Report').closest('.data-source-item');
            expect(excelItem).toHaveClass('source-type-excel');
        });

        it('should set data-source-type attribute on each item', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={mockDataSources} />);
            const mysqlItem = screen.getByText('Sales Database').closest('[data-source-type]');
            expect(mysqlItem).toHaveAttribute('data-source-type', 'mysql');
        });
    });

    describe('Accessibility', () => {
        it('should have a region role with "Data Sources" label', () => {
            render(<DataSourcesSection {...defaultProps} />);
            expect(screen.getByRole('region', { name: 'Data Sources' })).toBeInTheDocument();
        });

        it('should have aria-label on the add button', () => {
            render(<DataSourcesSection {...defaultProps} />);
            const addButton = screen.getByRole('button', { name: 'Add Data Source' });
            expect(addButton).toHaveAttribute('aria-label', 'Add Data Source');
        });

        it('should have role="status" on empty state', () => {
            render(<DataSourcesSection {...defaultProps} dataSources={[]} />);
            expect(screen.getByRole('status')).toBeInTheDocument();
        });
    });
});

describe('getDataSourceIcon', () => {
    it('should return correct icon for known types', () => {
        expect(getDataSourceIcon('excel')).toBe('📊');
        expect(getDataSourceIcon('mysql')).toBe('🗄️');
        expect(getDataSourceIcon('postgresql')).toBe('🐘');
        expect(getDataSourceIcon('doris')).toBe('💾');
        expect(getDataSourceIcon('csv')).toBe('📄');
        expect(getDataSourceIcon('json')).toBe('📋');
    });

    it('should return default icon for unknown types', () => {
        expect(getDataSourceIcon('unknown')).toBe('📁');
        expect(getDataSourceIcon('sqlite')).toBe('📁');
    });

    it('should be case-insensitive', () => {
        expect(getDataSourceIcon('EXCEL')).toBe('📊');
        expect(getDataSourceIcon('MySQL')).toBe('🗄️');
        expect(getDataSourceIcon('PostgreSQL')).toBe('🐘');
    });
});

describe('getDataSourceTypeClass', () => {
    it('should return correct CSS class for each type', () => {
        expect(getDataSourceTypeClass('mysql')).toBe('source-type-mysql');
        expect(getDataSourceTypeClass('excel')).toBe('source-type-excel');
        expect(getDataSourceTypeClass('postgresql')).toBe('source-type-postgresql');
    });

    it('should lowercase the type in the class name', () => {
        expect(getDataSourceTypeClass('MySQL')).toBe('source-type-mysql');
        expect(getDataSourceTypeClass('EXCEL')).toBe('source-type-excel');
    });
});
