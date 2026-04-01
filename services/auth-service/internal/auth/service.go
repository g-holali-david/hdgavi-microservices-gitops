package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	jwtSecret []byte
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type VerifyResponse struct {
	Valid    bool   `json:"valid"`
	Subject string `json:"subject,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role,omitempty"`
}

func NewService(secret string) *Service {
	return &Service{jwtSecret: []byte(secret)}
}

// LoginHandler authenticates a user and returns JWT tokens.
func (s *Service) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	// In production, validate against database
	// For demo: accept any user with password "password"
	if !validateCredentials(req.Username, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	accessToken, err := s.generateToken(req.Username, "user", 15*time.Minute)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	refreshToken, err := s.generateToken(req.Username, "user", 7*24*time.Hour)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate refresh token"})
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
		TokenType:    "Bearer",
	})
}

// VerifyHandler validates a JWT token from the Authorization header.
func (s *Service) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	tokenStr := extractBearerToken(r)
	if tokenStr == "" {
		writeJSON(w, http.StatusUnauthorized, VerifyResponse{Valid: false, Error: "missing authorization header"})
		return
	}

	claims, err := s.parseToken(tokenStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, VerifyResponse{Valid: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, VerifyResponse{
		Valid:   true,
		Subject: claims.Subject,
	})
}

// RefreshHandler issues a new access token from a valid refresh token.
func (s *Service) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	claims, err := s.parseToken(body.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}

	accessToken, err := s.generateToken(claims.Subject, claims.Role, 15*time.Minute)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   900,
		TokenType:   "Bearer",
	})
}

func (s *Service) generateToken(subject, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "auth-service",
		},
		Role: role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) parseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func validateCredentials(username, password string) bool {
	// Demo-only: in production, check against a database with hashed passwords
	return password == "password"
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
