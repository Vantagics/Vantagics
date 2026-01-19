import React from 'react';

export enum ComponentType {
  METRICS = 'metrics',
  TABLE = 'table',
  IMAGE = 'image',
  INSIGHTS = 'insights',
  FILE_DOWNLOAD = 'file_download'
}

export interface PaginationControlProps {
  componentType: ComponentType;
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  visible: boolean;
  className?: string;
}

export const PaginationControl: React.FC<PaginationControlProps> = ({
  componentType,
  currentPage,
  totalPages,
  onPageChange,
  visible,
  className = ''
}) => {
  // Don't render if not visible or only one page
  if (!visible || totalPages <= 1) {
    return null;
  }

  const handlePrevious = () => {
    if (currentPage > 0) {
      onPageChange(currentPage - 1);
    }
  };

  const handleNext = () => {
    if (currentPage < totalPages - 1) {
      onPageChange(currentPage + 1);
    }
  };

  const handlePageClick = (page: number) => {
    if (page >= 0 && page < totalPages) {
      onPageChange(page);
    }
  };

  return (
    <div 
      className={`pagination-control flex items-center justify-center space-x-2 py-2 ${className}`}
      data-testid={`pagination-${componentType}`}
    >
      {/* Previous button */}
      <button
        onClick={handlePrevious}
        disabled={currentPage === 0}
        className={`
          px-3 py-1 rounded text-sm font-medium transition-colors
          ${currentPage === 0 
            ? 'bg-gray-100 text-gray-400 cursor-not-allowed' 
            : 'bg-blue-500 text-white hover:bg-blue-600 active:bg-blue-700'
          }
        `}
        data-testid="pagination-previous"
        aria-label="Previous page"
      >
        ←
      </button>

      {/* Page indicators */}
      <div className="flex items-center space-x-1">
        {Array.from({ length: totalPages }, (_, index) => (
          <button
            key={index}
            onClick={() => handlePageClick(index)}
            className={`
              w-8 h-8 rounded text-sm font-medium transition-colors
              ${index === currentPage
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200 active:bg-gray-300'
              }
            `}
            data-testid={`pagination-page-${index}`}
            aria-label={`Go to page ${index + 1}`}
            aria-current={index === currentPage ? 'page' : undefined}
          >
            {index + 1}
          </button>
        ))}
      </div>

      {/* Next button */}
      <button
        onClick={handleNext}
        disabled={currentPage === totalPages - 1}
        className={`
          px-3 py-1 rounded text-sm font-medium transition-colors
          ${currentPage === totalPages - 1
            ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
            : 'bg-blue-500 text-white hover:bg-blue-600 active:bg-blue-700'
          }
        `}
        data-testid="pagination-next"
        aria-label="Next page"
      >
        →
      </button>

      {/* Page info */}
      <span className="text-sm text-gray-600 ml-2">
        {currentPage + 1} of {totalPages}
      </span>
    </div>
  );
};

export default PaginationControl;