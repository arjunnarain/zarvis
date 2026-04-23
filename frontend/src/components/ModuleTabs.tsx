import { motion } from 'framer-motion';

export interface ModuleDef {
  id: string;
  name: string;
  emoji: string;
  color: string;
  placeholder: string;
}

export interface TabState {
  enabled: boolean;
  reason: string;
  chart_types?: string[];
}

export const MODULES: ModuleDef[] = [
  { id: 'explorer', name: 'Explorer', emoji: '🦊', color: '#f97316', placeholder: 'Parse this document...' },
  { id: 'table', name: 'Table', emoji: '🦉', color: '#6366f1', placeholder: 'Show me the data as a table...' },
  { id: 'schema', name: 'Schema', emoji: '🐉', color: '#ef4444', placeholder: 'What fields are in this data?' },
  { id: 'summary', name: 'Summary', emoji: '🦦', color: '#14b8a6', placeholder: 'Summarize this document...' },
  { id: 'graphs', name: 'Graphs', emoji: '📈', color: '#eab308', placeholder: 'Visualize this data...' },
  { id: 'oracle', name: 'Oracle', emoji: '🌲', color: '#22c55e', placeholder: 'Ask across all documents in this forest...' },
];

interface Props {
  active: string;
  onChange: (id: string) => void;
  tabStates: Record<string, TabState>;
}

export default function ModuleTabs({ active, onChange, tabStates }: Props) {
  return (
    <div className="flex items-center gap-0.5 px-5 py-1 overflow-x-auto" style={{ borderBottom: '1px solid var(--border)' }}>
      {MODULES.map((mod) => {
        const isActive = active === mod.id;
        const state = tabStates[mod.id];
        const enabled = !state || state.enabled;

        return (
          <div key={mod.id} className="relative group">
            <button
              onClick={() => enabled && onChange(mod.id)}
              disabled={!enabled}
              className="relative flex items-center gap-2 px-4 py-3 transition-all whitespace-nowrap"
              style={{
                opacity: !enabled ? 0.25 : 1,
                cursor: !enabled ? 'not-allowed' : 'pointer',
              }}
            >
              {/* Active bottom indicator */}
              {isActive && enabled && (
                <motion.div
                  layoutId="tabIndicator"
                  className="absolute bottom-0 left-2 right-2 h-[2px] rounded-full"
                  style={{ background: `linear-gradient(90deg, ${mod.color}, ${mod.color}80)` }}
                  transition={{ type: 'spring', stiffness: 400, damping: 35 }}
                />
              )}

              <span className="text-sm" style={{ filter: enabled ? 'none' : 'grayscale(1)' }}>
                {enabled ? mod.emoji : '🔒'}
              </span>
              <span
                className="text-[11px] tracking-wide"
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontWeight: isActive ? 500 : 400,
                  color: isActive ? '#ffffff' : enabled ? '#737373' : '#404040',
                  letterSpacing: '0.04em',
                  textTransform: 'uppercase' as const,
                }}
              >
                {mod.name}
              </span>
            </button>

            {!enabled && state?.reason && (
              <div className="absolute top-full left-1/2 -translate-x-1/2 mt-2 px-3 py-1.5 rounded-lg text-[10px] whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-20"
                style={{ background: 'var(--surface-3)', border: '1px solid var(--border)', color: '#737373' }}>
                {state.reason}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
