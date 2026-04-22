import { motion } from 'framer-motion';

export interface ModuleDef {
  id: string;
  name: string;
  emoji: string;
  color: string;
  placeholder: string;
}

export const MODULES: ModuleDef[] = [
  { id: 'explorer', name: 'Explorer', emoji: '🦊', color: '#f97316', placeholder: 'Parse this document...' },
  { id: 'table', name: 'Table', emoji: '🦉', color: '#6366f1', placeholder: 'Show me the data as a table...' },
  { id: 'schema', name: 'Schema', emoji: '🐉', color: '#ef4444', placeholder: 'What fields are in this data?' },
  { id: 'summary', name: 'Summary', emoji: '🦦', color: '#14b8a6', placeholder: 'Summarize this document...' },
  { id: 'oracle', name: 'Oracle', emoji: '🌲', color: '#22c55e', placeholder: 'Ask across all documents in this forest...' },
];

interface Props {
  active: string;
  onChange: (id: string) => void;
}

export default function ModuleTabs({ active, onChange }: Props) {
  return (
    <div className="flex gap-1 px-4 py-2 border-b border-neutral-800/40">
      {MODULES.map((mod) => {
        const isActive = active === mod.id;
        return (
          <button
            key={mod.id}
            onClick={() => onChange(mod.id)}
            className={`
              relative flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all
              ${isActive ? 'text-white' : 'text-neutral-500 hover:text-neutral-300'}
            `}
          >
            {isActive && (
              <motion.div
                layoutId="activeTab"
                className="absolute inset-0 rounded-lg"
                style={{ backgroundColor: mod.color + '20', border: `1px solid ${mod.color}30` }}
                transition={{ type: 'spring', stiffness: 300, damping: 30 }}
              />
            )}
            <span className="relative z-10">{mod.emoji}</span>
            <span className="relative z-10">{mod.name}</span>
          </button>
        );
      })}
    </div>
  );
}
