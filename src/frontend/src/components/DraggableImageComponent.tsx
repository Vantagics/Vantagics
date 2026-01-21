import React, { useState, useEffect } from 'react';
import DraggableComponent from './DraggableComponent';
import ImageModal from './ImageModal';
import { ComponentInstance } from '../utils/ComponentManager';
import { GetSessionFileAsBase64 } from '../../wailsjs/go/main/App';
import { convertImageData } from '../utils/ImageConverter';

export interface DraggableImageComponentProps {
  instance: ComponentInstance;
  isEditMode: boolean;
  isLocked: boolean;
  onDragStart: (id: string) => void;
  onDrag: (id: string, x: number, y: number) => void;
  onDragStop: (id: string, x: number, y: number) => void;
  onResize: (id: string, width: number, height: number) => void;
  onResizeStop: (id: string, width: number, height: number) => void;
  onRemove?: (id: string) => void;
  threadId?: string;
}

export interface ImageData {
  src: string;
  alt?: string;
  title?: string;
}

export const DraggableImageComponent: React.FC<DraggableImageComponentProps> = ({
  instance,
  isEditMode,
  isLocked,
  onDragStart,
  onDrag,
  onDragStop,
  onResize,
  onResizeStop,
  onRemove,
  threadId
}) => {
  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Check if component has data
  const hasData = instance.hasData && instance.data && instance.data.src;
  
  // Don't render in locked mode if no data
  if (isLocked && !hasData) {
    return null;
  }

  const imageData = instance.data as ImageData;

  // Load image data
  useEffect(() => {
    if (!hasData || !imageData.src) {
      setLoading(false);
      return;
    }

    const loadImage = async () => {
      try {
        setLoading(true);
        setError(false);

        // Use the centralized image conversion function
        const convertedImageData = await convertImageData(
          imageData.src,
          threadId,
          GetSessionFileAsBase64
        );

        setImageSrc(convertedImageData);
        setLoading(false);
      } catch (err) {
        console.error('[DraggableImageComponent] Failed to load image:', err);
        setError(true);
        setLoading(false);
      }
    };

    loadImage();
  }, [hasData, imageData?.src, threadId]);

  // Render empty state in edit mode when no data
  const renderEmptyState = () => (
    <div className="w-full h-full bg-gray-50 border-2 border-dashed border-gray-300 rounded-xl flex flex-col items-center justify-center p-8 text-gray-500 min-h-[600px]">
      <div className="text-4xl mb-3">🖼️</div>
      <div className="text-sm font-medium text-center">
        Image Component
      </div>
      <div className="text-xs text-center mt-1">
        No image available
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

  // Render loading state
  const renderLoadingState = () => (
    <div className="w-full h-full bg-gray-50 rounded-xl flex flex-col items-center justify-center p-8 text-gray-500 min-h-[600px]">
      <div className="animate-spin text-3xl mb-3">⏳</div>
      <div className="text-sm">Loading image...</div>
    </div>
  );

  // Render error state
  const renderErrorState = () => (
    <div className="w-full h-full bg-blue-50 border-2 border-dashed border-blue-300 rounded-xl flex flex-col items-center justify-center p-8 text-blue-500 min-h-[600px]">
      <div className="text-4xl mb-3">❌</div>
      <div className="text-sm font-medium text-center">
        Failed to load image
      </div>
      <div className="text-xs text-center mt-1 break-all px-4">
        {imageData?.src}
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

  // Render actual image
  const renderImage = () => (
    <div className="w-full h-full relative">
      <div className="w-full h-full bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        {imageData?.title && (
          <div className="px-4 py-2 border-b border-slate-100 bg-slate-50">
            <h4 className="font-semibold text-sm text-slate-700">{imageData.title}</h4>
          </div>
        )}
        <div className="p-4 flex items-center justify-center h-full">
          <img
            src={imageSrc!}
            alt={imageData?.alt || 'Dashboard image'}
            className="max-w-full max-h-full object-contain cursor-pointer hover:opacity-90 transition-opacity"
            onClick={() => setIsModalOpen(true)}
            data-testid="dashboard-image"
          />
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

      {/* Image Modal */}
      <ImageModal
        isOpen={isModalOpen}
        imageUrl={imageSrc!}
        onClose={() => setIsModalOpen(false)}
      />
    </div>
  );

  // Determine what to render
  const renderContent = () => {
    if (!hasData) {
      return renderEmptyState();
    }
    if (loading) {
      return renderLoadingState();
    }
    if (error || !imageSrc) {
      return renderErrorState();
    }
    return renderImage();
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
      {renderContent()}
    </DraggableComponent>
  );
};

export default DraggableImageComponent;