import { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { apiFetch } from '../lib/api';

export interface ForestInfo {
  id: number;
  name: string;
  doc_count: number;
  created_at: string;
}

interface Props {
  sessionId: string;
  forests: ForestInfo[];
  activeForestId: number | null;
  onSelectForest: (id: number) => void;
  onForestCreated: (f: ForestInfo) => void;
  onForestUpdated: () => void;
  onForestCleared: () => void;
  documents: Array<{ id: number; filename: string }>;
}

export default function ForestManager({ sessionId, forests, activeForestId, onSelectForest, onForestCreated, onForestUpdated, onForestCleared, documents }: Props) {
  const [open, setOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');
  const [addingDoc, setAddingDoc] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const activeForest = forests.find((f) => f.id === activeForestId);

  // Click-outside to close
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        setCreating(false);
        setAddingDoc(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const createForest = async () => {
    if (!newName.trim()) return;
    const res = await apiFetch('/api/forest', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: sessionId, name: newName.trim() }),
    });
    if (res.ok) {
      const f = await res.json();
      onForestCreated(f);
      setNewName('');
      setCreating(false);
    }
  };

  const clearForest = async () => {
    if (!activeForestId) return;
    await apiFetch(`/api/forest/${activeForestId}/documents`, { method: 'DELETE' });
    onForestCleared();
    setOpen(false);
  };

  const addDocToForest = async (docId: number) => {
    if (!activeForestId) return;
    await apiFetch(`/api/forest/${activeForestId}/documents`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ document_id: docId }),
    });
    onForestUpdated();
    setAddingDoc(false);
  };

  return (
    <div className="relative" ref={ref}>
      <button onClick={() => setOpen((o) => !o)} className="flex items-center gap-1.5 text-[11px] text-neutral-400 hover:text-neutral-200 transition-colors">
        <span>🌲</span>
        <span className="max-w-[160px] truncate">
          {activeForest ? `${activeForest.name} (${activeForest.doc_count} doc${activeForest.doc_count !== 1 ? 's' : ''})` : 'No forest'}
        </span>
        <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" className={`transition-transform ${open ? 'rotate-180' : ''}`}>
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -4 }}
            className="absolute top-full left-0 mt-1 w-72 bg-neutral-900 border border-neutral-800 rounded-lg shadow-xl z-30 overflow-hidden"
          >
            <div className="flex items-center justify-between px-3 pt-2 pb-1">
              <span className="text-[10px] text-neutral-500 font-medium uppercase tracking-wider">Forests</span>
              <button onClick={() => setCreating(true)} className="text-[10px] text-indigo-400 hover:text-indigo-300">+ New</button>
            </div>

            <AnimatePresence>
              {creating && (
                <motion.div initial={{ height: 0 }} animate={{ height: 'auto' }} exit={{ height: 0 }} className="overflow-hidden">
                  <div className="px-3 py-2 flex gap-1.5">
                    <input value={newName} onChange={(e) => setNewName(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && createForest()}
                      placeholder="Forest name..." autoFocus
                      className="flex-1 bg-neutral-800 border border-neutral-700 rounded px-2 py-1 text-xs text-neutral-100 outline-none focus:border-indigo-500/50" />
                    <button onClick={createForest} disabled={!newName.trim()} className="px-2 py-1 text-xs bg-indigo-500 rounded text-white disabled:opacity-30">Create</button>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            <div className="max-h-40 overflow-y-auto">
              {forests.length === 0 && (
                <div className="px-3 py-3 text-xs text-neutral-600 text-center">No forests yet</div>
              )}
              {forests.map((f) => (
                <button key={f.id} onClick={() => { onSelectForest(f.id); setOpen(false); }}
                  className={`w-full text-left px-3 py-2 text-xs hover:bg-neutral-800/60 transition-colors flex items-center gap-2 ${
                    f.id === activeForestId ? 'bg-neutral-800/40 text-neutral-100' : 'text-neutral-400'
                  }`}>
                  <span>🌲</span>
                  <div className="flex-1 min-w-0">
                    <div className="truncate font-medium">{f.name}</div>
                    <div className="text-[10px] text-neutral-600">{f.doc_count} doc{f.doc_count !== 1 ? 's' : ''}</div>
                  </div>
                  {f.id === activeForestId && <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 flex-shrink-0" />}
                </button>
              ))}
            </div>

            {activeForestId && (
              <div className="border-t border-neutral-800/50 px-3 py-2 flex items-center justify-between">
                {documents.length > 0 ? (
                  <button onClick={() => setAddingDoc((a) => !a)} className="text-[10px] text-emerald-400 hover:text-emerald-300">
                    + Add document
                  </button>
                ) : <span />}
                {activeForest && activeForest.doc_count > 0 && (
                  <button onClick={clearForest} className="text-[10px] text-red-400 hover:text-red-300">
                    Reset forest
                  </button>
                )}
                <AnimatePresence>
                  {addingDoc && (
                    <motion.div initial={{ height: 0 }} animate={{ height: 'auto' }} exit={{ height: 0 }} className="overflow-hidden mt-1">
                      {documents.map((d) => (
                        <button key={d.id} onClick={() => addDocToForest(d.id)}
                          className="w-full text-left px-2 py-1.5 text-[11px] text-neutral-400 hover:bg-neutral-800/40 rounded transition-colors">
                          📄 {d.filename}
                        </button>
                      ))}
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
