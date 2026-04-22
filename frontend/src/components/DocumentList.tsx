import { useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

export interface DocInfo {
  id: number;
  filename: string;
  summary: string;
  created_at: string;
}

interface Props {
  documents: DocInfo[];
  activeId: number | null;
  onSelect: (id: number) => void;
  open: boolean;
  onToggle: () => void;
}

export default function DocumentList({ documents, activeId, onSelect, open, onToggle }: Props) {
  const activeDoc = documents.find((d) => d.id === activeId);
  const ref = useRef<HTMLDivElement>(null);

  // Click-outside to close
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onToggle();
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open, onToggle]);

  return (
    <div className="relative" ref={ref}>
      <button onClick={onToggle} className="flex items-center gap-1.5 text-[11px] text-neutral-400 hover:text-neutral-200 transition-colors">
        <span>📄</span>
        <span className="max-w-[140px] truncate">
          {activeDoc ? activeDoc.filename : `${documents.length} doc${documents.length !== 1 ? 's' : ''}`}
        </span>
        <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" className={`transition-transform ${open ? 'rotate-180' : ''}`}>
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>

      <AnimatePresence>
        {open && (
          <motion.div initial={{ opacity: 0, y: -4 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -4 }}
            className="absolute top-full left-0 mt-1 w-64 bg-neutral-900 border border-neutral-800 rounded-lg shadow-xl z-20 overflow-hidden">
            <div className="text-[10px] text-neutral-500 px-3 pt-2 pb-1 font-medium uppercase tracking-wider">
              Documents ({documents.length})
            </div>
            <div className="max-h-48 overflow-y-auto">
              {documents.map((doc) => (
                <button key={doc.id} onClick={() => { onSelect(doc.id); onToggle(); }}
                  className={`w-full text-left px-3 py-2 text-xs hover:bg-neutral-800/60 transition-colors flex items-center gap-2 ${
                    doc.id === activeId ? 'bg-neutral-800/40 text-neutral-100' : 'text-neutral-400'
                  }`}>
                  <span className="text-base">📄</span>
                  <div className="flex-1 min-w-0">
                    <div className="truncate font-medium">{doc.filename}</div>
                    <div className="text-[10px] text-neutral-600 truncate">{new Date(doc.created_at).toLocaleDateString()}</div>
                  </div>
                  {doc.id === activeId && <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 flex-shrink-0" />}
                </button>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
