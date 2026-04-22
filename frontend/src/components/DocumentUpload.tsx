import { useState, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { apiFetch } from '../lib/api';

interface Props {
  sessionId: string;
  onUploaded: (doc: { id: number; filename: string }) => void;
}

const SAMPLES = [
  { id: 'messy_csv', label: 'Messy Employee CSV', icon: '📊', desc: '10 rows with 5 date formats, $ vs plain numbers, missing names, "eighty thousand" as salary' },
  { id: 'invoice', label: 'Cloud Invoice', icon: '🧾', desc: 'Unstructured text — line items, addresses, tax math. No schema at all.' },
  { id: 'server_log', label: 'Server Crash Log', icon: '🖥️', desc: '20 log entries with circuit breaker events, errors, latency metrics, mixed levels' },
  { id: 'api_response', label: 'Org Chart JSON', icon: '🔗', desc: '3 levels deep — departments → teams → projects, plus financial metrics' },
  { id: 'support_tickets', label: 'Support Tickets', icon: '🎫', desc: 'Semi-structured email dump — timestamps, priorities, agents, mixed formatting' },
  { id: 'bank_statement', label: 'Bank Statement', icon: '🏦', desc: 'Tab-separated transactions with credits/debits, running balance, date ranges' },
  { id: 'resume', label: 'Resume / CV', icon: '👤', desc: 'Free-form text with education, experience, skills — entity extraction challenge' },
  { id: 'config_yaml', label: 'K8s Config', icon: '⚙️', desc: 'YAML deployment with nested specs, env vars, resource limits — hierarchical data' },
];

export default function DocumentUpload({ sessionId, onUploaded }: Props) {
  const [dragging, setDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [loadingLabel, setLoadingLabel] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = async (file: File) => {
    setUploading(true);
    setLoadingLabel(file.name);
    const form = new FormData();
    form.append('session_id', sessionId);
    form.append('file', file);
    try {
      const res = await apiFetch('/api/upload', { method: 'POST', body: form });
      if (!res.ok) { const t = await res.text(); throw new Error(t); }
      const doc = await res.json();
      onUploaded({ id: doc.id, filename: doc.filename });
    } catch (e) { console.error(e); }
    finally { setUploading(false); setLoadingLabel(''); }
  };

  const handleSample = async (sampleId: string) => {
    const sample = SAMPLES.find((s) => s.id === sampleId);
    setUploading(true);
    setLoadingLabel(sample?.label ?? sampleId);
    try {
      const res = await apiFetch('/api/sample', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId, sample: sampleId }),
      });
      if (!res.ok) throw new Error('sample load failed');
      const doc = await res.json();
      onUploaded({ id: doc.id, filename: doc.filename });
    } catch (e) { console.error(e); }
    finally { setUploading(false); setLoadingLabel(''); }
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer.files[0];
    if (file) handleFile(file);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -8 }}
      className="py-3 space-y-4"
    >
      {/* Drop zone */}
      <div
        onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        onClick={() => !uploading && inputRef.current?.click()}
        className={`
          border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-all
          ${dragging ? 'border-indigo-500 bg-indigo-500/5' : 'border-neutral-700/60 hover:border-neutral-500 bg-neutral-900/20'}
          ${uploading ? 'pointer-events-none opacity-60' : ''}
        `}
      >
        <input
          ref={inputRef}
          type="file"
          className="hidden"
          accept=".txt,.csv,.json,.md,.log,.xml,.html,.tsv,.yaml,.yml,.pdf"
          onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
        />
        <AnimatePresence mode="wait">
          {uploading ? (
            <motion.div key="up" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="py-2">
              <motion.div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full mx-auto" animate={{ rotate: 360 }} transition={{ duration: 1, repeat: Infinity, ease: 'linear' }} />
              <div className="text-xs text-neutral-400 mt-2">Processing {loadingLabel}...</div>
            </motion.div>
          ) : (
            <motion.div key="idle" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
              <div className="text-xl mb-1">📄</div>
              <div className="text-sm text-neutral-300">Drop a file here or click to browse</div>
              <div className="text-[11px] text-neutral-500 mt-1">
                PDF, CSV, JSON, TXT, XML, YAML, logs — up to 10MB
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Sample documents */}
      {!uploading && (
        <div>
          <div className="text-[11px] text-neutral-500 text-center mb-2">
            or try a sample messy document
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
            {SAMPLES.map((sample, idx) => (
              <motion.button
                key={sample.id}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.05 * idx }}
                onClick={() => handleSample(sample.id)}
                className="flex items-start gap-2 p-2.5 bg-neutral-900/50 border border-neutral-800/50 rounded-lg text-left hover:border-neutral-600/50 hover:bg-neutral-800/30 transition-all group"
              >
                <span className="text-base mt-0.5 group-hover:scale-110 transition-transform">{sample.icon}</span>
                <div>
                  <div className="text-xs font-medium text-neutral-200">{sample.label}</div>
                  <div className="text-[10px] text-neutral-500 leading-snug">{sample.desc}</div>
                </div>
              </motion.button>
            ))}
          </div>
        </div>
      )}
    </motion.div>
  );
}
