import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { apiFetch } from '../lib/api';

interface Props {
  sessionId: string;
  onClose: () => void;
}

export default function RawViewer({ sessionId, onClose }: Props) {
  const [content, setContent] = useState<string | null>(null);
  const [filename, setFilename] = useState('');

  useEffect(() => {
    apiFetch(`/api/session/${sessionId}/document`)
      .then((r) => r.json())
      .then((doc) => {
        setFilename(doc.filename || 'document');
        setContent(doc.raw_content || '(empty)');
      })
      .catch(() => setContent('Failed to load document'));
  }, [sessionId]);

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.95, y: 10 }}
        animate={{ scale: 1, y: 0 }}
        exit={{ scale: 0.95, y: 10 }}
        onClick={(e) => e.stopPropagation()}
        className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-2xl max-h-[80vh] flex flex-col overflow-hidden"
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-neutral-800">
          <div>
            <h2 className="text-sm font-semibold text-neutral-100">Raw Document</h2>
            <p className="text-[11px] text-neutral-500">{filename}</p>
          </div>
          <button onClick={onClose} className="text-neutral-500 hover:text-neutral-300 transition-colors">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="flex-1 overflow-auto p-4">
          {content === null ? (
            <div className="text-neutral-500 text-xs animate-pulse">Loading...</div>
          ) : (
            <pre className="text-xs font-mono text-neutral-300 whitespace-pre-wrap leading-relaxed">
              {content}
            </pre>
          )}
        </div>
      </motion.div>
    </motion.div>
  );
}
