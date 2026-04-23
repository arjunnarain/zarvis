import { useEffect, useState, useRef, Suspense } from 'react';
import Chat from './components/Chat';
import SpiritOrb from './components/SpiritOrb';
import ModuleTabs, { type TabState } from './components/ModuleTabs';
import DocumentList, { type DocInfo } from './components/DocumentList';
import ForestManager, { type ForestInfo } from './components/ForestManager';
import AuthScreen from './components/AuthScreen';
import LandingPage from './components/LandingPage';
import ExportModal from './components/ExportModal';
import DiffView from './components/DiffView';
import { getToken, clearToken, apiFetch } from './lib/api';

export interface Session {
  id: string;
  primary_animal: string;
}

const FOREST_NAMES = [
  'Whispering Pines', 'Moonlit Grove', 'Ember Woods', 'Crystal Thicket',
  'Shadow Canopy', 'Starfall Glen', 'Iron Root', 'Mistwood',
  'Crimson Hollow', 'Thunderpeak Forest', 'Sapphire Glade', 'Obsidian Reach',
];

export default function App() {
  const [showApp, setShowApp] = useState(!!getToken());
  const [authed, setAuthed] = useState(!!getToken());
  const [userName, setUserName] = useState(localStorage.getItem('zarvis_user_name') || '');

  // Landing page → click "Open App" → auth screen → main app
  if (!showApp) {
    return <LandingPage onEnterApp={() => setShowApp(true)} />;
  }

  if (!authed) {
    return <AuthScreen onAuth={(name) => { setUserName(name); setAuthed(true); }} onBack={() => setShowApp(false)} />;
  }

  return <MainApp userName={userName} onLogout={() => { clearToken(); localStorage.removeItem('zarvis_user_name'); setAuthed(false); setShowApp(false); }} />;
}

