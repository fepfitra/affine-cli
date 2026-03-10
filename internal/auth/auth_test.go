package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignInSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/auth/sign-in" {
			t.Errorf("Path = %q, want /api/auth/sign-in", r.URL.Path)
		}

		var req SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Email != "test@example.com" {
			t.Errorf("Email = %q", req.Email)
		}
		if req.Password != "password123" {
			t.Errorf("Password = %q", req.Password)
		}

		http.SetCookie(w, &http.Cookie{Name: "affine_session", Value: "sess-abc"})
		http.SetCookie(w, &http.Cookie{Name: "affine_csrf", Value: "csrf-xyz"})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	cookie, err := SignIn(context.Background(), server.URL, "test@example.com", "password123")
	if err != nil {
		t.Fatalf("SignIn error: %v", err)
	}

	if cookie == "" {
		t.Fatal("cookie is empty")
	}
	// Should contain both cookies
	if !contains(cookie, "affine_session=sess-abc") {
		t.Errorf("cookie missing affine_session: %q", cookie)
	}
	if !contains(cookie, "affine_csrf=csrf-xyz") {
		t.Errorf("cookie missing affine_csrf: %q", cookie)
	}
}

func TestSignInHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid credentials"}`))
	}))
	defer server.Close()

	_, err := SignIn(context.Background(), server.URL, "bad@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSignInNoCookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	_, err := SignIn(context.Background(), server.URL, "test@example.com", "pass")
	if err == nil {
		t.Fatal("expected error for no cookies, got nil")
	}
}

func TestSignInBadURL(t *testing.T) {
	_, err := SignIn(context.Background(), "://bad-url", "a@b.com", "pass")
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
