import { useEffect, useState, useRef, Suspense } from 'react';
import Chat from './components/Chat';
import SpiritOrb from './components/SpiritOrb';
import ModuleTabs from './components/ModuleTabs';
import BadgeShelf from './components/BadgeShelf';
import DocumentList, { type DocInfo } from './components/DocumentList';
import ForestManager, { type ForestInfo } from './components/ForestManager';

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
  const [session, setSession] = useState<Session | null>(null);
  const [activeModule, setActiveModule] = useState('explorer');
  const [earnedBadges, setEarnedBadges] = useState<Set<string>>(new Set());
  const [documents, setDocuments] = useState<DocInfo[]>([]);
  const [activeDocId, setActiveDocId] = useState<number | null>(null);
  const [docListOpen, setDocListOpen] = useState(false);
  const [forests, setForests] = useState<ForestInfo[]>([]);
  const [activeForestId, setActiveForestId] = useState<number | null>(null);
  const autoForestCreated = useRef(false);

  useEffect(() => {
    const stored = localStorage.getItem('zarvis_session_id');
    if (stored) {
      fetch(`/api/session/${stored}`)
        .then((r) => (r.ok ? r.json() : Promise.reject()))
        .then((s: Session) => {
          setSession(s);
          loadBadges(s.id);
          loadDocuments(s.id);
          loadForests(s.id);
        })
        .catch(() => createSession());
    } else {
      createSession();
    }
  }, []);

  const createSession = () => {
    fetch('/api/session', {
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
      const res = await fetch('/api/forest', {
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

  const loadBadges = (sid: string) => {
    fetch(`/api/session/${sid}/badges`).then((r) => r.json())
      .then((b: Array<{ badge_key: string }>) => { if (b) setEarnedBadges(new Set(b.map((x) => x.badge_key))); })
      .catch(() => {});
  };

  const loadDocuments = (sid: string) => {
    fetch(`/api/session/${sid}/documents`).then((r) => r.json())
      .then((docs: DocInfo[]) => {
        setDocuments(docs ?? []);
        if (docs?.length > 0 && !activeDocId) setActiveDocId(docs[0].id);
      }).catch(() => {});
  };

  const loadForests = (sid: string) => {
    fetch(`/api/session/${sid}/forests`).then((r) => r.json())
      .then((f: ForestInfo[]) => {
        setForests(f ?? []);
        if (f?.length > 0 && !activeForestId) setActiveForestId(f[0].id);
        // Auto-create if no forests exist (returning user with wiped DB)
        if (!f || f.length === 0) autoCreateForest(sid);
      }).catch(() => {});
  };

  const handleDocumentUploaded = async (doc: { id: number; filename: string }) => {
    setDocuments((prev) => [{ id: doc.id, filename: doc.filename, summary: '', created_at: new Date().toISOString() }, ...prev]);
    setActiveDocId(doc.id);
    // Auto-add to active forest and refresh count
    if (activeForestId) {
      try {
        await fetch(`/api/forest/${activeForestId}/documents`, {
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
    fetch(`/api/forest/${activeForestId}/documents`).then((r) => r.json())
      .then((docs: Array<{ id: number; filename: string }>) => setForestDocs(docs ?? []))
      .catch(() => setForestDocs([]));
  }, [activeForestId, documents.length]); // re-fetch when docs change too

  const handleForestDocAdded = () => {
    if (activeForestId) {
      setForests((prev) => prev.map((f) =>
        f.id === activeForestId ? { ...f, doc_count: f.doc_count + 1 } : f
      ));
      // Also refresh forest docs list
      fetch(`/api/forest/${activeForestId}/documents`).then((r) => r.json())
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
    <div className="h-screen flex flex-col max-w-2xl mx-auto">
      <header className="flex items-center gap-3 px-4 py-2.5 border-b border-neutral-800/40">
        <Suspense fallback={<div className="w-10 h-10" />}>
          <SpiritOrb stage={1} size="sm" />
        </Suspense>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="text-sm font-semibold text-neutral-100">Zarvis</h1>
            <span className="text-[10px] text-neutral-600">Document Intelligence</span>
          </div>
          <div className="flex items-center gap-3 mt-0.5">
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
        <BadgeShelf earnedKeys={earnedBadges} />
      </header>

      <ModuleTabs active={activeModule} onChange={setActiveModule} />

      {/* Render all module chats, hide inactive ones to preserve state */}
      {['explorer', 'table', 'schema', 'summary', 'oracle'].map((mod) => (
        <div key={mod} className={`flex-1 flex flex-col min-h-0 ${mod === activeModule ? '' : 'hidden'}`}>
          <Chat
            sessionId={session.id}
            module={mod}
            isActive={mod === activeModule}
            hasDocument={documents.length > 0}
            activeForestId={activeForestId}
            forestDocs={forestDocs}
            onBadgeEarned={(k) => setEarnedBadges((p) => new Set([...p, k]))}
            onDocumentUploaded={handleDocumentUploaded}
          />
        </div>
      ))}
    </div>
  );
}
