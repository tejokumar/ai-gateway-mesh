package gatewayhttp

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tejokumar/ai-gateway-mesh/internal/backend"
	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/metrics"
	"github.com/tejokumar/ai-gateway-mesh/internal/router"
)

func TestChatCompletionsRejectsInvalidJSON(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestIndexReturnsDiscoveryDocument(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "AI Gateway Mesh") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestChatCompletionsRejectsMissingMessages(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"auto"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func testServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.Config{
		Server:  config.ServerConfig{Port: 8080},
		Routing: config.RoutingConfig{DefaultBackend: "general-small"},
		Backends: []config.Backend{
			{Name: "general-small", Type: "openai_compatible", BaseURL: "http://example.test/v1", Model: "small"},
		},
	}
	registry := prometheus.NewRegistry()
	return New(cfg, router.New(cfg), backend.NewClient(), metrics.New(registry), slog.New(slog.NewTextHandler(testWriter{t}, nil)))
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}
