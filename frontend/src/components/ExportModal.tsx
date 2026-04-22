import { useState } from 'react';
import { motion } from 'framer-motion';
import { getToken } from '../lib/api';

interface Props {
  sessionId: string;
  onClose: () => void;
}

const FORMATS = [
  { id: 'json', label: 'JSON', icon: '{ }', desc: 'Clean, structured JSON — best for APIs and further processing', ext: '.json' },
  { id: 'csv', label: 'CSV', icon: '📊', desc: 'Comma-separated — open in Excel, Google Sheets, or pandas', ext: '.csv' },
  { id: 'tsv', label: 'TSV', icon: '📋', desc: 'Tab-separated — paste-friendly, no comma conflicts', ext: '.tsv' },
  { id: 'markdown', label: 'Markdown', icon: '📝', desc: 'Markdown table — great for docs, README, Notion', ext: '.md' },
];

export default function ExportModal({ sessionId, onClose }: Props) {
  const [downloading, setDownloading] = useState<string | null>(null);

  const handleExport = async (format: string) => {
    setDownloading(format);
    try {
      const token = getToken();
      const res = await fetch(`/api/session/${sessionId}/export?format=${format}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        const text = await res.text();
        alert(text || 'Export failed');
        return;
      }
      // Get filename from Content-Disposition header
      const disposition = res.headers.get('Content-Disposition') || '';
      const match = disposition.match(/filename="([^"]+)"/);
      const filename = match ? match[1] : `export.${format}`;

      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
      onClose();
    } catch (e) {
      console.error(e);
      alert('Export failed');
    } finally {
      setDownloading(null);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.95, y: 10 }}
        animate={{ scale: 1, y: 0 }}
        exit={{ scale: 0.95, y: 10 }}
        onClick={(e) => e.stopPropagation()}
        className="bg-neutral-900 border border-neutral-800 rounded-2xl p-6 max-w-sm w-full space-y-4"
      >
        <div>
          <h2 className="text-base font-semibold text-neutral-100">Export Structured Data</h2>
          <p className="text-xs text-neutral-500 mt-1">
            Download the parsed data in your preferred format
          </p>
        </div>

        <div className="space-y-2">
          {FORMATS.map((fmt) => (
            <button
              key={fmt.id}
              onClick={() => handleExport(fmt.id)}
              disabled={!!downloading}
              className="w-full flex items-center gap-3 p-3 bg-neutral-800/50 border border-neutral-700/40 rounded-xl text-left hover:bg-neutral-800 hover:border-neutral-600/50 transition-all disabled:opacity-50 group"
            >
              <span className="text-lg w-8 text-center group-hover:scale-110 transition-transform">
                {fmt.icon}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-neutral-100">{fmt.label}</span>
                  <span className="text-[10px] text-neutral-600">{fmt.ext}</span>
                </div>
                <p className="text-[11px] text-neutral-500 mt-0.5">{fmt.desc}</p>
              </div>
              {downloading === fmt.id ? (
                <motion.div className="w-4 h-4 border-2 border-indigo-400 border-t-transparent rounded-full flex-shrink-0"
                  animate={{ rotate: 360 }} transition={{ duration: 0.8, repeat: Infinity, ease: 'linear' }} />
              ) : (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-neutral-600 group-hover:text-neutral-300 flex-shrink-0">
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3" />
                </svg>
              )}
            </button>
          ))}
        </div>

        <button
          onClick={onClose}
          className="w-full py-2 text-xs text-neutral-500 hover:text-neutral-300 transition-colors"
        >
          Cancel
        </button>
      </motion.div>
    </motion.div>
  );
}
