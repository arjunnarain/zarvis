import { motion } from 'framer-motion';

interface Animal {
  species: string;
  emoji: string;
  name: string;
  style: string;
  description: string;
  color: string;
}

const ANIMALS: Animal[] = [
  {
    species: 'fox',
    emoji: '🦊',
    name: 'Fox',
    style: 'Quick & Curious',
    description: 'Asks probing questions, spots patterns, keeps things moving fast. Great for rapid iteration days.',
    color: '#f97316',
  },
  {
    species: 'owl',
    emoji: '🦉',
    name: 'Owl',
    style: 'Precise & Thorough',
    description: 'Structured responses, detailed summaries, tracks everything methodically. Great for documentation.',
    color: '#6366f1',
  },
  {
    species: 'dragon',
    emoji: '🐉',
    name: 'Dragon',
    style: 'Bold & Direct',
    description: 'No fluff, action-oriented, cuts to the point. Great for busy sprint days.',
    color: '#ef4444',
  },
  {
    species: 'otter',
    emoji: '🦦',
    name: 'Otter',
    style: 'Friendly & Supportive',
    description: 'Encouraging, celebrates wins, keeps morale up. Great for those tough debugging days.',
    color: '#14b8a6',
  },
];

interface Props {
  onSelect: (species: string) => void;
  disabled?: boolean;
}

export default function AnimalPicker({ onSelect, disabled }: Props) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95 }}
      transition={{ duration: 0.5 }}
      className="py-4 px-2"
    >
      <div className="text-center mb-5">
        <h2 className="text-sm font-semibold text-neutral-200">
          Pick your assistant's personality
        </h2>
        <p className="text-xs text-neutral-500 mt-1">
          This shapes how Zarvis talks to you across all modules
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        {ANIMALS.map((animal, idx) => (
          <motion.button
            key={animal.species}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.1 + idx * 0.08 }}
            whileHover={{ y: -3 }}
            whileTap={{ scale: 0.97 }}
            onClick={() => onSelect(animal.species)}
            disabled={disabled}
            className="
              bg-neutral-900/80 border border-neutral-800/80 rounded-xl p-3.5
              text-left transition-all duration-200
              hover:border-neutral-600/60
              disabled:opacity-40 disabled:cursor-not-allowed
              group
            "
          >
            <div className="flex items-center gap-2.5 mb-2">
              <span className="text-xl group-hover:scale-110 transition-transform">{animal.emoji}</span>
              <div>
                <div className="font-semibold text-sm text-neutral-100">{animal.name}</div>
                <div className="text-[10px] font-medium" style={{ color: animal.color }}>{animal.style}</div>
              </div>
            </div>
            <p className="text-[11px] text-neutral-400 leading-relaxed">
              {animal.description}
            </p>
            <div className="h-0.5 w-0 group-hover:w-full mt-2.5 rounded transition-all duration-300" style={{ backgroundColor: animal.color }} />
          </motion.button>
        ))}
      </div>
    </motion.div>
  );
}
