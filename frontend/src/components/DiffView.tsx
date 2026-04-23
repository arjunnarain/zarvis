import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { apiFetch } from '../lib/api';

interface Props {
  sessionId: string;
  onClose: () => void;
}

export default function DiffView({ sessionId, onClose }: Props) {
  const [raw, setRaw] = useState('');
  const [structured, setStructured] = useState('');
  const [filename, setFilename] = useState('');

  useEffect(() => {
    apiFetch(`/api/session/${sessionId}/document`)
      .then((r) => r.json())
      .then((doc) => {
        setFilename(doc.filename ?? '');
        setRaw(doc.raw_content ?? '');
        if (doc.structured_json) {
          try {
            const parsed = JSON.parse(doc.structured_json);
            setStructured(JSON.stringify(parsed, null, 2));
          } catch {
            setStructured(doc.structured_json);
          }
        } else {
          setStructured('(not yet parsed — go to Explorer tab first)');
        }
      })
      .catch(() => {});
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
        initial={{ scale: 0.95 }}
        animate={{ scale: 1 }}
        exit={{ scale: 0.95 }}
        onClick={(e) => e.stopPropagation()}
        className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-5xl max-h-[85vh] flex flex-col overflow-hidden"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-neutral-800">
          <div>
            <h2 className="text-sm font-semibold text-neutral-100">Before → After</h2>
            <p className="text-[11px] text-neutral-500">{filename}</p>
          </div>
          <button onClick={onClose} className="text-neutral-500 hover:text-neutral-300">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M18 6L6 18M6 6l12 12" /></svg>
          </button>
        </div>

        {/* Split panes */}
        <div className="flex-1 flex min-h-0">
          {/* Left: Raw */}
          <div className="flex-1 flex flex-col border-r border-neutral-800/50 min-w-0">
            <div className="px-3 py-2 border-b border-neutral-800/30 flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-red-500/60" />
              <span className="text-[10px] font-medium text-neutral-400 uppercase tracking-wider">Raw Input</span>
              <span className="text-[9px] text-neutral-600 ml-auto">{raw.length.toLocaleString()} chars</span>
            </div>
            <div className="flex-1 overflow-auto p-3">
              <pre className="text-[11px] font-mono text-neutral-400 whitespace-pre-wrap leading-relaxed">{raw || 'Loading...'}</pre>
            </div>
          </div>

          {/* Right: Structured */}
          <div className="flex-1 flex flex-col min-w-0">
            <div className="px-3 py-2 border-b border-neutral-800/30 flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-green-500/60" />
              <span className="text-[10px] font-medium text-neutral-400 uppercase tracking-wider">Structured Output</span>
              <span className="text-[9px] text-neutral-600 ml-auto">{structured.length.toLocaleString()} chars</span>
            </div>
            <div className="flex-1 overflow-auto p-3">
              <pre className="text-[11px] font-mono text-emerald-400/80 whitespace-pre-wrap leading-relaxed">{structured || 'Loading...'}</pre>
            </div>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}
