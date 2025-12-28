import React, { useEffect, useRef } from 'react';
import { Copy, ClipboardPaste, Scissors, Maximize } from 'lucide-react';

interface ContextMenuProps {
    position: { x: number; y: number };
    onClose: () => void;
    target: HTMLElement | null;
}

const ContextMenu: React.FC<ContextMenuProps> = ({ position, onClose, target }) => {
    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [onClose]);

    const handleAction = async (action: 'copy' | 'cut' | 'paste' | 'selectAll') => {
        if (!target || !(target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement)) return;

        target.focus();

        switch (action) {
            case 'copy':
                // Use execCommand for copy/cut as it works on selection reliably
                document.execCommand('copy');
                break;
            case 'cut':
                document.execCommand('cut');
                break;
            case 'paste':
                try {
                    const text = await navigator.clipboard.readText();
                    // Insert text at cursor position
                    const start = target.selectionStart || 0;
                    const end = target.selectionEnd || 0;
                    const value = target.value;
                    target.value = value.substring(0, start) + text + value.substring(end);
                    // Update cursor position
                    target.selectionStart = target.selectionEnd = start + text.length;
                    // Trigger change event so React state updates
                    target.dispatchEvent(new Event('input', { bubbles: true }));
                } catch (err) {
                    console.error('Failed to read clipboard:', err);
                    // Fallback to execCommand if available (often not for paste)
                    document.execCommand('paste');
                }
                break;
            case 'selectAll':
                target.select();
                break;
        }
        onClose();
    };

    return (
        <div 
            ref={menuRef}
            role="menu"
            className="fixed bg-white border border-slate-200 rounded-lg shadow-xl z-[9999] w-48 py-1 overflow-hidden"
            style={{ top: position.y, left: position.x }}
        >
            <button 
                onClick={() => handleAction('cut')}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Scissors className="w-4 h-4 text-slate-400" />
                Cut
            </button>
            <button 
                onClick={() => handleAction('copy')}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Copy className="w-4 h-4 text-slate-400" />
                Copy
            </button>
            <button 
                onClick={() => handleAction('paste')}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <ClipboardPaste className="w-4 h-4 text-slate-400" />
                Paste
            </button>
            <div className="h-px bg-slate-100 my-1" />
            <button 
                onClick={() => handleAction('selectAll')}
                className="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
            >
                <Maximize className="w-4 h-4 text-slate-400" />
                Select All
            </button>
        </div>
    );
};

export default ContextMenu;
