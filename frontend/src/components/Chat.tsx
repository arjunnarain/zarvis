import { useState, useRef, useEffect, Suspense, type ReactNode } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { MODULES } from './ModuleTabs';
import DocumentUpload from './DocumentUpload';
import RawViewer from './RawViewer';
import SearchPanel from './SearchPanel';
import SpiritOrb from './SpiritOrb';
import { apiFetch } from '../lib/api';

interface ToolCall { tool: string; display_name: string }
interface Message {
  role: 'user' | 'assistant';
  content: string;
  toolCalls?: ToolCall[];
  hidden?: boolean;
}

interface Props {
  sessionId: string;
  module: string;
  isActive: boolean;
  userName: string;
  hasDocument: boolean;
  activeForestId: number | null;
  forestDocs: Array<{ id: number; filename: string }>;
  onBadgeEarned: (key: string) => void;
  onDocumentUploaded: (doc: { id: number; filename: string }) => void;
}

const SUGGESTED_PROMPTS: Record<string, string[]> = {
  explorer: ['Parse this document into structured JSON', 'What type of document is this?', 'Extract all entities', 'Flag data quality issues'],
  table: ['Show the data as a table', 'Sort by the largest values', 'Show only rows with missing data', 'Calculate totals and averages'],
  schema: ['Infer the schema for this data', 'What field types are present?', 'Are there any enum fields?', 'Show data quality report'],
  summary: ['Summarize this document', 'What are the key findings?', 'Generate a brief report', 'What patterns do you see?'],
  graphs: ['What charts can you make from this data?', 'Show a bar chart of the top values', 'Show a pie chart breakdown', 'Plot a trend line'],
  oracle: ['Compare all documents in this forest', 'Find common patterns', 'Summarize everything', 'What are the differences?'],
};

