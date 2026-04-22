package state

import (
	"os"
	"testing"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	f, err := os.CreateTemp("", "zarvis-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	store, err := NewSQLiteStore(f.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestCreateAndGetUser(t *testing.T) {
	s := setupTestStore(t)
	user, err := s.CreateUser("test@example.com", "hashed_pw", "Test User")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("email = %q, want test@example.com", user.Email)
	}

	got, err := s.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("IDs don't match: %q vs %q", got.ID, user.ID)
	}

	gotByID, err := s.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if gotByID.Email != "test@example.com" {
		t.Errorf("email = %q", gotByID.Email)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.CreateUser("dup@test.com", "hash1", "User1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser("dup@test.com", "hash2", "User2")
	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.GetUserByEmail("noone@test.com")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateAndGetSession(t *testing.T) {
	s := setupTestStore(t)
	sess, err := s.CreateSession("user123", "fox")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.UserID != "user123" || sess.PrimaryAnimal != "fox" {
		t.Errorf("unexpected session: %+v", sess)
	}

	got, err := s.GetSession(sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.UserID != "user123" {
		t.Errorf("UserID = %q, want user123", got.UserID)
	}
}

func TestMessages(t *testing.T) {
	s := setupTestStore(t)
	sess, _ := s.CreateSession("u1", "")

	err := s.AppendMessage(sess.ID, "explorer", "user", "hello")
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	err = s.AppendMessage(sess.ID, "explorer", "assistant", "hi there")
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	msgs, err := s.RecentMessages(sess.ID, "explorer", 10)
	if err != nil {
		t.Fatalf("RecentMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" || msgs[1].Content != "hi there" {
		t.Errorf("wrong message order: %v", msgs)
	}

	// Different module should be empty
	msgs2, _ := s.RecentMessages(sess.ID, "table", 10)
	if len(msgs2) != 0 {
		t.Errorf("table module should have 0 messages, got %d", len(msgs2))
	}
}

func TestDocuments(t *testing.T) {
	s := setupTestStore(t)
	sess, _ := s.CreateSession("u1", "")

	doc, err := s.SaveDocument(sess.ID, "test.csv", "a,b,c\n1,2,3")
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if doc.Filename != "test.csv" {
		t.Errorf("filename = %q", doc.Filename)
	}

	got, err := s.GetLatestDocument(sess.ID)
	if err != nil {
		t.Fatalf("GetLatestDocument: %v", err)
	}
	if got.RawContent != "a,b,c\n1,2,3" {
		t.Errorf("wrong content: %q", got.RawContent)
	}

	err = s.UpdateDocumentStructured(doc.ID, `{"rows":[]}`, `{"fields":[]}`, "A CSV file")
	if err != nil {
		t.Fatalf("UpdateDocumentStructured: %v", err)
	}

	updated, _ := s.GetDocument(doc.ID)
	if updated.StructuredJSON != `{"rows":[]}` {
		t.Errorf("structured not saved: %q", updated.StructuredJSON)
	}
}

func TestForests(t *testing.T) {
	s := setupTestStore(t)
	sess, _ := s.CreateSession("u1", "")

	forest, err := s.CreateForest(sess.ID, "Test Forest")
	if err != nil {
		t.Fatalf("CreateForest: %v", err)
	}

	doc, _ := s.SaveDocument(sess.ID, "doc1.txt", "content1")
	err = s.AddDocumentToForest(forest.ID, doc.ID)
	if err != nil {
		t.Fatalf("AddDocumentToForest: %v", err)
	}

	docs, err := s.GetForestDocuments(forest.ID)
	if err != nil {
		t.Fatalf("GetForestDocuments: %v", err)
	}
	if len(docs) != 1 || docs[0].Filename != "doc1.txt" {
		t.Errorf("unexpected docs: %v", docs)
	}

	forests, _ := s.ListForests(sess.ID)
	if len(forests) != 1 || forests[0].DocCount != 1 {
		t.Errorf("expected 1 forest with 1 doc, got %+v", forests)
	}

	// Clear forest
	err = s.ClearForest(forest.ID)
	if err != nil {
		t.Fatalf("ClearForest: %v", err)
	}
	docs2, _ := s.GetForestDocuments(forest.ID)
	if len(docs2) != 0 {
		t.Errorf("expected 0 docs after clear, got %d", len(docs2))
	}
}

func TestBadges(t *testing.T) {
	s := setupTestStore(t)
	sess, _ := s.CreateSession("u1", "")

	err := s.EarnBadge(sess.ID, "first_upload")
	if err != nil {
		t.Fatalf("EarnBadge: %v", err)
	}

	// Earning same badge again should not error (INSERT OR IGNORE)
	err = s.EarnBadge(sess.ID, "first_upload")
	if err != nil {
		t.Fatalf("duplicate EarnBadge: %v", err)
	}

	badges, err := s.GetBadges(sess.ID)
	if err != nil {
		t.Fatalf("GetBadges: %v", err)
	}
	if len(badges) != 1 || badges[0].BadgeKey != "first_upload" {
		t.Errorf("unexpected badges: %v", badges)
	}
}

func TestChunks(t *testing.T) {
	s := setupTestStore(t)

	chunks := []ChunkRecord{
		{DocumentID: 1, ForestID: 1, Content: "chunk one", Position: 0},
		{DocumentID: 1, ForestID: 1, Content: "chunk two", Position: 1},
	}
	err := s.SaveChunks(chunks)
	if err != nil {
		t.Fatalf("SaveChunks: %v", err)
	}

	got, err := s.GetForestChunks(1)
	if err != nil {
		t.Fatalf("GetForestChunks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}

	err = s.DeleteDocumentChunks(1, 1)
	if err != nil {
		t.Fatalf("DeleteDocumentChunks: %v", err)
	}
	got2, _ := s.GetForestChunks(1)
	if len(got2) != 0 {
		t.Errorf("expected 0 chunks after delete, got %d", len(got2))
	}
}
