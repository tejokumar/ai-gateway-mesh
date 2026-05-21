package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/openai"
)

func TestForwardRewritesModelAndUsesBackendAPIKey(t *testing.T) {
	t.Setenv("TEST_BACKEND_KEY", "secret")

	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("authorization"); got != "Bearer secret" {
			t.Fatalf("authorization = %q", got)
		}
		var req openai.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "backend-model" {
			t.Fatalf("model = %q", req.Model)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ok","choices":[]}`))
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.Client())
	resp, err := client.Forward(context.Background(), config.Backend{
		Name:      "test",
		BaseURL:   server.URL + "/v1",
		APIKeyEnv: "TEST_BACKEND_KEY",
		Model:     "backend-model",
	}, openai.ChatCompletionRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}
	defer resp.Body.Close()
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardRetriesRetryableStatus(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.Client())
	resp, err := client.Forward(context.Background(), config.Backend{
		Name:       "test",
		BaseURL:    server.URL + "/v1",
		Model:      "backend-model",
		MaxRetries: 1,
	}, openai.ChatCompletionRequest{
		Model:    "auto",
		Messages: []openai.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}
