import React from 'react';
import { useLanguage } from '../i18n';

/**
 * Data source item shape used by DataSourcesSection.
 * Compatible with agent.DataSource from the Wails models.
 * Requirements: 2.1, 2.2, 2.5
 */
export interface DataSourceItem {
    id: string;
    name: string;
    type: string;
}

/**
 * Props interface for DataSourcesSection component
 * Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7
 */
export interface DataSourcesSectionProps {
    dataSources: DataSourceItem[];
    selectedId: string | null;
    onSelect: (id: string) => void;
    onContextMenu: (e: React.MouseEvent, id: string) => void;
    onAdd: () => void;
}

/**
 * Get icon for data source type
 * Requirements: 2.5 - Display data source type indicators (icons or colors)
 */
export function getDataSourceIcon(type: string): string {
    const icons: Record<string, string> = {
        'excel': '📊',
        'mysql': '🗄️',
        'postgresql': '🐘',
        'doris': '💾',
        'csv': '📄',
        'json': '📋',
    };
    return icons[type.toLowerCase()] || '📁';
}

/**
 * Get CSS class for data source type indicator
 * Requirements: 2.5 - Display data source type indicators (icons or colors)
 */
export function getDataSourceTypeClass(type: string): string {
    return `source-type-${type.toLowerCase()}`;
}

/**
 * DataSourcesSection component
 *
 * Displays the list of data sources with an "Add Data Source" button in the header.
 * Supports selection, right-click context menu, type indicators, and empty state.
 *
 * Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7
 * Validates: Property 4 (List rendering completeness), Property 6 (Data source type indicators)
 */
const DataSourcesSection: React.FC<DataSourcesSectionProps> = ({
    dataSources,
    selectedId,
    onSelect,
    onContextMenu,
    onAdd,
}) => {
    const { t } = useLanguage();
    return (
        <div className="data-sources-section" role="region" aria-label={t('data_sources')}>
            {/* Header with title and add button - Requirements: 2.1, 2.7 */}
            <div className="section-header">
                <h3>{t('data_sources')}</h3>
                <button
                    className="add-button"
                    onClick={onAdd}
                    aria-label={t('add_data_source')}
                    title={t('add_data_source')}
                >
                    +
                </button>
            </div>

            {/* Data sources list or empty state */}
            {dataSources.length === 0 ? (
                /* Empty state - Requirements: 2.6 */
                <div className="empty-state" role="status">
                    {t('no_data_sources_available')}
                </div>
            ) : (
                /* Scrollable list - Requirements: 2.2 */
                <div className="data-sources-list" role="list">
                    {dataSources.map((source) => (
                        <div
                            key={source.id}
                            className={`data-source-item ${selectedId === source.id ? 'selected' : ''} ${getDataSourceTypeClass(source.type)}`}
                            onClick={() => onSelect(source.id)}
                            onContextMenu={(e) => onContextMenu(e, source.id)}
                            role="listitem"
                            aria-selected={selectedId === source.id}
                            data-source-id={source.id}
                            data-source-type={source.type}
                        >
                            {/* Type indicator icon - Requirements: 2.5 */}
                            <span className="source-icon" aria-label={t('data_source_type_label', source.type)}>
                                {getDataSourceIcon(source.type)}
                            </span>
                            <span className="source-name">{source.name}</span>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default DataSourcesSection;
