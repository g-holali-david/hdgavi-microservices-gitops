package main

import (
	"log"
	"net/http"
	"os"

	"github.com/g-holali-david/hdgavi-microservices-gitops/services/api-gateway/internal/gateway"
	"github.com/g-holali-david/hdgavi-microservices-gitops/services/api-gateway/internal/middleware"
)

func main() {
	port := getEnv("PORT", "8080")
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8081")
	workerServiceURL := getEnv("WORKER_SERVICE_URL", "http://worker-service:8082")

	gw := gateway.New(authServiceURL, workerServiceURL)

	mux := http.NewServeMux()

	// Health & readiness
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Public routes
	mux.HandleFunc("POST /api/v1/login", gw.ProxyAuth("/login"))
	mux.HandleFunc("POST /api/v1/refresh", gw.ProxyAuth("/refresh"))

	// Protected routes — require JWT
	mux.HandleFunc("POST /api/v1/tasks", gw.Authenticated(gw.ProxyWorker("/tasks")))
	mux.HandleFunc("GET /api/v1/tasks", gw.Authenticated(gw.ProxyWorker("/tasks")))
	mux.HandleFunc("GET /api/v1/tasks/{id}", gw.Authenticated(gw.ProxyWorker("/tasks/")))

	// Metrics
	mux.Handle("GET /metrics", middleware.MetricsHandler())

	handler := middleware.CORS(middleware.Logging(middleware.RequestMetrics(mux)))

	log.Printf("api-gateway starting on :%s", port)
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
