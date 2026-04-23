import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { apiFetch } from '../lib/api';

interface Quality {
  score: number;
  grade: string;
  status: string;
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
}

const GRADE_COLORS: Record<string, string> = {
  A: '#22c55e', B: '#84cc16', C: '#eab308', D: '#f97316', F: '#ef4444', '-': '#525252',
};

export default function QualityBadge({ sessionId }: Props) {
  const [quality, setQuality] = useState<Quality | null>(null);

  const load = () => {
    apiFetch(`/api/session/${sessionId}/quality`).then((r) => r.json())
      .then(setQuality).catch(() => {});
  };

  useEffect(() => { load(); }, [sessionId]);

  // Refresh periodically to pick up post-parse improvements
  useEffect(() => {
    const interval = setInterval(load, 8000);
    return () => clearInterval(interval);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId]);

  if (!quality || quality.status === 'pending') return null;

  const color = GRADE_COLORS[quality.grade] ?? GRADE_COLORS.C;

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="my-3 bg-neutral-800/40 border border-neutral-700/30 rounded-xl p-3"
    >
      <div className="flex items-center gap-3">
        {/* Circular score */}
        <div className="relative w-12 h-12 flex-shrink-0">
          <svg width="48" height="48" viewBox="0 0 48 48">
            <circle cx="24" cy="24" r="20" fill="none" stroke="#262626" strokeWidth="3" />
            <circle cx="24" cy="24" r="20" fill="none" stroke={color} strokeWidth="3"
              strokeDasharray={`${quality.score * 1.26} 200`}
              strokeLinecap="round"
              transform="rotate(-90 24 24)" />
          </svg>
          <span className="absolute inset-0 flex items-center justify-center text-xs font-bold" style={{ color }}>
            {quality.score}
          </span>
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold text-neutral-200">Output Data Quality</span>
            <span className="text-sm font-bold" style={{ color }}>{quality.grade}</span>
          </div>

          {/* Mini bars */}
          <div className="grid grid-cols-4 gap-1.5 mt-2">
            {([
              ['Complete', quality.breakdown.completeness],
              ['Consistent', quality.breakdown.consistency],
              ['Valid', quality.breakdown.validity],
              ['Structure', quality.breakdown.structure],
            ] as const).map(([label, val]) => (
              <div key={label}>
                <div className="text-[8px] text-neutral-500">{label}</div>
                <div className="h-1 bg-neutral-700/50 rounded-full mt-0.5">
                  <div className="h-full rounded-full" style={{
                    width: `${val}%`,
                    backgroundColor: val >= 80 ? '#22c55e' : val >= 60 ? '#eab308' : '#ef4444',
                  }} />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {quality.suggestions.length > 0 && (
        <div className="mt-2 pt-2 border-t border-neutral-700/20">
          {quality.suggestions.map((s, i) => (
            <div key={i} className="text-[10px] text-neutral-400 flex gap-1.5 mt-0.5">
              <span className="text-amber-400/70">→</span>{s}
            </div>
          ))}
        </div>
      )}
    </motion.div>
  );
}
