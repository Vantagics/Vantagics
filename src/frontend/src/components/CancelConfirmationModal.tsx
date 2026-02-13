import React from 'react';
import { createPortal } from 'react-dom';
import { AlertTriangle } from 'lucide-react';

interface CancelConfirmationModalProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
}

const CancelConfirmationModal: React.FC<CancelConfirmationModalProps> = ({ isOpen, onClose, onConfirm }) => {
    if (!isOpen) return null;

    return createPortal(
        <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="bg-white dark:bg-[#252526] rounded-xl shadow-2xl p-6 w-[380px] transform transition-all animate-in zoom-in-95 duration-200">
                <div className="flex items-start gap-4 mb-4">
                    <div className="bg-amber-100 dark:bg-[#3d3830] p-2 rounded-full">
                        <AlertTriangle className="w-6 h-6 text-amber-600 dark:text-[#dcdcaa]" />
                    </div>
                    <div className="flex-1">
                        <h3 className="text-lg font-bold text-slate-900 dark:text-[#d4d4d4] mb-1">取消分析</h3>
                        <p className="text-sm text-slate-600 dark:text-[#9d9d9d]">
                            确定要取消当前的分析任务吗？
                        </p>
                        <p className="text-xs text-slate-400 dark:text-[#808080] mt-2">
                            已经生成的结果将会丢失。
                        </p>
                    </div>
                </div>

                <div className="flex justify-end gap-3 mt-6">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-[#d4d4d4] hover:bg-slate-100 dark:hover:bg-[#2d2d30] rounded-lg transition-colors"
                    >
                        继续分析
                    </button>
                    <button
                        onClick={onConfirm}
                        className="px-4 py-2 text-sm font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg shadow-sm transition-colors"
                    >
                        确认取消
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default CancelConfirmationModal;
