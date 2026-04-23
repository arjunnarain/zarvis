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
    <div
      className="flex items-center gap-0.5 px-5 py-1 overflow-x-auto"
      style={{ borderBottom: '1px solid var(--border)' }}
    >
      {MODULES.map((mod) => {
        const isActive = active === mod.id;
        const state = tabStates[mod.id];
        const enabled = !state || state.enabled;

        return (
          <div key={mod.id} className="relative group">
            <motion.button
              onClick={() => enabled && onChange(mod.id)}
              disabled={!enabled}
              whileHover={enabled && !isActive ? { y: -1 } : {}}
              whileTap={enabled ? { scale: 0.97 } : {}}
              className="relative flex items-center gap-2 px-4 py-3 rounded-lg transition-all duration-200 whitespace-nowrap"
              style={{
                opacity: !enabled ? 0.2 : 1,
                cursor: !enabled ? 'not-allowed' : 'pointer',
                background: isActive ? `${mod.color}10` : 'transparent',
              }}
              onMouseEnter={(e) => {
                if (enabled && !isActive) {
                  e.currentTarget.style.background = `${mod.color}08`;
                }
              }}
              onMouseLeave={(e) => {
                if (!isActive) {
                  e.currentTarget.style.background = 'transparent';
                }
              }}
            >
              {/* Active indicator — bottom glow line */}
              {isActive && enabled && (
                <motion.div
                  layoutId="tabIndicator"
                  className="absolute bottom-0 left-3 right-3 h-[2px]"
                  style={{
                    background: `linear-gradient(90deg, transparent, ${mod.color}, transparent)`,
                    boxShadow: `0 0 8px ${mod.color}60, 0 0 20px ${mod.color}20`,
                  }}
                  transition={{ type: 'spring', stiffness: 400, damping: 35 }}
                />
              )}

              {/* Hover glow dot behind emoji */}
              {enabled && !isActive && (
                <div
                  className="absolute left-3 top-1/2 -translate-y-1/2 w-6 h-6 rounded-full opacity-0 group-hover:opacity-100 transition-opacity duration-300 blur-md"
                  style={{ background: mod.color }}
                />
              )}

              <span
                className="relative z-10 text-sm transition-transform duration-200 group-hover:scale-110"
                style={{ filter: enabled ? 'none' : 'grayscale(1)' }}
              >
                {enabled ? mod.emoji : '🔒'}
              </span>
              <span
                className="relative z-10 transition-colors duration-200"
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: '11px',
                  fontWeight: isActive ? 500 : 400,
                  color: isActive ? '#ffffff' : enabled ? '#737373' : '#404040',
                  letterSpacing: '0.04em',
                  textTransform: 'uppercase' as const,
                }}
              >
                {mod.name}
              </span>

              {/* Active dot */}
              {isActive && (
                <motion.div
                  layoutId="tabDot"
                  className="w-1 h-1 rounded-full relative z-10"
                  style={{ background: mod.color, boxShadow: `0 0 4px ${mod.color}` }}
                  transition={{ type: 'spring', stiffness: 400, damping: 35 }}
                />
              )}
            </motion.button>

            {/* Tooltip for disabled tabs */}
            {!enabled && state?.reason && (
              <div
                className="absolute top-full left-1/2 -translate-x-1/2 mt-2 px-3 py-1.5 rounded-lg text-[10px] whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-20"
                style={{ background: 'var(--surface-3)', border: '1px solid var(--border)', color: '#737373' }}
              >
                {state.reason}
                <div className="absolute -top-1 left-1/2 -translate-x-1/2 w-2 h-2 rotate-45" style={{ background: 'var(--surface-3)', borderLeft: '1px solid var(--border)', borderTop: '1px solid var(--border)' }} />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