export default function Chat({ sessionId, module, isActive, userName, hasDocument, activeForestId, forestDocs, onBadgeEarned, onDocumentUploaded }: Props) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [streaming, setStreaming] = useState(false);
  const [greetingDone, setGreetingDone] = useState(false);
  const [showUploadPanel, setShowUploadPanel] = useState(false);
  const [showMentions, setShowMentions] = useState(false);
  const [activeTools, setActiveTools] = useState<string[]>([]);
  const [showRawViewer, setShowRawViewer] = useState(false);
  const [showSearch, setShowSearch] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const streamingRef = useRef(false);
  const greetingSent = useRef(false);
  const sessionRef = useRef(sessionId);
  const moduleRef = useRef(module);
  sessionRef.current = sessionId;
  moduleRef.current = module;

  const showInitialUpload = greetingDone && !hasDocument && !streaming && module === 'explorer';
  // Show suggestions when not streaming and the last message is from assistant
  const lastVisible = messages.filter(m => !m.hidden).at(-1);
  const showSuggestions = greetingDone && hasDocument && !streaming && (!lastVisible || lastVisible.role === 'assistant');
  const moduleDef = MODULES.find((m) => m.id === module);
  const suggestions = SUGGESTED_PROMPTS[module] ?? [];

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages, showInitialUpload, activeTools]);

  // Core send function — uses ref to check streaming, not stale closure
  const doSend = async (text: string, hidden = false) => {
    if (streamingRef.current) return;
    streamingRef.current = true;
    setStreaming(true);

    setMessages((m) => [
      ...m,
      ...(hidden ? [] : [{ role: 'user' as const, content: text }]),
      { role: 'assistant' as const, content: '' },
    ]);

    try {
      const curSession = sessionRef.current;
      const curModule = moduleRef.current;
      const res = await apiFetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: curSession,
          module: curModule,
          message: curModule === 'oracle' && activeForestId ? `[forest_id:${activeForestId}] ${text}` : text,
        }),
      });
      if (!res.ok || !res.body) {
        // If 404, the session might not be ready — don't crash, just skip
        console.warn('chat request failed:', res.status);
        return;
      }
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const chunks = buffer.split('\n\n');
        buffer = chunks.pop() || '';
        for (const block of chunks) {
          if (!block.trim()) continue;
          let eventType = 'message';
          let data = '';
          for (const line of block.split('\n')) {
            if (line.startsWith('event: ')) eventType = line.slice(7).trim();
            else if (line.startsWith('data: ')) data = line.slice(6);
          }
          if (!data) continue;
          try { handleEvent(eventType, JSON.parse(data)); }
          catch (e) { console.error('parse SSE', e); }
        }
      }
    } catch (e) { console.error(e); }
    finally {
      streamingRef.current = false;
      setStreaming(false);
    }
  };

  // Auto-greet only when this tab becomes active for the first time
  useEffect(() => {
    if (!isActive || greetingSent.current || !sessionId) return;
    greetingSent.current = true;
    const mod = moduleRef.current;
    const nameHint = userName ? ` IMPORTANT: The user's name is "${userName}". You MUST greet them as "${userName}" — never use "friend", "traveller", or generic terms.` : '';
    let greeting: string;
    if (hasDocument) {
      const toolHint = mod === 'oracle'
        ? 'Use get_forest_documents or query_forest tools to access the data.'
        : 'Use your available tools to read the document data. Do NOT ask me to paste content.';
      greeting = `I switched to the ${mod} tab. I have documents uploaded. ${toolHint}${nameHint}`;
    } else {
      greeting = `Hello, I just arrived.${nameHint}`;
    }
    doSend(greeting, true);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isActive, sessionId]);

  const send = (text?: string) => {
    const msg = text ?? input;
    if (!msg.trim() || streamingRef.current) return;
    if (!text) setInput('');
    doSend(msg);
  };

  const handleDocUploaded = (doc: { id: number; filename: string }) => {
    onDocumentUploaded(doc);
    setShowUploadPanel(false);
    // Show the upload as a user message and add a parse prompt
    setMessages((m) => [
      ...m,
      { role: 'user' as const, content: `Uploaded: ${doc.filename}` },
    ]);
    setUploadedNeedsParse(true);
  };

  const [uploadedNeedsParse, setUploadedNeedsParse] = useState(false);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleEvent = (type: string, payload: any) => {
    switch (type) {
      case 'delta':
        setMessages((m) => {
          const copy = [...m]; const last = copy[copy.length - 1];
          if (last?.role === 'assistant') copy[copy.length - 1] = { ...last, content: last.content + payload.text };
          return copy;
        });
        break;
      case 'tool_use':
        setActiveTools((t) => [...t, payload.display_name]);
        setMessages((m) => {
          const copy = [...m]; const last = copy[copy.length - 1];
          if (last?.role === 'assistant') copy[copy.length - 1] = { ...last, toolCalls: [...(last.toolCalls ?? []), { tool: payload.tool, display_name: payload.display_name }] };
          return copy;
        });
        break;
      case 'tool_result':
        setActiveTools((t) => t.slice(1));
        break;
      case 'badge': onBadgeEarned(payload.badge_key); break;
      case 'done': setGreetingDone(true); setActiveTools([]); break;
      case 'error': console.error('server:', payload.message); setGreetingDone(true); setActiveTools([]); break;
    }
  };

  const visibleMessages = messages.filter((m) => !m.hidden);
  const showWelcome = !greetingDone && messages.length === 0;

  return (
    <div className="flex-1 flex min-h-0">
      <div className="flex-1 flex flex-col min-h-0">
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 pb-2 scroll-smooth">
        {showWelcome && (
          <div className="flex flex-col items-center justify-center h-full gap-3">
            <Suspense fallback={null}><SpiritOrb stage={1} size="lg" /></Suspense>
            <motion.p animate={{ opacity: [0.4, 0.9, 0.4] }} transition={{ duration: 2.5, repeat: Infinity }} className="text-neutral-500 text-xs">
              Initializing {moduleDef?.name ?? module}...
            </motion.p>
          </div>
        )}

        {!showWelcome && (
          <div className="space-y-3 pt-3">
            <AnimatePresence initial={false}>
              {visibleMessages.map((m, i) => (
                <motion.div key={i} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.2 }}>
                  <MessageBubble message={m} color={moduleDef?.color ?? '#a5b4fc'} />
                </motion.div>
              ))}
            </AnimatePresence>

            {/* Processing indicator — shows during tool execution */}
            {streaming && activeTools.length > 0 && (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="mx-1 my-2">
                <div className="bg-neutral-800/60 border border-neutral-700/40 rounded-xl px-4 py-3 space-y-2">
                  <div className="flex items-center gap-2">
                    <motion.div className="w-3 h-3 border-2 border-indigo-400 border-t-transparent rounded-full"
                      animate={{ rotate: 360 }} transition={{ duration: 0.8, repeat: Infinity, ease: 'linear' }} />
                    <span className="text-xs text-neutral-300 font-medium">Processing document...</span>
                  </div>
                  <div className="space-y-1">
                    {activeTools.map((tool, i) => (
                      <motion.div key={`${tool}-${i}`} initial={{ opacity: 0, x: -4 }} animate={{ opacity: 1, x: 0 }}
                        className="flex items-center gap-2 text-[11px] text-neutral-400">
                        <motion.span animate={{ opacity: [0.3, 1, 0.3] }} transition={{ duration: 1, repeat: Infinity }}>⚡</motion.span>
                        {tool}...
                      </motion.div>
                    ))}
                  </div>
                  <div className="h-1 bg-neutral-700/50 rounded-full overflow-hidden">
                    <motion.div className="h-full bg-indigo-500/60 rounded-full"
                      animate={{ width: ['0%', '100%'] }}
                      transition={{ duration: 15, repeat: Infinity, ease: 'linear' }} />
                  </div>
                </div>
              </motion.div>
            )}

            {/* Typing dots — only when streaming text with no active tools */}
            {streaming && activeTools.length === 0 && visibleMessages.length > 0 &&
              visibleMessages[visibleMessages.length - 1]?.role === 'assistant' &&
              !visibleMessages[visibleMessages.length - 1]?.content && (
              <div className="flex items-center gap-1.5 pl-1 py-1">
                {[0, 1, 2].map((i) => (
                  <motion.div key={i} className="w-1.5 h-1.5 rounded-full bg-neutral-500"
                    animate={{ opacity: [0.3, 1, 0.3] }}
                    transition={{ duration: 0.8, repeat: Infinity, delay: i * 0.2 }} />
                ))}
              </div>
            )}

            <AnimatePresence>
              {showInitialUpload && <DocumentUpload sessionId={sessionId} onUploaded={handleDocUploaded} />}
            </AnimatePresence>

            {/* Parse button after upload */}
            {uploadedNeedsParse && !streaming && (
              <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className="flex flex-wrap gap-2 pt-2">
                <button
                  onClick={() => { setUploadedNeedsParse(false); doSend('Please parse this document into structured data.'); }}
                  className="px-4 py-2 text-xs bg-indigo-500 text-white rounded-lg hover:bg-indigo-400 transition-colors flex items-center gap-2"
                >
                  <span>🦊</span> Parse into structured data
                </button>
                <button
                  onClick={() => { setUploadedNeedsParse(false); doSend('What type of document is this? Give me a quick summary.'); }}
                  className="px-4 py-2 text-xs bg-neutral-800 text-neutral-300 border border-neutral-700 rounded-lg hover:bg-neutral-700 transition-colors"
                >
                  Quick summary first
                </button>
                <button
                  onClick={() => setShowRawViewer(true)}
                  className="px-4 py-2 text-xs bg-neutral-800 text-neutral-300 border border-neutral-700 rounded-lg hover:bg-neutral-700 transition-colors flex items-center gap-1.5"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" />
                  </svg>
                  View raw file
                </button>
              </motion.div>
            )}

            {showSuggestions && suggestions.length > 0 && (
              <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className="flex flex-wrap gap-2 pt-2">
                {suggestions.map((prompt, i) => (
                  <motion.button key={prompt}
                    initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }} transition={{ delay: i * 0.05 }}
                    onClick={() => send(prompt)}
                    className="px-3 py-1.5 text-xs bg-neutral-800/60 border border-neutral-700/40 rounded-lg text-neutral-300 hover:bg-neutral-700/60 hover:border-neutral-600/60 hover:text-neutral-100 transition-all">
                    {prompt}
                  </motion.button>
                ))}
              </motion.div>
            )}
          </div>
        )}
      </div>

      {/* Inline upload panel */}
      <AnimatePresence>
        {showUploadPanel && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} className="overflow-hidden px-4">
            <DocumentUpload sessionId={sessionId} onUploaded={handleDocUploaded} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* @ mention popup */}
      <div className="px-4">
        <AnimatePresence>
          {showMentions && forestDocs.length > 0 && (
            <motion.div initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 4 }}
              className="bg-neutral-900 border border-neutral-700 rounded-lg shadow-xl mb-1 overflow-hidden">
              <div className="text-[10px] text-neutral-500 px-3 pt-1.5 pb-1">Reference a document</div>
              {forestDocs.map((d) => (
                <button key={d.id} onClick={() => { setInput((prev) => prev.replace(/@$/, `@${d.filename} `)); setShowMentions(false); }}
                  className="w-full text-left px-3 py-1.5 text-xs text-neutral-300 hover:bg-neutral-800/60 transition-colors flex items-center gap-2">
                  <span className="text-neutral-500">📄</span> {d.filename}
                </button>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Input bar */}
      <div className="px-4 pt-2 pb-4">
        <div className="flex items-center gap-2 bg-neutral-900/90 border border-neutral-800/70 rounded-xl px-3 py-2 focus-within:border-neutral-600/60 transition-colors">
          {/* Search toggle (explorer only) */}
          {module === 'explorer' && hasDocument && (
            <button
              onClick={() => setShowSearch((s) => !s)}
              className={`w-7 h-7 flex items-center justify-center rounded-lg transition-colors flex-shrink-0 ${
                showSearch ? 'bg-indigo-500/20 text-indigo-400' : 'text-neutral-500 hover:text-neutral-300 hover:bg-neutral-800'
              }`}
              title="Search document">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                <circle cx="11" cy="11" r="8" /><path d="M21 21l-4.35-4.35" />
              </svg>
            </button>
          )}
          <button
            onClick={() => setShowUploadPanel((p) => !p)}
            className={`w-7 h-7 flex items-center justify-center rounded-lg transition-colors flex-shrink-0 ${
              showUploadPanel ? 'bg-indigo-500/20 text-indigo-400' : 'text-neutral-500 hover:text-neutral-300 hover:bg-neutral-800'
            }`}
            title="Upload document">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48" />
            </svg>
          </button>
          <input
            value={input}
            onChange={(e) => {
              setInput(e.target.value);
              if (module === 'oracle' && e.target.value.endsWith('@') && forestDocs.length > 0) setShowMentions(true);
              else if (showMentions && !e.target.value.includes('@')) setShowMentions(false);
            }}
            onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && send()}
            placeholder={streaming ? 'Processing...' : moduleDef?.placeholder ?? 'Ask me anything...'}
            disabled={streaming}
            className="flex-1 bg-transparent outline-none text-sm text-neutral-100 placeholder:text-neutral-600"
          />
          <button onClick={() => send()} disabled={streaming || !input.trim()}
            className="w-7 h-7 flex items-center justify-center rounded-lg bg-indigo-500 text-white disabled:opacity-20 hover:bg-indigo-400 transition-colors flex-shrink-0">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>

      {showRawViewer && (
        <RawViewer sessionId={sessionId} onClose={() => setShowRawViewer(false)} />
      )}
      </div>{/* end inner column */}

      {/* Search panel */}
      <AnimatePresence>
        {showSearch && module === 'explorer' && (
          <SearchPanel
            sessionId={sessionId}
            onInsertQuery={(q) => { setInput(q); setShowSearch(false); }}
            onClose={() => setShowSearch(false)}
          />
        )}
      </AnimatePresence>
    </div>
  );
}

