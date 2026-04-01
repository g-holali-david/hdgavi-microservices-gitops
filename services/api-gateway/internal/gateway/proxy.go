package gateway

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Gateway struct {
	authURL   string
	workerURL string
	client    *http.Client
}

func New(authURL, workerURL string) *Gateway {
	return &Gateway{
		authURL:   authURL,
		workerURL: workerURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ProxyAuth returns a handler that proxies requests to the auth service.
func (g *Gateway) ProxyAuth(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g.proxyTo(w, r, g.authURL+path)
	}
}

// ProxyWorker returns a handler function that proxies to the worker service.
func (g *Gateway) ProxyWorker(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetPath := path
		// Append path parameters if present
		if id := r.PathValue("id"); id != "" {
			targetPath = path + id
		}
		g.proxyTo(w, r, g.workerURL+targetPath)
	}
}

// Authenticated wraps a handler with JWT validation via the auth service.
func (g *Gateway) Authenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}

		// Verify token with auth service
		verifyReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, g.authURL+"/verify", nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		verifyReq.Header.Set("Authorization", authHeader)

		resp, err := g.client.Do(verifyReq)
		if err != nil {
			log.Printf("auth service error: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		// Parse verified claims and forward as headers
		var verifyResp struct {
			Subject string `json:"subject"`
		}
		json.NewDecoder(resp.Body).Decode(&verifyResp)

		r.Header.Set("X-User-ID", verifyResp.Subject)

		next(w, r)
	}
}

func (g *Gateway) proxyTo(w http.ResponseWriter, r *http.Request, targetURL string) {
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create proxy request"})
		return
	}

	// Copy relevant headers
	for _, h := range []string{"Content-Type", "Authorization", "X-User-ID", "X-Request-ID"} {
		if v := r.Header.Get(h); v != "" {
			proxyReq.Header.Set(h, v)
		}
	}

	resp, err := g.client.Do(proxyReq)
	if err != nil {
		log.Printf("proxy error to %s: %v", targetURL, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		if strings.HasPrefix(k, "Content-") || k == "X-Request-ID" {
			w.Header().Set(k, v[0])
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
