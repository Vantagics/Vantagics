import React from 'react';
import { DraggableComponent } from './DraggableComponent';
import SmartInsight from './SmartInsight';
import { ComponentInstance } from '../utils/ComponentManager';

export interface DraggableSmartInsightProps {
  instance: ComponentInstance;
  isEditMode: boolean;
  isLocked: boolean;
  onDragStart: (id: string) => void;
  onDrag: (id: string, x: number, y: number) => void;
  onDragStop: (id: string, x: number, y: number) => void;
  onResize: (id: string, width: number, height: number) => void;
  onResizeStop: (id: string, width: number, height: number) => void;
  onRemove?: (id: string) => void;
  onInsightClick?: (insightData: InsightData) => void;
}

export interface InsightData {
  text: string;
  icon: string;
  title?: string;
}

export const DraggableSmartInsight: React.FC<DraggableSmartInsightProps> = ({
  instance,
  isEditMode,
  isLocked,
  onDragStart,
  onDrag,
  onDragStop,
  onResize,
  onResizeStop,
  onRemove,
  onInsightClick
}) => {
  // Check if component has data
  const hasData = instance.hasData && instance.data && 
    instance.data.text && instance.data.text.trim().length > 0;
  
  // Don't render in locked mode if no data
  if (isLocked && !hasData) {
    return null;
  }

  // Render empty state in edit mode when no data
  const renderEmptyState = () => (
    <div className="w-full h-full bg-gray-50 border-2 border-dashed border-gray-300 rounded-xl flex flex-col items-center justify-center p-4 text-gray-500 min-h-[120px]">
      <div className="text-4xl mb-2">💡</div>
      <div className="text-sm font-medium text-center">
        Insights Component
      </div>
      <div className="text-xs text-center mt-1">
        No insights available
      </div>
      {isEditMode && onRemove && (
        <button
          onClick={() => onRemove(instance.id)}
          className="mt-3 px-3 py-1 bg-red-500 text-white text-xs rounded hover:bg-red-600 transition-colors"
          data-testid="remove-component-button"
        >
          Remove
        </button>
      )}
    </div>
  );

  // Render actual smart insight with data
  const renderSmartInsight = () => {
    const insightData = instance.data as InsightData;
    
    const handleInsightClick = () => {
      if (onInsightClick && !isEditMode) {
        onInsightClick(insightData);
      }
    };

    return (
      <div className="w-full h-full relative">
        {insightData.title && (
          <div className="px-4 py-2 border-b border-slate-100 bg-slate-50 rounded-t-xl">
            <h4 className="font-semibold text-sm text-slate-700">{insightData.title}</h4>
          </div>
        )}
        
        <div className="p-2">
          <SmartInsight
            text={insightData.text}
            icon={insightData.icon || 'info'}
            onClick={!isEditMode ? handleInsightClick : undefined}
          />
        </div>
        
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
      {hasData ? renderSmartInsight() : renderEmptyState()}
    </DraggableComponent>
  );
};

export default DraggableSmartInsight;