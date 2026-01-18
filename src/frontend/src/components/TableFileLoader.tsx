import React, { useState, useEffect } from 'react';
import { ReadChartDataFile } from '../../wailsjs/go/main/App';
import { Table } from 'lucide-react';

interface TableFileLoaderProps {
    fileRef: string;
    threadId: string | null;
}

const TableFileLoader: React.FC<TableFileLoaderProps> = ({ fileRef, threadId }) => {
    const [fileData, setFileData] = useState<any[] | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const loadFileData = async () => {
            try {
                setLoading(true);
                setError(null);

                if (!threadId) {
                    throw new Error('No active thread ID available');
                }

                console.log("[TableFileLoader] Loading table data from file:", fileRef, "for thread:", threadId);
                const data = await ReadChartDataFile(threadId, fileRef);
                console.log("[TableFileLoader] Successfully loaded table data from file, length:", data.length);

                // Parse the JSON data
                const tableData = JSON.parse(data);
                setFileData(tableData);
                setLoading(false);
            } catch (error) {
                console.error("[TableFileLoader] Failed to load table data file:", error);
                setError(error instanceof Error ? error.message : String(error));
                setLoading(false);
            }
        };

        loadFileData();
    }, [fileRef, threadId]);

    // Show loading state
    if (loading) {
        return (
            <div className="w-full bg-blue-50 border border-blue-200 rounded-xl p-4 shadow-sm">
                <div className="flex items-center gap-3">
                    <div className="animate-spin">
                        <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                    </div>
                    <div className="flex-1">
                        <p className="text-sm font-medium text-blue-800">Loading Table Data...</p>
                        <p className="text-xs text-blue-600 mt-1">Reading table data from file</p>
                    </div>
                </div>
            </div>
        );
    }

    // Show error state
    if (error) {
        return (
            <div className="w-full bg-red-50 border border-red-200 rounded-xl p-4 shadow-sm">
                <div className="flex items-center gap-3">
                    <div className="bg-red-100 p-2 rounded-lg">
                        <svg className="w-5 h-5 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                        </svg>
                    </div>
                    <div className="flex-1">
                        <p className="text-sm font-medium text-red-800">Failed to Load Table Data</p>
                        <p className="text-xs text-red-600 mt-1">{error}</p>
                    </div>
                </div>
            </div>
        );
    }

    // Render table with loaded data
    if (!fileData || !Array.isArray(fileData) || fileData.length === 0) {
        return null;
    }

    const columns = Object.keys(fileData[0]);

    return (
        <div className="w-full bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50">
                <div className="flex items-center gap-2">
                    <Table className="w-4 h-4 text-blue-500" />
                    <span className="text-sm font-medium text-slate-700">Analysis Result</span>
                    <span className="text-xs text-slate-400">({fileData.length} rows)</span>
                </div>
            </div>
            <div className="overflow-x-auto max-h-96">
                <table className="w-full">
                    <thead className="bg-slate-50 sticky top-0">
                        <tr>
                            {columns.map((col, idx) => (
                                <th
                                    key={idx}
                                    className="px-4 py-2 text-left text-xs font-medium text-slate-600 uppercase tracking-wider border-b border-slate-200"
                                >
                                    {col}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                        {fileData.map((row, rowIdx) => (
                            <tr key={rowIdx} className="hover:bg-slate-50 transition">
                                {columns.map((col, colIdx) => (
                                    <td
                                        key={colIdx}
                                        className="px-4 py-3 text-sm text-slate-700 whitespace-nowrap"
                                    >
                                        {typeof row[col] === 'object'
                                            ? JSON.stringify(row[col])
                                            : String(row[col])}
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default TableFileLoader;
