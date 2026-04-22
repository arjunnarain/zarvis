// Package state persists Zarvis sessions, documents, and badges.
package state

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	PrimaryAnimal string    `json:"primary_animal"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Module    string    `json:"module"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Document struct {
	ID             int64     `json:"id"`
	SessionID      string    `json:"session_id"`
	Filename       string    `json:"filename"`
	RawContent     string    `json:"raw_content"`
	StructuredJSON string    `json:"structured_json,omitempty"`
	SchemaJSON     string    `json:"schema_json,omitempty"`
	Summary        string    `json:"summary,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type Badge struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	BadgeKey  string    `json:"badge_key"`
	EarnedAt  time.Time `json:"earned_at"`
}

type ChunkRecord struct {
	ID         int64  `json:"id"`
	DocumentID int64  `json:"document_id"`
	ForestID   int64  `json:"forest_id"`
	Content    string `json:"content"`
	Position   int    `json:"position"`
}

type Forest struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Name      string    `json:"name"`
	DocCount  int       `json:"doc_count"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	CreateUser(email, passwordHash, name string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)

	CreateSession(userID, animal string) (*Session, error)
	GetSession(id string) (*Session, error)
	UpdateSession(s *Session) error

	RecentMessages(sessionID, module string, limit int) ([]Message, error)
	AppendMessage(sessionID, module, role, content string) error

	SaveDocument(sessionID, filename, rawContent string) (*Document, error)
	GetDocument(id int64) (*Document, error)
	GetLatestDocument(sessionID string) (*Document, error)
	ListDocuments(sessionID string) ([]Document, error)
	UpdateDocumentStructured(id int64, structured, schema, summary string) error

	CreateForest(sessionID, name string) (*Forest, error)
	GetForest(id int64) (*Forest, error)
	ListForests(sessionID string) ([]Forest, error)
	AddDocumentToForest(forestID, docID int64) error
	GetForestDocuments(forestID int64) ([]Document, error)

	SaveChunks(chunks []ChunkRecord) error
	GetForestChunks(forestID int64) ([]ChunkRecord, error)
	DeleteDocumentChunks(docID, forestID int64) error
	GetForestsForDocument(docID int64) ([]int64, error)
	ClearForest(forestID int64) error

	EarnBadge(sessionID, badgeKey string) error
	GetBadges(sessionID string) ([]Badge, error)

	Close() error
}

type SQLiteStore struct{ db *sql.DB }

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT '',
    primary_animal TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    module TEXT NOT NULL DEFAULT 'explorer',
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
CREATE INDEX IF NOT EXISTS idx_messages_session_module ON messages(session_id, module, id);
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    raw_content TEXT NOT NULL,
    structured_json TEXT NOT NULL DEFAULT '',
    schema_json TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
CREATE INDEX IF NOT EXISTS idx_documents_session ON documents(session_id, id);
CREATE TABLE IF NOT EXISTS badges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    badge_key TEXT NOT NULL,
    earned_at DATETIME NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id),
    UNIQUE(session_id, badge_key)
);
CREATE TABLE IF NOT EXISTS forests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
CREATE TABLE IF NOT EXISTS forest_documents (
    forest_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    FOREIGN KEY (forest_id) REFERENCES forests(id),
    FOREIGN KEY (document_id) REFERENCES documents(id),
    UNIQUE(forest_id, document_id)
);
CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL,
    forest_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    position INTEGER NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents(id)
);
CREATE INDEX IF NOT EXISTS idx_chunks_forest ON chunks(forest_id);
`

func (s *SQLiteStore) CreateUser(email, passwordHash, name string) (*User, error) {
	now := time.Now().UTC()
	user := &User{ID: uuid.NewString(), Email: email, PasswordHash: passwordHash, Name: name, CreatedAt: now}
	_, err := s.db.Exec(`INSERT INTO users (id, email, password_hash, name, created_at) VALUES (?,?,?,?,?)`,
		user.ID, user.Email, user.PasswordHash, user.Name, user.CreatedAt)
	return user, err
}

func (s *SQLiteStore) GetUserByID(id string) (*User, error) {
	var u User
	err := s.db.QueryRow(`SELECT id, email, password_hash, name, created_at FROM users WHERE id=?`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *SQLiteStore) GetUserByEmail(email string) (*User, error) {
	var u User
	err := s.db.QueryRow(`SELECT id, email, password_hash, name, created_at FROM users WHERE email=?`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *SQLiteStore) CreateSession(userID, animal string) (*Session, error) {
	now := time.Now().UTC()
	sess := &Session{ID: uuid.NewString(), UserID: userID, PrimaryAnimal: animal, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_id, primary_animal, created_at, updated_at) VALUES (?,?,?,?,?)`,
		sess.ID, sess.UserID, sess.PrimaryAnimal, sess.CreatedAt, sess.UpdatedAt)
	return sess, err
}