function MainApp({ userName, onLogout }: { userName: string; onLogout: () => void }) {
  const [session, setSession] = useState<Session | null>(null);
  const [activeModule, setActiveModule] = useState('explorer');
  const [documents, setDocuments] = useState<DocInfo[]>([]);
  const [activeDocId, setActiveDocId] = useState<number | null>(null);
  const [docListOpen, setDocListOpen] = useState(false);
  const [forests, setForests] = useState<ForestInfo[]>([]);
  const [activeForestId, setActiveForestId] = useState<number | null>(null);
  const [showExport, setShowExport] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const [tabStates, setTabStates] = useState<Record<string, TabState>>({});
  const autoForestCreated = useRef(false);

  useEffect(() => {
    const stored = localStorage.getItem('zarvis_session_id');
    if (stored) {
      apiFetch(`/api/session/${stored}`)
        .then((r) => (r.ok ? r.json() : Promise.reject()))
        .then((s: Session) => {
          setSession(s);
          loadDocuments(s.id);
          loadForests(s.id);
          loadTabStates(s.id);
        })
        .catch(() => createSession());
    } else {
      createSession();
    }
  }, []);

  const createSession = () => {
    apiFetch('/api/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ primary_animal: '' }),
    })
      .then((r) => r.json())
      .then((s: Session) => {
        localStorage.setItem('zarvis_session_id', s.id);
        setSession(s);
        // Auto-create a default forest for new sessions
        autoCreateForest(s.id);
      });
  };

  const autoCreateForest = async (sid: string) => {
    if (autoForestCreated.current) return;
    autoForestCreated.current = true;
    const name = FOREST_NAMES[Math.floor(Math.random() * FOREST_NAMES.length)];
    try {
      const res = await apiFetch('/api/forest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sid, name }),
      });
      if (res.ok) {
        const f = await res.json();
        setForests([f]);
        setActiveForestId(f.id);
      }
    } catch {}
  };


  const loadDocuments = (sid: string) => {
    apiFetch(`/api/session/${sid}/documents`).then((r) => r.json())
      .then((docs: DocInfo[]) => {
        setDocuments(docs ?? []);
        if (docs?.length > 0 && !activeDocId) setActiveDocId(docs[0].id);
      }).catch(() => {});
  };

  const loadForests = (sid: string) => {
    apiFetch(`/api/session/${sid}/forests`).then((r) => r.json())
      .then((f: ForestInfo[]) => {
        setForests(f ?? []);
        if (f?.length > 0 && !activeForestId) setActiveForestId(f[0].id);
        // Auto-create if no forests exist (returning user with wiped DB)
        if (!f || f.length === 0) autoCreateForest(sid);
      }).catch(() => {});
  };

  const loadTabStates = (sid: string) => {
    apiFetch(`/api/session/${sid}/tabs`).then((r) => r.json())
      .then((tabs: Record<string, TabState>) => setTabStates(tabs ?? {}))
      .catch(() => {});
  };

  const handleDocumentUploaded = async (doc: { id: number; filename: string }) => {
    setDocuments((prev) => [{ id: doc.id, filename: doc.filename, summary: '', created_at: new Date().toISOString() }, ...prev]);
    setActiveDocId(doc.id);
    // Refresh tab states after upload
    if (session) setTimeout(() => loadTabStates(session.id), 500);
    // Auto-add to active forest and refresh count
    if (activeForestId) {
      try {
        await apiFetch(`/api/forest/${activeForestId}/documents`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ document_id: doc.id }),
        });
        // Immediately bump the local count so UI updates without waiting for server roundtrip
        setForests((prev) => prev.map((f) =>
          f.id === activeForestId ? { ...f, doc_count: f.doc_count + 1 } : f
        ));
      } catch {}
    }
  };

  const handleForestCreated = (f: ForestInfo) => {
    setForests((prev) => [f, ...prev]);
    setActiveForestId(f.id);
  };

  const [forestDocs, setForestDocs] = useState<Array<{ id: number; filename: string }>>([]);

  // Load forest docs whenever active forest changes
  useEffect(() => {
    if (!activeForestId) { setForestDocs([]); return; }
    apiFetch(`/api/forest/${activeForestId}/documents`).then((r) => r.json())
      .then((docs: Array<{ id: number; filename: string }>) => setForestDocs(docs ?? []))
      .catch(() => setForestDocs([]));
  }, [activeForestId, documents.length]); // re-fetch when docs change too

  const handleForestDocAdded = () => {
    if (activeForestId) {
      setForests((prev) => prev.map((f) =>
        f.id === activeForestId ? { ...f, doc_count: f.doc_count + 1 } : f
      ));
      // Also refresh forest docs list
      apiFetch(`/api/forest/${activeForestId}/documents`).then((r) => r.json())
        .then((docs: Array<{ id: number; filename: string }>) => setForestDocs(docs ?? []))
        .catch(() => {});
    }
  };

  if (!session) {
    return (
      <div className="h-screen flex items-center justify-center">
        <div className="text-neutral-600 animate-pulse text-sm">Awakening...</div>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col max-w-3xl mx-auto">
      {/* Header */}
      <header className="flex items-center gap-4 px-5 py-3 border-b border-white/5">
        <Suspense fallback={<div className="w-10 h-10" />}>
          <SpiritOrb stage={1} size="sm" />
        </Suspense>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2.5">
            <h1 style={{ fontFamily: 'var(--font-serif)' }} className="text-base text-white">Zarvis</h1>
            {userName && <span className="text-[10px] text-neutral-600">Hi, {userName}</span>}
          </div>
          <div className="flex items-center gap-3 mt-1">
            <ForestManager
              sessionId={session.id}
              forests={forests}
              activeForestId={activeForestId}
              onSelectForest={setActiveForestId}
              onForestCreated={handleForestCreated}
              onForestUpdated={handleForestDocAdded}
              onForestCleared={() => {
                if (activeForestId) {
                  setForests((prev) => prev.map((f) => f.id === activeForestId ? { ...f, doc_count: 0 } : f));
                  setForestDocs([]);
                }
              }}
              documents={documents.map((d) => ({ id: d.id, filename: d.filename }))}
            />
            {documents.length > 0 && (
              <DocumentList documents={documents} activeId={activeDocId} onSelect={setActiveDocId} open={docListOpen} onToggle={() => setDocListOpen((o) => !o)} />
            )}
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-1">
          {documents.length > 0 && (
            <>
              <button onClick={() => setShowDiff(true)} className="w-8 h-8 flex items-center justify-center rounded-lg text-neutral-500 hover:text-white hover:bg-white/5 transition-all" title="Before/After">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
                  <rect x="3" y="3" width="7" height="18" rx="1" /><rect x="14" y="3" width="7" height="18" rx="1" />
                </svg>
              </button>
              <button onClick={() => setShowExport(true)} className="w-8 h-8 flex items-center justify-center rounded-lg text-neutral-500 hover:text-white hover:bg-white/5 transition-all" title="Export">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3" />
                </svg>
              </button>
            </>
          )}
          <button onClick={onLogout} className="w-8 h-8 flex items-center justify-center rounded-lg text-neutral-600 hover:text-neutral-300 hover:bg-white/5 transition-all" title="Sign out">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4M16 17l5-5-5-5M21 12H9" />
            </svg>
          </button>
        </div>
      </header>

      <ModuleTabs active={activeModule} onChange={setActiveModule} tabStates={tabStates} />

      {/* Render all module chats, hide inactive ones to preserve state */}
      {['explorer', 'table', 'schema', 'summary', 'graphs', 'oracle'].map((mod) => (
        <div key={mod} className={`flex-1 flex flex-col min-h-0 ${mod === activeModule ? '' : 'hidden'}`}>
          <Chat
            sessionId={session.id}
            module={mod}
            isActive={mod === activeModule}
            userName={userName}
            hasDocument={documents.length > 0}
            activeForestId={activeForestId}
            forestDocs={forestDocs}
            onBadgeEarned={() => {}}
            onDocumentUploaded={handleDocumentUploaded}
            onDataParsed={() => { if (session) loadTabStates(session.id); }}
          />
        </div>
      ))}

      {showExport && (
        <ExportModal sessionId={session.id} onClose={() => setShowExport(false)} />
      )}
      {showDiff && (
        <DiffView sessionId={session.id} onClose={() => setShowDiff(false)} />
      )}
    </div>
  );
}
