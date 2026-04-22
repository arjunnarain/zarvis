import { useState, useEffect, useRef } from 'react';
import { motion } from 'framer-motion';
import { apiFetch } from '../lib/api';

interface SearchResult {
  source: string;
  line: string;
  text: string;
}

interface Props {
  sessionId: string;
  onInsertQuery: (text: string) => void;
  onClose: () => void;
}

export default function SearchPanel({ sessionId, onInsertQuery, onClose }: Props) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      setSearching(true);
      try {
        const res = await apiFetch(`/api/session/${sessionId}/search?q=${encodeURIComponent(query)}`);
        const data = await res.json();
        setResults(data ?? []);
      } catch { setResults([]); }
      finally { setSearching(false); }
    }, 300);
    return () => clearTimeout(debounceRef.current);
  }, [query, sessionId]);

  return (
    <motion.div
      initial={{ width: 0, opacity: 0 }}
      animate={{ width: 280, opacity: 1 }}
      exit={{ width: 0, opacity: 0 }}
      transition={{ duration: 0.2 }}
      className="border-l border-neutral-800/60 flex flex-col min-h-0 overflow-hidden"
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-neutral-800/40">
        <span className="text-xs font-medium text-neutral-400">Search Document</span>
        <button onClick={onClose} className="text-neutral-600 hover:text-neutral-400">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12" /></svg>
        </button>
      </div>

      <div className="px-3 py-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search values, fields..."
          autoFocus
          className="w-full bg-neutral-800 border border-neutral-700 rounded-lg px-2.5 py-1.5 text-xs text-neutral-100 outline-none focus:border-indigo-500/50 placeholder:text-neutral-600"
        />
      </div>

      <div className="flex-1 overflow-y-auto px-2 pb-2">
        {searching && <div className="text-[10px] text-neutral-600 text-center py-2">Searching...</div>}

        {!searching && query && results.length === 0 && (
          <div className="text-[10px] text-neutral-600 text-center py-4">No matches</div>
        )}

        {results.map((r, i) => (
          <button
            key={i}
            onClick={() => {
              onInsertQuery(`Tell me about: ${r.text.slice(0, 60)}`);
              onClose();
            }}
            className="w-full text-left p-2 rounded-lg text-[11px] hover:bg-neutral-800/60 transition-colors mb-0.5 group"
          >
            <div className="flex items-center gap-1.5 mb-0.5">
              <span className={`text-[9px] px-1 py-0.5 rounded ${r.source === 'raw' ? 'bg-orange-500/20 text-orange-400' : 'bg-indigo-500/20 text-indigo-400'}`}>
                {r.source}
              </span>
              <span className="text-[9px] text-neutral-600">line {r.line}</span>
            </div>
            <div className="text-neutral-400 group-hover:text-neutral-200 truncate">
              {highlightMatch(r.text, query)}
            </div>
          </button>
        ))}
      </div>
    </motion.div>
  );
}

function highlightMatch(text: string, query: string) {
  if (!query) return text;
  const idx = text.toLowerCase().indexOf(query.toLowerCase());
  if (idx === -1) return text;
  return (
    <>
      {text.slice(0, idx)}
      <span className="bg-yellow-500/30 text-yellow-200 rounded px-0.5">{text.slice(idx, idx + query.length)}</span>
      {text.slice(idx + query.length)}
    </>
  );
}