function MessageBubble({ message, color }: { message: Message; color: string }) {
  const isUser = message.role === 'user';
  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'}`}>
      <div className={`max-w-[90%] rounded-2xl px-3.5 py-2.5 text-sm leading-relaxed ${
        isUser ? 'bg-indigo-600/80 text-white rounded-br-md' : 'bg-neutral-800/70 text-neutral-200 rounded-bl-md'
      }`}>
        {!isUser && message.toolCalls?.map((t, i) => (
          <div key={i} className="text-[11px] mb-1.5 flex items-center gap-1.5" style={{ color }}>
            <span className="inline-block w-1 h-1 rounded-full" style={{ backgroundColor: color }} />
            {t.display_name}
          </div>
        ))}
        <div>{renderContent(message.content)}</div>
      </div>
    </div>
  );
}

function renderContent(text: string): ReactNode {
  if (!text) return null;
  const blocks = text.split(/(```[\s\S]*?```|\n\|[^\n]*\|(?:\n\|[^\n]*\|)*\n?)/g);
  return blocks.map((block, i) => {
    if (block.startsWith('```') && block.endsWith('```')) {
      const inner = block.slice(3, -3);
      // Chart rendering
      if (inner.startsWith('chart\n') || inner.startsWith('chart{')) {
        const jsonStr = inner.replace(/^chart\n?/, '');
        try {
          const chart = JSON.parse(jsonStr);
          return <InlineChart key={i} chart={chart} />;
        } catch { /* fall through to code block */ }
      }
      const code = inner.replace(/^\w+\n/, '');
      return <pre key={i} className="bg-neutral-900/80 rounded-lg px-3 py-2 my-2 text-xs overflow-x-auto font-mono text-neutral-300">{code}</pre>;
    }
    if (block.includes('|') && block.trim().startsWith('|')) return renderTable(block, i);
    return <span key={i} className="whitespace-pre-wrap">{renderInline(block)}</span>;
  });
}

