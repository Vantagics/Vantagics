import React from 'react';
import { ComponentType } from './PaginationControl';

export interface LayoutEditorProps {
  isLocked: boolean;
  onToggleLock: () => void;
  onAddComponent: (type: ComponentType) => void;
  onRemoveComponent: (id: string) => void;
  className?: string;
}

export const LayoutEditor: React.FC<LayoutEditorProps> = ({
  isLocked,
  onToggleLock,
  onAddComponent,
  onRemoveComponent,
  className = ''
}) => {
  const handleAddComponent = (type: ComponentType) => {
    onAddComponent(type);
  };

  return (
    <div 
      className={`layout-editor bg-white border-b border-gray-200 shadow-sm ${className}`}
      data-testid="layout-editor"
    >
      <div className="flex items-center justify-between px-4 py-3">
        {/* Left side - Lock/Unlock toggle */}
        <div className="flex items-center space-x-3">
          <button
            onClick={onToggleLock}
            className={`
              flex items-center space-x-2 px-4 py-2 rounded-lg font-medium transition-all duration-200
              ${isLocked 
                ? 'bg-red-100 text-red-700 hover:bg-red-200 border border-red-300' 
                : 'bg-green-100 text-green-700 hover:bg-green-200 border border-green-300'
              }
            `}
            data-testid="lock-toggle-button"
            aria-label={isLocked ? 'Unlock layout for editing' : 'Lock layout to prevent editing'}
          >
            {/* Lock/Unlock Icon */}
            <span className="text-lg" aria-hidden="true">
              {isLocked ? '🔒' : '🔓'}
            </span>
            <span>
              {isLocked ? 'Locked' : 'Editing'}
            </span>
          </button>

          {/* Visual lock state indicator */}
          <div className="flex items-center space-x-2">
            <div 
              className={`
                w-3 h-3 rounded-full transition-colors duration-200
                ${isLocked ? 'bg-red-500' : 'bg-green-500'}
              `}
              data-testid="lock-state-indicator"
              aria-hidden="true"
            />
            <span className="text-sm text-gray-600">
              {isLocked ? 'Dashboard is locked' : 'Dashboard is editable'}
            </span>
          </div>
        </div>

        {/* Right side - Add component buttons (only visible when unlocked) */}
        {!isLocked && (
          <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-600 mr-2">Add Component:</span>
            
            <button
              onClick={() => handleAddComponent(ComponentType.METRICS)}
              className="px-3 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 active:bg-blue-700 transition-colors text-sm font-medium"
              data-testid="add-metrics-button"
              aria-label="Add metrics component"
            >
              📊 Metrics
            </button>

            <button
              onClick={() => handleAddComponent(ComponentType.TABLE)}
              className="px-3 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 active:bg-blue-700 transition-colors text-sm font-medium"
              data-testid="add-table-button"
              aria-label="Add table component"
            >
              📋 Table
            </button>

            <button
              onClick={() => handleAddComponent(ComponentType.IMAGE)}
              className="px-3 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 active:bg-blue-700 transition-colors text-sm font-medium"
              data-testid="add-image-button"
              aria-label="Add image component"
            >
              🖼️ Image
            </button>

            <button
              onClick={() => handleAddComponent(ComponentType.INSIGHTS)}
              className="px-3 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 active:bg-blue-700 transition-colors text-sm font-medium"
              data-testid="add-insights-button"
              aria-label="Add insights component"
            >
              💡 Insights
            </button>

            <button
              onClick={() => handleAddComponent(ComponentType.FILE_DOWNLOAD)}
              className="px-3 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 active:bg-blue-700 transition-colors text-sm font-medium"
              data-testid="add-file-download-button"
              aria-label="Add file download component"
            >
              📁 Files
            </button>
          </div>
        )}
      </div>

      {/* Help text when in edit mode */}
      {!isLocked && (
        <div className="px-4 pb-3">
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
            <p className="text-sm text-blue-800">
              <span className="font-medium">Edit Mode:</span> Drag components to reposition, resize using corner handles, or add new components using the buttons above.
            </p>
          </div>
        </div>
      )}
    </div>
  );
};

export default LayoutEditor;