import React from 'react';
import { useLanguage } from '../i18n';

interface DataTableProps {
    data: any[];
    title?: string;
}

// Render cell content with basic markdown support (bold text)
const renderCellContent = (value: any): React.ReactNode => {
    if (value === null || value === undefined) {
        return null;
    }
    
    const str = String(value);
    
    // Check if the string contains markdown bold markers
    if (str.includes('**')) {
        // Split by ** and render bold parts
        const parts = str.split(/(\*\*[^*]+\*\*)/g);
        return (
            <>
                {parts.map((part, idx) => {
                    if (part.startsWith('**') && part.endsWith('**')) {
                        // Remove ** markers and render as bold
                        const boldText = part.slice(2, -2);
                        return <strong key={idx} className="font-semibold">{boldText}</strong>;
                    }
                    return <span key={idx}>{part}</span>;
                })}
            </>
        );
    }
    
    return str;
};

const DataTable: React.FC<DataTableProps> = ({ data, title }) => {
    const { t } = useLanguage();
    if (!data || data.length === 0) return null;

    // Extract columns from first item
    const columns = Object.keys(data[0]);
    
    // Limit displayed rows to prevent UI freeze with large datasets
    const MAX_DISPLAY_ROWS = 200;
    const displayData = data.length > MAX_DISPLAY_ROWS ? data.slice(0, MAX_DISPLAY_ROWS) : data;
    const isTruncated = data.length > MAX_DISPLAY_ROWS;

    return (
        <div className="w-full bg-white dark:bg-[#252526] overflow-hidden flex flex-col">
            {title && (
                <div className="px-4 py-3 border-b border-slate-100 dark:border-[#3c3c3c] bg-slate-50 dark:bg-[#2d2d30]">
                    <h4 className="font-semibold text-sm text-slate-700 dark:text-[#d4d4d4]">{renderCellContent(title)}</h4>
                </div>
            )}
            <div className="overflow-x-auto">
                <table className="w-full text-sm text-left text-slate-600 dark:text-[#d4d4d4]">
                    <thead className="text-xs text-slate-500 dark:text-[#808080] uppercase bg-slate-50 dark:bg-[#2d2d30] border-b border-slate-100 dark:border-[#3c3c3c]">
                        <tr>
                            {columns.map((col) => (
                                <th key={col} className="px-6 py-3 font-medium whitespace-nowrap">
                                    {renderCellContent(col)}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {displayData.map((row, idx) => (
                            <tr key={idx} className="bg-white dark:bg-[#252526] border-b border-slate-50 dark:border-[#2d2d30] hover:bg-slate-50/50 dark:hover:bg-[#2d2d30]">
                                {columns.map((col) => (
                                    <td key={`${idx}-${col}`} className="px-6 py-4 whitespace-nowrap">
                                        {row[col] !== null && row[col] !== undefined 
                                            ? renderCellContent(row[col]) 
                                            : <span className="text-slate-300 dark:text-[#4d4d4d]">{t('null_value')}</span>}
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
            <div className="px-4 py-2 bg-slate-50 dark:bg-[#2d2d30] border-t border-slate-100 dark:border-[#3c3c3c] text-xs text-slate-400 dark:text-[#808080] text-right">
                {isTruncated 
                    ? t('rows_stats', String(MAX_DISPLAY_ROWS), String(data.length), String(MAX_DISPLAY_ROWS))
                    : t('rows_total', String(data.length))
                }
            </div>
        </div>
    );
};

export default DataTable;