function renderTable(tableText: string, key: number): ReactNode {
  const lines = tableText.trim().split('\n').filter(l => l.trim());
  if (lines.length < 2) return <span key={key} className="whitespace-pre-wrap">{tableText}</span>;
  const parseRow = (line: string) => line.split('|').map(c => c.trim()).filter((_, i, arr) => i > 0 && i < arr.length);
  const headers = parseRow(lines[0]);
  const sepIdx = lines.findIndex(l => /^\|[\s\-:]+\|/.test(l.trim()));
  const dataStart = sepIdx >= 0 ? sepIdx + 1 : 1;
  const rows = lines.slice(dataStart).map(parseRow);
  return (
    <div key={key} className="my-2 overflow-x-auto rounded-lg border border-neutral-700/50">
      <table className="w-full text-xs">
        <thead><tr className="bg-neutral-800/80 border-b border-neutral-700/50">
          {headers.map((h, j) => <th key={j} className="px-3 py-2 text-left font-medium text-neutral-300 whitespace-nowrap">{h}</th>)}
        </tr></thead>
        <tbody>
          {rows.map((row, ri) => (
            <tr key={ri} className={`border-b border-neutral-800/30 ${ri % 2 === 0 ? 'bg-neutral-900/30' : 'bg-neutral-900/10'} hover:bg-neutral-700/20 transition-colors`}>
              {row.map((cell, ci) => <td key={ci} className="px-3 py-1.5 text-neutral-300 whitespace-nowrap">{cell}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function renderInline(text: string): ReactNode {
  if (!text) return null;
  const parts = text.split(/(\*\*[^*]+\*\*|\*[^*]+\*|`[^`]+`)/g);
  return parts.map((part, i) => {
    if (part.startsWith('**') && part.endsWith('**')) return <strong key={i} className="font-semibold">{part.slice(2, -2)}</strong>;
    if (part.startsWith('*') && part.endsWith('*') && !part.startsWith('**')) return <em key={i}>{part.slice(1, -1)}</em>;
    if (part.startsWith('`') && part.endsWith('`')) return <code key={i} className="bg-neutral-900/60 px-1 py-0.5 rounded text-xs font-mono text-indigo-300">{part.slice(1, -1)}</code>;
    return <span key={i}>{part}</span>;
  });
}

// Inline SVG chart renderer
interface ChartData {
  type: string;
  title?: string;
  labels: string[];
  datasets: Array<{ label: string; data: number[] }>;
}

const CHART_COLORS = ['#6366f1', '#f97316', '#22c55e', '#ef4444', '#eab308', '#14b8a6', '#c084fc', '#f472b6'];

function InlineChart({ chart }: { chart: ChartData }) {
  const { type, title, labels, datasets } = chart;
  const data = datasets?.[0]?.data ?? [];
  const maxVal = Math.max(...data, 1);
  const w = 400;
  const h = 220;
  const pad = 40;

  if (type === 'bar') {
    const barW = Math.min(30, (w - pad * 2) / data.length - 4);
    const chartH = h - pad * 2;
    return (
      <div className="my-3 bg-neutral-900/60 rounded-xl p-3 overflow-x-auto">
        {title && <div className="text-xs font-medium text-neutral-300 mb-2">{title}</div>}
        <svg width={w} height={h} className="text-neutral-400">
          {data.map((val, i) => {
            const barH = (val / maxVal) * chartH;
            const x = pad + i * ((w - pad * 2) / data.length) + 2;
            const y = h - pad - barH;
            return (
              <g key={i}>
                <rect x={x} y={y} width={barW} height={barH} rx={3} fill={CHART_COLORS[i % CHART_COLORS.length]} opacity={0.85} />
                <text x={x + barW / 2} y={h - pad + 14} textAnchor="middle" fontSize="9" fill="#737373">{(labels[i] ?? '').slice(0, 8)}</text>
                <text x={x + barW / 2} y={y - 4} textAnchor="middle" fontSize="8" fill="#a3a3a3">{val.toLocaleString()}</text>
              </g>
            );
          })}
          <line x1={pad} y1={h - pad} x2={w - pad} y2={h - pad} stroke="#404040" strokeWidth="1" />
        </svg>
      </div>
    );
  }

  if (type === 'pie') {
    const total = data.reduce((a, b) => a + b, 0);
    let angle = 0;
    const cx = 100, cy = 100, r = 80;
    return (
      <div className="my-3 bg-neutral-900/60 rounded-xl p-3">
        {title && <div className="text-xs font-medium text-neutral-300 mb-2">{title}</div>}
        <div className="flex items-start gap-4">
          <svg width={200} height={200}>
            {data.map((val, i) => {
              const sliceAngle = (val / total) * Math.PI * 2;
              const startAngle = angle;
              angle += sliceAngle;
              const x1 = cx + r * Math.cos(startAngle);
              const y1 = cy + r * Math.sin(startAngle);
              const x2 = cx + r * Math.cos(startAngle + sliceAngle);
              const y2 = cy + r * Math.sin(startAngle + sliceAngle);
              const large = sliceAngle > Math.PI ? 1 : 0;
              return (
                <path key={i} d={`M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${large} 1 ${x2} ${y2} Z`}
                  fill={CHART_COLORS[i % CHART_COLORS.length]} opacity={0.85} stroke="#171717" strokeWidth="1" />
              );
            })}
          </svg>
          <div className="space-y-1 pt-2">
            {labels.map((label, i) => (
              <div key={i} className="flex items-center gap-2 text-[10px] text-neutral-400">
                <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }} />
                {label}: {data[i]?.toLocaleString()}
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (type === 'line') {
    const chartH = h - pad * 2;
    const points = data.map((val, i) => ({
      x: pad + (i / Math.max(data.length - 1, 1)) * (w - pad * 2),
      y: h - pad - (val / maxVal) * chartH,
    }));
    const pathD = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
    return (
      <div className="my-3 bg-neutral-900/60 rounded-xl p-3 overflow-x-auto">
        {title && <div className="text-xs font-medium text-neutral-300 mb-2">{title}</div>}
        <svg width={w} height={h}>
          <path d={pathD} fill="none" stroke="#6366f1" strokeWidth="2" />
          {points.map((p, i) => (
            <g key={i}>
              <circle cx={p.x} cy={p.y} r={3} fill="#6366f1" />
              <text x={p.x} y={h - pad + 14} textAnchor="middle" fontSize="9" fill="#737373">{(labels[i] ?? '').slice(0, 8)}</text>
            </g>
          ))}
          <line x1={pad} y1={h - pad} x2={w - pad} y2={h - pad} stroke="#404040" strokeWidth="1" />
        </svg>
      </div>
    );
  }

  // Fallback: show raw data
  return <pre className="bg-neutral-900/80 rounded-lg px-3 py-2 my-2 text-xs font-mono text-neutral-300">{JSON.stringify(chart, null, 2)}</pre>;
}
