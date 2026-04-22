import { motion } from 'framer-motion';

const ALL_BADGES = [
  { key: 'first_upload', name: 'First Upload', description: 'Upload your first document', icon: '📄' },
  { key: 'structured', name: 'Structured', description: 'Parse a document', icon: '🔧' },
  { key: 'schema_master', name: 'Schema Master', description: 'Infer a schema', icon: '📐' },
  { key: 'summarizer', name: 'Summarizer', description: 'Generate a summary', icon: '📝' },
  { key: 'queried', name: 'Data Explorer', description: 'Query structured data', icon: '🔍' },
  { key: 'power_user', name: 'Power User', description: 'Process 5 documents', icon: '🚀' },
];

interface Props { earnedKeys: Set<string> }

export default function BadgeShelf({ earnedKeys }: Props) {
  return (
    <div className="flex gap-1">
      {ALL_BADGES.map((badge) => {
        const earned = earnedKeys.has(badge.key);
        return (
          <div key={badge.key} className="relative group">
            <motion.div
              className={`w-6 h-6 flex items-center justify-center rounded text-xs cursor-default ${earned ? '' : 'grayscale opacity-30'}`}
              whileHover={{ scale: 1.2 }}
            >
              {badge.icon}
            </motion.div>
            <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 bg-neutral-800 border border-neutral-700 rounded text-[10px] text-neutral-300 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-10">
              <div className="font-medium">{badge.name}</div>
              <div className="text-neutral-500">{badge.description}</div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
