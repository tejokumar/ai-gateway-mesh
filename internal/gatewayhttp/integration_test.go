package gatewayhttp

import (
	"encoding/json"
	"io"
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

func TestChatCompletionsProxiesNonStreamingResponse(t *testing.T) {
	backendServer := newJSONBackend(t, "general-small", http.StatusOK)
	defer backendServer.Close()

	gateway := httptest.NewServer(newIntegrationServer(t, integrationConfig(backendServer.URL, "")).Handler())
	defer gateway.Close()

	resp, body := postChat(t, gateway.URL, `{
		"model": "auto",
		"messages": [{"role": "user", "content": "hello"}],
		"stream": false
	}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("x-ai-gateway-backend"); got != "general-small" {
		t.Fatalf("x-ai-gateway-backend = %q", got)
	}
	if got := resp.Header.Get("x-ai-gateway-fallback-used"); got != "false" {
		t.Fatalf("x-ai-gateway-fallback-used = %q", got)
	}
	if !strings.Contains(body, `"model":"small-model"`) {
		t.Fatalf("body did not contain rewritten model: %s", body)
	}
}

func TestChatCompletionsStreamsSSE(t *testing.T) {
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("content-type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer backendServer.Close()

	gateway := httptest.NewServer(newIntegrationServer(t, integrationConfig(backendServer.URL, "")).Handler())
	defer gateway.Close()

	resp, body := postChat(t, gateway.URL, `{
		"model": "auto",
		"messages": [{"role": "user", "content": "hello"}],
		"stream": true
	}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("content-type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("content-type = %q", got)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("stream body missing done marker: %s", body)
	}
}

func TestChatCompletionsFallsBackOnRetryableBackendStatus(t *testing.T) {
	primary := newJSONBackend(t, "general-small", http.StatusServiceUnavailable)
	defer primary.Close()
	fallback := newJSONBackend(t, "general-large", http.StatusOK)
	defer fallback.Close()

	cfg := integrationConfig(primary.URL, fallback.URL)
	gateway := httptest.NewServer(newIntegrationServer(t, cfg).Handler())
	defer gateway.Close()

	resp, body := postChat(t, gateway.URL, `{
		"model": "auto",
		"messages": [{"role": "user", "content": "hello"}],
		"stream": false
	}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("x-ai-gateway-backend"); got != "general-large" {
		t.Fatalf("x-ai-gateway-backend = %q", got)
	}
	if got := resp.Header.Get("x-ai-gateway-fallback-used"); got != "true" {
		t.Fatalf("x-ai-gateway-fallback-used = %q", got)
	}
	if !strings.Contains(body, "general-large") {
		t.Fatalf("body did not come from fallback: %s", body)
	}
}

func newIntegrationServer(t *testing.T, cfg config.Config) *Server {
	t.Helper()
	registry := prometheus.NewRegistry()
	return New(cfg, router.New(cfg), backend.NewClient(), metrics.New(registry), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func integrationConfig(primaryURL, fallbackURL string) config.Config {
	backends := []config.Backend{
		{
			Name:       "general-small",
			Type:       "openai_compatible",
			BaseURL:    primaryURL + "/v1",
			Model:      "small-model",
			MaxRetries: 0,
		},
	}
	if fallbackURL != "" {
		backends[0].Fallback = "general-large"
		backends = append(backends, config.Backend{
			Name:       "general-large",
			Type:       "openai_compatible",
			BaseURL:    fallbackURL + "/v1",
			Model:      "large-model",
			MaxRetries: 0,
		})
	}
	return config.Config{
		Server:   config.ServerConfig{Port: 8080},
		Routing:  config.RoutingConfig{DefaultBackend: "general-small"},
		Backends: backends,
		Policies: []config.Policy{
			{
				Name:     "small",
				Priority: 10,
				Condition: config.Condition{
					MaxPromptTokens: 512,
				},
				Backend: "general-small",
			},
		},
	}
}

func newJSONBackend(t *testing.T, name string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if status != http.StatusOK {
			http.Error(w, "backend unavailable", status)
			return
		}
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 0,
			"model":   req.Model,
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]string{"role": "assistant", "content": "response from " + name},
					"finish_reason": "stop",
				},
			},
		})
	}))
}

func postChat(t *testing.T, baseURL, body string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Post(baseURL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body = io.NopCloser(strings.NewReader(string(raw)))
	return resp, string(raw)
}
