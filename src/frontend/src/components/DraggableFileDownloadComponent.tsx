import React from 'react';
import { DraggableComponent } from './DraggableComponent';
import { ComponentInstance } from '../utils/ComponentManager';
import { Download, File, Calendar, HardDrive } from 'lucide-react';

export interface DraggableFileDownloadComponentProps {
  instance: ComponentInstance;
  isEditMode: boolean;
  isLocked: boolean;
  onDragStart: (id: string) => void;
  onDrag: (id: string, x: number, y: number) => void;
  onDragStop: (id: string, x: number, y: number) => void;
  onResize: (id: string, width: number, height: number) => void;
  onResizeStop: (id: string, width: number, height: number) => void;
  onRemove?: (id: string) => void;
  onFileDownload?: (fileId: string, fileName: string) => void;
}

export interface FileInfo {
  id: string;
  name: string;
  size: number;
  createdAt: Date;
  downloadUrl?: string;
}

export interface FileDownloadData {
  allFiles: FileInfo[];
  userRequestRelatedFiles: FileInfo[];
  title?: string;
}

export const DraggableFileDownloadComponent: React.FC<DraggableFileDownloadComponentProps> = ({
  instance,
  isEditMode,
  isLocked,
  onDragStart,
  onDrag,
  onDragStop,
  onResize,
  onResizeStop,
  onRemove,
  onFileDownload
}) => {
  // Check if component has data (at least one category has files)
  const fileData = instance.data as FileDownloadData;
  const hasAllFiles = fileData?.allFiles && fileData.allFiles.length > 0;
  const hasUserRequestFiles = fileData?.userRequestRelatedFiles && fileData.userRequestRelatedFiles.length > 0;
  const hasData = instance.hasData && (hasAllFiles || hasUserRequestFiles);
  
  // Don't render in locked mode if no data
  if (isLocked && !hasData) {
    return null;
  }

  // Format file size for display
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  // Format date for display
  const formatDate = (date: Date): string => {
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(new Date(date));
  };

  // Handle file download
  const handleFileDownload = (file: FileInfo) => {
    if (onFileDownload && !isEditMode) {
      onFileDownload(file.id, file.name);
    }
  };

  // Render file list for a category
  const renderFileList = (files: FileInfo[], categoryName: string, emptyMessage: string) => (
    <div className="mb-4">
      <h5 className="text-sm font-medium text-slate-700 mb-2 flex items-center">
        <HardDrive className="w-4 h-4 mr-2" />
        {categoryName}
      </h5>
      
      {files.length === 0 ? (
        <div className="text-xs text-slate-400 italic p-3 bg-slate-50 rounded border">
          {emptyMessage}
        </div>
      ) : (
        <div className="space-y-2">
          {files.map((file) => (
            <div
              key={file.id}
              className={`
                flex items-center justify-between p-3 bg-white border border-slate-200 rounded hover:bg-slate-50 transition-colors
                ${!isEditMode ? 'cursor-pointer hover:border-blue-300' : ''}
              `}
              onClick={() => handleFileDownload(file)}
              data-testid={`file-item-${file.id}`}
            >
              <div className="flex items-center flex-1 min-w-0">
                <File className="w-4 h-4 text-slate-400 mr-2 flex-shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium text-slate-700 truncate" title={file.name}>
                    {file.name}
                  </div>
                  <div className="text-xs text-slate-500 flex items-center mt-1">
                    <Calendar className="w-3 h-3 mr-1" />
                    {formatDate(file.createdAt)}
                    <span className="mx-2">•</span>
                    {formatFileSize(file.size)}
                  </div>
                </div>
              </div>
              
              {!isEditMode && (
                <Download 
                  className="w-4 h-4 text-blue-500 flex-shrink-0 ml-2" 
                  data-testid={`download-icon-${file.id}`}
                />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );

  // Render empty state in edit mode when no data
  const renderEmptyState = () => (
    <div className="w-full h-full bg-gray-50 border-2 border-dashed border-gray-300 rounded-xl flex flex-col items-center justify-center p-4 text-gray-500 min-h-[200px]">
      <div className="text-4xl mb-2">📁</div>
      <div className="text-sm font-medium text-center">
        File Download Component
      </div>
      <div className="text-xs text-center mt-1">
        No files available
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

  // Render actual file download component with data
  const renderFileDownloadComponent = () => (
    <div className="w-full h-full relative">
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        {/* Header */}
        <div className="px-4 py-3 border-b border-slate-100 bg-slate-50">
          <h4 className="font-semibold text-sm text-slate-700 flex items-center">
            <Download className="w-4 h-4 mr-2" />
            {fileData?.title || 'File Downloads'}
          </h4>
        </div>

        {/* Content */}
        <div className="p-4 max-h-96 overflow-y-auto">
          {/* All Files Category */}
          {renderFileList(
            fileData?.allFiles || [],
            'All Files',
            'No files available in this category'
          )}

          {/* User Request Related Files Category */}
          {renderFileList(
            fileData?.userRequestRelatedFiles || [],
            'User Request Related Files',
            'No user request related files available'
          )}
        </div>

        {/* Footer with file count */}
        <div className="px-4 py-2 bg-slate-50 border-t border-slate-100 text-xs text-slate-500">
          Total: {(fileData?.allFiles?.length || 0) + (fileData?.userRequestRelatedFiles?.length || 0)} files
        </div>
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
      {hasData ? renderFileDownloadComponent() : renderEmptyState()}
    </DraggableComponent>
  );
};

export default DraggableFileDownloadComponent;