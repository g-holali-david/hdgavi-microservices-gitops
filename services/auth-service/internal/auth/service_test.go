package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginSuccess(t *testing.T) {
	svc := NewService("test-secret")

	body, _ := json.Marshal(LoginRequest{Username: "alice", Password: "password"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	svc.LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp TokenResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected refresh token")
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("expected Bearer, got %s", resp.TokenType)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	svc := NewService("test-secret")

	body, _ := json.Marshal(LoginRequest{Username: "alice", Password: "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	svc.LoginHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLoginMissingFields(t *testing.T) {
	svc := NewService("test-secret")

	body, _ := json.Marshal(LoginRequest{Username: "", Password: ""})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	svc.LoginHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyValidToken(t *testing.T) {
	svc := NewService("test-secret")

	// First login to get a token
	body, _ := json.Marshal(LoginRequest{Username: "alice", Password: "password"})
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	loginW := httptest.NewRecorder()
	svc.LoginHandler(loginW, loginReq)

	var loginResp TokenResponse
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Verify the token
	verifyReq := httptest.NewRequest(http.MethodGet, "/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
	verifyW := httptest.NewRecorder()
	svc.VerifyHandler(verifyW, verifyReq)

	if verifyW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", verifyW.Code)
	}

	var verifyResp VerifyResponse
	json.NewDecoder(verifyW.Body).Decode(&verifyResp)

	if !verifyResp.Valid {
		t.Fatal("expected valid token")
	}
	if verifyResp.Subject != "alice" {
		t.Fatalf("expected subject alice, got %s", verifyResp.Subject)
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	svc := NewService("test-secret")

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	svc.VerifyHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestVerifyMissingHeader(t *testing.T) {
	svc := NewService("test-secret")

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	w := httptest.NewRecorder()

	svc.VerifyHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRefreshToken(t *testing.T) {
	svc := NewService("test-secret")

	// Login
	loginBody, _ := json.Marshal(LoginRequest{Username: "bob", Password: "password"})
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	loginW := httptest.NewRecorder()
	svc.LoginHandler(loginW, loginReq)

	var loginResp TokenResponse
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Refresh
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": loginResp.RefreshToken})
	refreshReq := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(refreshBody))
	refreshW := httptest.NewRecorder()
	svc.RefreshHandler(refreshW, refreshReq)

	if refreshW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", refreshW.Code)
	}

	var refreshResp TokenResponse
	json.NewDecoder(refreshW.Body).Decode(&refreshResp)

	if refreshResp.AccessToken == "" {
		t.Fatal("expected new access token")
	}
}