func (s *SQLiteStore) GetSession(id string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(
		`SELECT id, user_id, primary_animal, created_at, updated_at FROM sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.UserID, &sess.PrimaryAnimal, &sess.CreatedAt, &sess.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &sess, err
}

func (s *SQLiteStore) UpdateSession(sess *Session) error {
	sess.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`UPDATE sessions SET primary_animal=?, updated_at=? WHERE id=?`,
		sess.PrimaryAnimal, sess.UpdatedAt, sess.ID)
	return err
}

func (s *SQLiteStore) RecentMessages(sessionID, module string, limit int) ([]Message, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, module, role, content, created_at FROM messages WHERE session_id=? AND module=? ORDER BY id DESC LIMIT ?`,
		sessionID, module, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Module, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

func (s *SQLiteStore) AppendMessage(sessionID, module, role, content string) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (session_id, module, role, content, created_at) VALUES (?,?,?,?,?)`,
		sessionID, module, role, content, time.Now().UTC())
	return err
}

func (s *SQLiteStore) SaveDocument(sessionID, filename, rawContent string) (*Document, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO documents (session_id, filename, raw_content, created_at) VALUES (?,?,?,?)`,
		sessionID, filename, rawContent, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Document{ID: id, SessionID: sessionID, Filename: filename, RawContent: rawContent, CreatedAt: now}, nil
}

func (s *SQLiteStore) GetDocument(id int64) (*Document, error) {
	var d Document
	err := s.db.QueryRow(
		`SELECT id, session_id, filename, raw_content, structured_json, schema_json, summary, created_at FROM documents WHERE id=?`, id,
	).Scan(&d.ID, &d.SessionID, &d.Filename, &d.RawContent, &d.StructuredJSON, &d.SchemaJSON, &d.Summary, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &d, err
}

func (s *SQLiteStore) GetLatestDocument(sessionID string) (*Document, error) {
	var d Document
	err := s.db.QueryRow(
		`SELECT id, session_id, filename, raw_content, structured_json, schema_json, summary, created_at FROM documents WHERE session_id=? ORDER BY id DESC LIMIT 1`, sessionID,
	).Scan(&d.ID, &d.SessionID, &d.Filename, &d.RawContent, &d.StructuredJSON, &d.SchemaJSON, &d.Summary, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &d, err
}

func (s *SQLiteStore) ListDocuments(sessionID string) ([]Document, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, filename, '', '', '', summary, created_at FROM documents WHERE session_id=? ORDER BY id DESC LIMIT 20`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.SessionID, &d.Filename, &d.RawContent, &d.StructuredJSON, &d.SchemaJSON, &d.Summary, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (s *SQLiteStore) UpdateDocumentStructured(id int64, structured, schemaJSON, summary string) error {
	_, err := s.db.Exec(`UPDATE documents SET structured_json=?, schema_json=?, summary=? WHERE id=?`,
		structured, schemaJSON, summary, id)
	return err
}

func (s *SQLiteStore) EarnBadge(sessionID, badgeKey string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO badges (session_id, badge_key, earned_at) VALUES (?,?,?)`,
		sessionID, badgeKey, time.Now().UTC())
	return err
}

func (s *SQLiteStore) GetBadges(sessionID string) ([]Badge, error) {
	rows, err := s.db.Query(`SELECT id, session_id, badge_key, earned_at FROM badges WHERE session_id=? ORDER BY earned_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var badges []Badge
	for rows.Next() {
		var b Badge
		if err := rows.Scan(&b.ID, &b.SessionID, &b.BadgeKey, &b.EarnedAt); err != nil {
			return nil, err
		}
		badges = append(badges, b)
	}
	return badges, rows.Err()
}

func (s *SQLiteStore) CreateForest(sessionID, name string) (*Forest, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO forests (session_id, name, created_at) VALUES (?,?,?)`, sessionID, name, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Forest{ID: id, SessionID: sessionID, Name: name, CreatedAt: now}, nil
}

func (s *SQLiteStore) GetForest(id int64) (*Forest, error) {
	var f Forest
	err := s.db.QueryRow(`SELECT id, session_id, name, created_at FROM forests WHERE id=?`, id).Scan(&f.ID, &f.SessionID, &f.Name, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	s.db.QueryRow(`SELECT COUNT(*) FROM forest_documents WHERE forest_id=?`, id).Scan(&f.DocCount)
	return &f, nil
}

func (s *SQLiteStore) ListForests(sessionID string) ([]Forest, error) {
	rows, err := s.db.Query(`SELECT f.id, f.session_id, f.name, f.created_at, COUNT(fd.document_id) FROM forests f LEFT JOIN forest_documents fd ON f.id=fd.forest_id WHERE f.session_id=? GROUP BY f.id ORDER BY f.id DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var forests []Forest
	for rows.Next() {
		var f Forest
		if err := rows.Scan(&f.ID, &f.SessionID, &f.Name, &f.CreatedAt, &f.DocCount); err != nil {
			return nil, err
		}
		forests = append(forests, f)
	}
	return forests, rows.Err()
}

func (s *SQLiteStore) AddDocumentToForest(forestID, docID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO forest_documents (forest_id, document_id) VALUES (?,?)`, forestID, docID)
	return err
}

func (s *SQLiteStore) GetForestDocuments(forestID int64) ([]Document, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.session_id, d.filename, d.raw_content, d.structured_json, d.schema_json, d.summary, d.created_at
		 FROM documents d JOIN forest_documents fd ON d.id=fd.document_id WHERE fd.forest_id=? ORDER BY d.id`, forestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.SessionID, &d.Filename, &d.RawContent, &d.StructuredJSON, &d.SchemaJSON, &d.Summary, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (s *SQLiteStore) SaveChunks(chunks []ChunkRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO chunks (document_id, forest_id, content, position) VALUES (?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, c := range chunks {
		if _, err := stmt.Exec(c.DocumentID, c.ForestID, c.Content, c.Position); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetForestChunks(forestID int64) ([]ChunkRecord, error) {
	rows, err := s.db.Query(`SELECT id, document_id, forest_id, content, position FROM chunks WHERE forest_id=? ORDER BY document_id, position`, forestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chunks []ChunkRecord
	for rows.Next() {
		var c ChunkRecord
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.ForestID, &c.Content, &c.Position); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (s *SQLiteStore) GetForestsForDocument(docID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT forest_id FROM forest_documents WHERE document_id=?`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SQLiteStore) ClearForest(forestID int64) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE forest_id=?`, forestID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM forest_documents WHERE forest_id=?`, forestID)
	return err
}

func (s *SQLiteStore) DeleteDocumentChunks(docID, forestID int64) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE document_id=? AND forest_id=?`, docID, forestID)
	return err
}

func (s *SQLiteStore) Close() error { return s.db.Close() }
