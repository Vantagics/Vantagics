import React from 'react';
import DraggableComponent from './DraggableComponent';
import DataTable from './DataTable';
import { ComponentInstance } from '../utils/ComponentManager';

export interface DraggableDataTableProps {
  instance: ComponentInstance;
  isEditMode: boolean;
  isLocked: boolean;
  onDragStart: (id: string) => void;
  onDrag: (id: string, x: number, y: number) => void;
  onDragStop: (id: string, x: number, y: number) => void;
  onResize: (id: string, width: number, height: number) => void;
  onResizeStop: (id: string, width: number, height: number) => void;
  onRemove?: (id: string) => void;
}

export interface TableData {
  data: any[];
  title?: string;
}

export const DraggableDataTable: React.FC<DraggableDataTableProps> = ({
  instance,
  isEditMode,
  isLocked,
  onDragStart,
  onDrag,
  onDragStop,
  onResize,
  onResizeStop,
  onRemove
}) => {
  // Check if component has data
  const hasData = instance.hasData && instance.data && 
    Array.isArray(instance.data.data) && instance.data.data.length > 0;
  
  // Don't render in locked mode if no data
  if (isLocked && !hasData) {
    return null;
  }

  // Render empty state in edit mode when no data
  const renderEmptyState = () => (
    <div className="w-full h-full bg-gray-50 border-2 border-dashed border-gray-300 rounded-xl flex flex-col items-center justify-center p-8 text-gray-500 min-h-[600px]">
      <div className="text-4xl mb-3">📋</div>
      <div className="text-sm font-medium text-center">
        Data Table Component
      </div>
      <div className="text-xs text-center mt-1">
        No data available
      </div>
      {isEditMode && onRemove && (
        <button
          onClick={() => onRemove(instance.id)}
          className="mt-4 px-4 py-2 bg-red-500 text-white text-xs rounded hover:bg-red-600 transition-colors"
          data-testid="remove-component-button"
        >
          Remove
        </button>
      )}
    </div>
  );

  // Render actual data table with data
  const renderDataTable = () => {
    const tableData = instance.data as TableData;
    return (
      <div className="w-full h-full relative">
        <DataTable
          data={tableData.data}
          title={tableData.title}
        />
        {isEditMode && onRemove && (
          <button
            onClick={() => onRemove(instance.id)}
            className="absolute top-2 right-2 w-6 h-6 bg-red-500 text-white text-xs rounded-full hover:bg-red-600 transition-colors flex items-center justify-center z-20"
            data-testid="remove-component-button"
            aria-label="Remove component"
          >
            ×
          </button>
        )}
      </div>
    );
  };

  return (
    <DraggableComponent
      instance={instance}
      isEditMode={isEditMode}
      isLocked={isLocked}
      onDragStart={onDragStart}
      onDrag={onDrag}
      onDragStop={onDragStop}
      onResize={onResize}
      onResizeStop={onResizeStop}
    >
      {hasData ? renderDataTable() : renderEmptyState()}
    </DraggableComponent>
  );
};

export default DraggableDataTable;