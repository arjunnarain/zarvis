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
    <div className="flex gap-1 px-4 py-2 border-b border-neutral-800/40 overflow-x-auto">
      {MODULES.map((mod) => {
        const isActive = active === mod.id;
        const state = tabStates[mod.id];
        const enabled = !state || state.enabled;

        return (
          <div key={mod.id} className="relative group">
            <button
              onClick={() => enabled && onChange(mod.id)}
              disabled={!enabled}
              className={`
                relative flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all whitespace-nowrap
                ${!enabled ? 'opacity-30 cursor-not-allowed' : ''}
                ${isActive ? 'text-white' : enabled ? 'text-neutral-500 hover:text-neutral-300' : 'text-neutral-600'}
              `}
            >
              {isActive && enabled && (
                <motion.div
                  layoutId="activeTab"
                  className="absolute inset-0 rounded-lg"
                  style={{ backgroundColor: mod.color + '20', border: `1px solid ${mod.color}30` }}
                  transition={{ type: 'spring', stiffness: 300, damping: 30 }}
                />
              )}
              <span className="relative z-10">{enabled ? mod.emoji : '🔒'}</span>
              <span className="relative z-10">{mod.name}</span>
            </button>

            {/* Tooltip for disabled tabs */}
            {!enabled && state?.reason && (
              <div className="absolute top-full left-1/2 -translate-x-1/2 mt-1 px-2 py-1 bg-neutral-800 border border-neutral-700 rounded text-[10px] text-neutral-400 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-20">
                {state.reason}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
