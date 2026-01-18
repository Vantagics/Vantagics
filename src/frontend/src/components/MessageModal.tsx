import React, { useEffect, useState } from 'react';
import { X, AlertTriangle, CheckCircle, Info } from 'lucide-react';

interface MessageModalProps {
    isOpen: boolean;
    type: 'info' | 'warning' | 'error';
    title: string;
    message: string;
    onClose: () => void;
}

const MessageModal: React.FC<MessageModalProps> = ({ isOpen, type, title, message, onClose }) => {
    const [isVisible, setIsVisible] = useState(false);

    useEffect(() => {
        if (isOpen) {
            setIsVisible(true);
        } else {
            const timer = setTimeout(() => setIsVisible(false), 200);
            return () => clearTimeout(timer);
        }
    }, [isOpen]);

    if (!isVisible) return null;

    let Icon = Info;
    let colorClass = 'text-blue-500 bg-blue-50';
    
    if (type === 'warning') {
        Icon = AlertTriangle;
        colorClass = 'text-yellow-500 bg-yellow-50';
    } else if (type === 'error') {
        Icon = AlertTriangle;
        colorClass = 'text-red-500 bg-red-50';
    } else if (type === 'info') {
        Icon = CheckCircle;
        colorClass = 'text-green-500 bg-green-50';
    }

    return (
        <div 
            className={`fixed inset-0 z-[100] flex items-center justify-center p-4 transition-opacity duration-200 ${isOpen ? 'opacity-100' : 'opacity-0'}`}
        >
            <div 
                className="absolute inset-0 bg-slate-900/20 backdrop-blur-[1px]"
                onClick={onClose}
            />
            <div 
                className={`bg-white rounded-2xl shadow-2xl w-full max-w-sm transform transition-all duration-200 p-6 flex flex-col items-center text-center ${isOpen ? 'scale-100 translate-y-0' : 'scale-95 translate-y-4'}`}
            >
                <div className={`w-12 h-12 rounded-full flex items-center justify-center mb-4 ${colorClass}`}>
                    <Icon className="w-6 h-6" />
                </div>
                
                <h3 className="text-lg font-bold text-slate-800 mb-2">{title}</h3>
                <p className="text-sm text-slate-500 mb-6 leading-relaxed">
                    {message}
                </p>

                <button
                    onClick={onClose}
                    className="w-full bg-slate-900 hover:bg-slate-800 text-white font-medium py-2.5 rounded-xl transition-colors active:scale-95"
                >
                    Close
                </button>
            </div>
        </div>
    );
};

export default MessageModal;
