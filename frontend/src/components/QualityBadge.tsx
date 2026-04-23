import { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { apiFetch } from '../lib/api';

interface Quality {
  score: number;
  grade: string;
  breakdown: {
    completeness: number;
    consistency: number;
    validity: number;
    structure: number;
  };
  suggestions: string[];
}

interface Props {
  sessionId: string;
  hasDocument: boolean;
}

const GRADE_COLORS: Record<string, string> = {
  A: '#22c55e',
  B: '#84cc16',
  C: '#eab308',
  D: '#f97316',
  F: '#ef4444',
};

export default function QualityBadge({ sessionId, hasDocument }: Props) {
  const [quality, setQuality] = useState<Quality | null>(null);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    if (!hasDocument || !sessionId) return;
    apiFetch(`/api/session/${sessionId}/quality`).then((r) => r.json())
      .then(setQuality).catch(() => {});
  }, [sessionId, hasDocument]);

  // Refresh periodically (picks up post-parse improvements)
  useEffect(() => {
    if (!hasDocument) return;
    const interval = setInterval(() => {
      apiFetch(`/api/session/${sessionId}/quality`).then((r) => r.json())
        .then(setQuality).catch(() => {});
    }, 10000);
    return () => clearInterval(interval);
  }, [sessionId, hasDocument]);

  if (!quality) return null;

  const color = GRADE_COLORS[quality.grade] ?? GRADE_COLORS.C;

  return (
    <div className="relative">
      <motion.button
        onClick={() => setExpanded((e) => !e)}
        className="flex items-center gap-1.5 px-2 py-1 rounded-lg border transition-all hover:bg-neutral-800/50"
        style={{ borderColor: color + '40' }}
        whileHover={{ scale: 1.02 }}
      >
        <div className="relative w-7 h-7">
          <svg width="28" height="28" viewBox="0 0 36 36">
            <circle cx="18" cy="18" r="15" fill="none" stroke="#262626" strokeWidth="3" />
            <circle cx="18" cy="18" r="15" fill="none" stroke={color} strokeWidth="3"
              strokeDasharray={`${quality.score * 0.94} 100`}
              strokeLinecap="round"
              transform="rotate(-90 18 18)" />
          </svg>
          <span className="absolute inset-0 flex items-center justify-center text-[8px] font-bold" style={{ color }}>
            {quality.score}
          </span>
        </div>
        <span className="text-[10px] text-neutral-400 hidden sm:block">Quality</span>
      </motion.button>

      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ opacity: 0, y: -4 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -4 }}
            className="absolute top-full right-0 mt-1 w-64 bg-neutral-900 border border-neutral-800 rounded-xl shadow-xl z-30 p-3 space-y-3"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-neutral-200">Data Quality</span>
              <span className="text-lg font-bold" style={{ color }}>{quality.grade}</span>
            </div>

            <div className="space-y-2">
              {([
                ['Completeness', quality.breakdown.completeness, 'How few missing values'],
                ['Consistency', quality.breakdown.consistency, 'Type uniformity across columns'],
                ['Validity', quality.breakdown.validity, 'How many values parse correctly'],
                ['Structure', quality.breakdown.structure, 'How well-organized the data is'],
              ] as const).map(([label, val, desc]) => (
                <div key={label}>
                  <div className="flex justify-between text-[10px]">
                    <span className="text-neutral-400">{label}</span>
                    <span className="text-neutral-500 tabular-nums">{val}%</span>
                  </div>
                  <div className="h-1 bg-neutral-800 rounded-full mt-0.5">
                    <div className="h-full rounded-full transition-all duration-500" style={{
                      width: `${val}%`,
                      backgroundColor: val >= 80 ? '#22c55e' : val >= 60 ? '#eab308' : '#ef4444',
                    }} />
                  </div>
                  <div className="text-[9px] text-neutral-600 mt-0.5">{desc}</div>
                </div>
              ))}
            </div>

            {quality.suggestions.length > 0 && (
              <div className="border-t border-neutral-800 pt-2">
                <div className="text-[10px] text-neutral-500 font-medium mb-1">Suggestions</div>
                {quality.suggestions.map((s, i) => (
                  <div key={i} className="text-[10px] text-neutral-400 flex gap-1.5 mt-1">
                    <span className="text-amber-400">•</span>{s}
                  </div>
                ))}
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
