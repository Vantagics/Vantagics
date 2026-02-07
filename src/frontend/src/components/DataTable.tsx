import React from 'react';

interface DataTableProps {
    data: any[];
    title?: string;
}

const DataTable: React.FC<DataTableProps> = ({ data, title }) => {
    if (!data || data.length === 0) return null;

    // Extract columns from first item
    const columns = Object.keys(data[0]);

    return (
        <div className="w-full bg-white overflow-hidden flex flex-col">
            {title && (
                <div className="px-4 py-3 border-b border-slate-100 bg-slate-50">
                    <h4 className="font-semibold text-sm text-slate-700">{title}</h4>
                </div>
            )}
            <div className="overflow-x-auto">
                <table className="w-full text-sm text-left text-slate-600">
                    <thead className="text-xs text-slate-500 uppercase bg-slate-50 border-b border-slate-100">
                        <tr>
                            {columns.map((col) => (
                                <th key={col} className="px-6 py-3 font-medium whitespace-nowrap">
                                    {col}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {data.map((row, idx) => (
                            <tr key={idx} className="bg-white border-b border-slate-50 hover:bg-slate-50/50">
                                {columns.map((col) => (
                                    <td key={`${idx}-${col}`} className="px-6 py-4 whitespace-nowrap">
                                        {row[col] !== null && row[col] !== undefined ? String(row[col]) : <span className="text-slate-300">NULL</span>}
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
            <div className="px-4 py-2 bg-slate-50 border-t border-slate-100 text-xs text-slate-400 text-right">
                {data.length} rows
            </div>
        </div>
    );
};

export default DataTable;
