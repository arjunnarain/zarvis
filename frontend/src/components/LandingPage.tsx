import { Suspense, useRef } from 'react';
import { motion, useInView } from 'framer-motion';
import SpiritOrb from './SpiritOrb';

interface Props {
  onEnterApp: () => void;
}

function FadeIn({ children, delay = 0 }: { children: React.ReactNode; delay?: number }) {
  const ref = useRef(null);
  const inView = useInView(ref, { once: true, margin: '-80px' });
  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 40 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.8, delay, ease: [0.25, 0.1, 0.25, 1] }}
    >
      {children}
    </motion.div>
  );
}

const FEATURES = [
  {
    emoji: '🦊',
    name: 'Explorer',
    title: 'Parse Any Document',
    desc: 'Upload messy CSVs, invoices, logs, PDFs, YAML — the Fox spirit extracts clean, structured JSON. Detects document type automatically, flags quality issues, normalizes dates and currencies.',
    color: '#f97316',
  },
  {
    emoji: '🦉',
    name: 'Table',
    title: 'Query Your Data',
    desc: 'View structured data as interactive tables. Filter, sort, aggregate — all through natural language. "Show rows where salary > 90000" just works.',
    color: '#6366f1',
  },
  {
    emoji: '🐉',
    name: 'Schema',
    title: 'Infer the Structure',
    desc: 'The Dragon maps every field: types, nullability, enums, relationships. Spots data quality issues — mixed types, missing values, inconsistent formats.',
    color: '#ef4444',
  },
  {
    emoji: '📈',
    name: 'Graphs',
    title: 'Visualize Patterns',
    desc: 'Bar charts, pie charts, trend lines — rendered inline as SVG. The Phoenix analyzes your data and suggests the most meaningful visualizations.',
    color: '#eab308',
  },
  {
    emoji: '🌲',
    name: 'Oracle',
    title: 'Cross-Document Intelligence',
    desc: 'Group documents into Forests. Query across all of them at once — compare invoices, find patterns in logs, correlate datasets. Powered by BM25 search.',
    color: '#22c55e',
  },
];

const BEFORE_AFTER = {
  before: `Name,  Age, City,  Salary, Start Date
Alice Smith, 30, New York, $85,000, 01/15/2023
Bob Johnson,  25,San Francisco,92000, 2023-03-22
Charlie Brown, 35, Chicago, $78000, March 1 2022
Diana Prince, , NYC, 95000.00, 15-Jan-2024
Frank Miller, 28, New York, eighty thousand, 01-2023
, 45, Boston, $102,000, 2022-06-01`,
  after: `{
  "employees": [
    { "name": "Alice Smith", "age": 30,
      "city": "New York", "salary": 85000,
      "startDate": "2023-01-15" },
    { "name": "Bob Johnson", "age": 25,
      "city": "San Francisco", "salary": 92000,
      "startDate": "2023-03-22" },
    ...
  ],
  "dataQuality": {
    "issues": 6,
    "dateFormats": 5,
    "missingValues": { "name": 1, "age": 2 }
  }
}`,
};

