package main

import (
	"log"
	"net/http"
	"os"

	"github.com/g-holali-david/hdgavi-microservices-gitops/services/auth-service/internal/auth"
	"github.com/g-holali-david/hdgavi-microservices-gitops/services/auth-service/internal/middleware"
)

func main() {
	port := getEnv("PORT", "8081")
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-change-me")

	svc := auth.NewService(jwtSecret)

	mux := http.NewServeMux()

	// Health & readiness
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"auth-service"}`))
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Auth endpoints
	mux.HandleFunc("POST /login", svc.LoginHandler)
	mux.HandleFunc("GET /verify", svc.VerifyHandler)
	mux.HandleFunc("POST /refresh", svc.RefreshHandler)

	// Metrics
	mux.Handle("GET /metrics", middleware.MetricsHandler())

	handler := middleware.Logging(middleware.RequestMetrics(mux))

	log.Printf("auth-service starting on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
