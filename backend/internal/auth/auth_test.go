package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("testpass123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "testpass123" {
		t.Error("hash should not equal plaintext")
	}
	if !CheckPassword(hash, "testpass123") {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrongpass") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashPassword_DifferentHashesForSameInput(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("bcrypt should produce different hashes for same input (different salt)")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "usr_12345"

	token, err := GenerateToken(userID, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}

	got, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if got != userID {
		t.Errorf("ValidateToken returned %q, want %q", got, userID)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := GenerateToken("usr_1", "secret-a")
	_, err := ValidateToken(token, "secret-b")
	if err == nil {
		t.Error("ValidateToken should fail with wrong secret")
	}
}

func TestValidateToken_GarbageToken(t *testing.T) {
	_, err := ValidateToken("not.a.jwt", "secret")
	if err == nil {
		t.Error("ValidateToken should fail with garbage token")
	}
}

func TestMiddleware_NoHeader(t *testing.T) {
	mw := Middleware("secret")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	secret := "test-secret"
	token, _ := GenerateToken("usr_42", secret)

	var gotUserID string
	mw := Middleware(secret)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = GetUserID(r)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotUserID != "usr_42" {
		t.Errorf("GetUserID = %q, want %q", gotUserID, "usr_42")
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	mw := Middleware("secret")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