export default function LandingPage({ onEnterApp }: Props) {
  return (
    <div className="min-h-screen bg-black">
      {/* Nav */}
      <nav className="fixed top-0 left-0 right-0 z-50 bg-black/80 backdrop-blur-md border-b border-white/5">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span style={{ fontFamily: 'var(--font-serif)', letterSpacing: '-0.02em' }} className="text-xl text-white">Zarvis</span>
          </div>
          <div className="flex items-center gap-6">
            <a href="#features" className="text-sm text-neutral-400 hover:text-white transition-colors">Features</a>
            <a href="#how-it-works" className="text-sm text-neutral-400 hover:text-white transition-colors">How it works</a>
            <button
              onClick={onEnterApp}
              className="px-5 py-2 text-sm bg-white text-black rounded-full font-medium hover:bg-neutral-200 transition-colors"
            >
              Open App
            </button>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="min-h-screen flex flex-col items-center justify-center px-6 pt-20 relative overflow-hidden">
        {/* Fullscreen orb — fills entire hero as ambient background */}
        <Suspense fallback={null}>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 0.35 }}
            transition={{ duration: 3, ease: 'easeOut' }}
            className="absolute inset-0 pointer-events-none"
          >
            <SpiritOrb stage={4} size="full" />
          </motion.div>
        </Suspense>

        {/* Radial glow behind title */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 2, delay: 0.5 }}
          className="absolute inset-0 pointer-events-none"
          style={{ background: 'radial-gradient(ellipse 60% 40% at 50% 45%, rgba(212, 168, 83, 0.06) 0%, transparent 70%)' }}
        />

        {/* ZARVIS wordmark */}
        <motion.div
          initial={{ opacity: 0, scale: 0.9, y: 20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          transition={{ duration: 1, delay: 0.2, ease: [0.25, 0.1, 0.25, 1] }}
          className="relative z-10"
        >
          <h1
            style={{
              fontFamily: 'var(--font-serif)',
              fontSize: 'clamp(4rem, 12vw, 10rem)',
              lineHeight: 0.9,
              letterSpacing: '-0.03em',
              color: 'transparent',
              backgroundImage: 'linear-gradient(180deg, #ffffff 0%, #a3a3a3 50%, #525252 100%)',
              backgroundClip: 'text',
              WebkitBackgroundClip: 'text',
              textAlign: 'center',
            }}
          >
            Zarvis
          </h1>
        </motion.div>

        {/* Tagline */}
        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.6 }}
          style={{ fontFamily: 'var(--font-serif)' }}
          className="text-xl sm:text-2xl lg:text-3xl text-center mt-6 max-w-3xl leading-snug text-white/80 relative z-10"
        >
          Turn Messy Documents Into Structured Data
        </motion.p>

        <motion.p
          initial={{ opacity: 0, y: 15 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.8 }}
          className="text-sm sm:text-base text-neutral-500 text-center mt-4 max-w-xl leading-relaxed relative z-10"
        >
          Upload any document — CSV, PDF, JSON, logs, invoices. Six AI-powered spirit guides parse, query, visualize, and compare your data.
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 1.0 }}
          className="flex gap-4 mt-10 relative z-10"
        >
          <button
            onClick={onEnterApp}
            className="px-8 py-3 bg-white text-black rounded-full font-medium text-sm hover:bg-neutral-200 transition-colors"
          >
            Try it now
          </button>
          <a
            href="#how-it-works"
            className="px-8 py-3 border border-neutral-700 text-neutral-300 rounded-full text-sm hover:border-neutral-500 hover:text-white transition-colors"
          >
            See how it works
          </a>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 1.5, duration: 1 }}
          className="mt-20 text-neutral-600 text-xs animate-bounce"
        >
          ↓ Scroll
        </motion.div>
      </section>

      {/* Before/After */}
      <section id="how-it-works" className="py-32 px-6">
        <div className="max-w-5xl mx-auto">
          <FadeIn>
            <p className="text-xs uppercase tracking-[0.2em] text-neutral-500 text-center">The Transformation</p>
            <h2 style={{ fontFamily: 'var(--font-serif)' }} className="text-3xl sm:text-5xl text-center mt-4 text-white">
              From Chaos to Clarity
            </h2>
          </FadeIn>

          <div className="grid md:grid-cols-2 gap-6 mt-16">
            <FadeIn delay={0.1}>
              <div className="rounded-2xl border border-red-500/20 bg-red-500/5 p-1">
                <div className="flex items-center gap-2 px-4 py-2">
                  <span className="w-2 h-2 rounded-full bg-red-500/60" />
                  <span className="text-[10px] uppercase tracking-wider text-red-400/60">Raw Input</span>
                </div>
                <pre className="px-4 pb-4 text-[11px] font-mono text-red-300/70 whitespace-pre-wrap leading-relaxed overflow-hidden max-h-64">
                  {BEFORE_AFTER.before}
                </pre>
              </div>
            </FadeIn>

            <FadeIn delay={0.3}>
              <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 p-1">
                <div className="flex items-center gap-2 px-4 py-2">
                  <span className="w-2 h-2 rounded-full bg-emerald-500/60" />
                  <span className="text-[10px] uppercase tracking-wider text-emerald-400/60">Structured Output</span>
                </div>
                <pre className="px-4 pb-4 text-[11px] font-mono text-emerald-300/70 whitespace-pre-wrap leading-relaxed overflow-hidden max-h-64">
                  {BEFORE_AFTER.after}
                </pre>
              </div>
            </FadeIn>
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="py-32 px-6">
        <div className="max-w-5xl mx-auto">
          <FadeIn>
            <p className="text-xs uppercase tracking-[0.2em] text-neutral-500 text-center">Five Spirit Guides</p>
            <h2 style={{ fontFamily: 'var(--font-serif)' }} className="text-3xl sm:text-5xl text-center mt-4 text-white">
              Every Lens You Need
            </h2>
          </FadeIn>

          <div className="mt-20 space-y-24">
            {FEATURES.map((f, i) => (
              <FadeIn key={f.name} delay={0.1}>
                <div className={`flex flex-col ${i % 2 === 1 ? 'md:flex-row-reverse' : 'md:flex-row'} items-center gap-12`}>
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-4">
                      <span className="text-3xl">{f.emoji}</span>
                      <span className="text-xs uppercase tracking-[0.15em] font-medium" style={{ color: f.color }}>{f.name}</span>
                    </div>
                    <h3 style={{ fontFamily: 'var(--font-serif)' }} className="text-2xl sm:text-3xl text-white mb-4">
                      {f.title}
                    </h3>
                    <p className="text-neutral-400 leading-relaxed">
                      {f.desc}
                    </p>
                  </div>
                  <div className="flex-shrink-0 w-48 h-48 rounded-2xl flex items-center justify-center" style={{ background: f.color + '08', border: `1px solid ${f.color}15` }}>
                    <span className="text-6xl">{f.emoji}</span>
                  </div>
                </div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* Tech */}
      <section className="py-32 px-6 border-t border-white/5">
        <div className="max-w-4xl mx-auto">
          <FadeIn>
            <p className="text-xs uppercase tracking-[0.2em] text-neutral-500 text-center">Under the Hood</p>
            <h2 style={{ fontFamily: 'var(--font-serif)' }} className="text-3xl sm:text-5xl text-center mt-4 text-white">
              Not Just a Wrapper
            </h2>
            <p className="text-neutral-400 text-center mt-6 max-w-2xl mx-auto leading-relaxed">
              The backend does real computation — document type detection, schema inference, data quality scoring, and BM25 search — all in Go, before the AI ever sees your data.
            </p>
          </FadeIn>

          <div className="grid sm:grid-cols-3 gap-6 mt-16">
            {[
              ['BM25 Search', 'Pure Go search engine. Documents chunked, tokenized, and scored for cross-document retrieval.', '🔍'],
              ['Schema Inference', 'Server-side type detection, nullability, enum detection — no LLM needed.', '📐'],
              ['Quality Scoring', '0-100 score with breakdown: completeness, consistency, validity, structure.', '📊'],
            ].map(([title, desc, icon], i) => (
              <FadeIn key={title as string} delay={i * 0.15}>
                <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6">
                  <span className="text-2xl">{icon}</span>
                  <h3 className="text-sm font-semibold text-white mt-3">{title}</h3>
                  <p className="text-xs text-neutral-500 mt-2 leading-relaxed">{desc}</p>
                </div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-32 px-6">
        <div className="max-w-3xl mx-auto text-center">
          <FadeIn>
            <h2 style={{ fontFamily: 'var(--font-serif)' }} className="text-3xl sm:text-5xl text-white">
              Ready to Structure Your Data?
            </h2>
            <p className="text-neutral-400 mt-6">
              Upload a document and see the transformation in seconds.
            </p>
            <button
              onClick={onEnterApp}
              className="mt-10 px-10 py-4 bg-white text-black rounded-full font-medium hover:bg-neutral-200 transition-colors"
            >
              Open Zarvis
            </button>
          </FadeIn>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-white/5 py-8 px-6">
        <div className="max-w-5xl mx-auto flex items-center justify-between text-xs text-neutral-600">
          <span style={{ fontFamily: 'var(--font-serif)' }}>Zarvis</span>
        </div>
      </footer>
    </div>
  );
}
