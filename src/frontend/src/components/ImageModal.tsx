import React, { useEffect, useState } from 'react';
import { X, ZoomIn, ZoomOut } from 'lucide-react';
import { useLanguage } from '../i18n';

interface ImageModalProps {
    isOpen: boolean;
    imageUrl: string;
    onClose: () => void;
}

const ImageModal: React.FC<ImageModalProps> = ({ isOpen, imageUrl, onClose }) => {
    const { t } = useLanguage();
    const [scale, setScale] = useState(1);
    const [isVisible, setIsVisible] = useState(false);

    useEffect(() => {
        if (isOpen) {
            setIsVisible(true);
            setScale(1); // Reset scale on open
        } else {
            const timer = setTimeout(() => setIsVisible(false), 200);
            return () => clearTimeout(timer);
        }
    }, [isOpen]);

    if (!isVisible) return null;

    const handleZoomIn = (e: React.MouseEvent) => {
        e.stopPropagation();
        setScale(prev => Math.min(prev + 0.5, 3));
    };

    const handleZoomOut = (e: React.MouseEvent) => {
        e.stopPropagation();
        setScale(prev => Math.max(prev - 0.5, 0.5));
    };

    return (
        <div 
            className={`fixed inset-0 z-[200] flex items-center justify-center transition-opacity duration-200 ${isOpen ? 'opacity-100' : 'opacity-0'}`}
            onClick={onClose}
        >
            {/* Backdrop */}
            <div className="absolute inset-0 bg-black/90 backdrop-blur-sm" />

            {/* Controls */}
            <div className="absolute top-4 right-4 flex gap-2 z-[210]">
                <button 
                    onClick={handleZoomOut}
                    className="p-2 bg-white/10 hover:bg-white/20 rounded-full text-white transition-colors"
                >
                    <ZoomOut className="w-6 h-6" />
                </button>
                <button 
                    onClick={handleZoomIn}
                    className="p-2 bg-white/10 hover:bg-white/20 rounded-full text-white transition-colors"
                >
                    <ZoomIn className="w-6 h-6" />
                </button>
                <button 
                    onClick={onClose}
                    className="p-2 bg-white/10 hover:bg-red-500/80 rounded-full text-white transition-colors ml-2"
                >
                    <X className="w-6 h-6" />
                </button>
            </div>

            {/* Image Container */}
            <div 
                className="relative z-[205] overflow-auto max-w-full max-h-full p-4 flex items-center justify-center"
                onClick={(e) => e.stopPropagation()}
            >
                <img 
                    src={imageUrl} 
                    alt={t('full_view')} 
                    className="max-w-[90vw] max-h-[90vh] object-contain transition-transform duration-200 ease-out shadow-2xl rounded-lg"
                    style={{ transform: `scale(${scale})` }}
                />
            </div>
        </div>
    );
};

export default ImageModal;
