package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  port: 8080
routing:
  default_backend: general-small
backends:
  - name: general-small
    type: openai_compatible
    base_url: http://example.test/v1
    api_key_env: SMALL_MODEL_API_KEY
    model: test-model
    timeout_ms: 1000
    max_retries: 1
policies:
  - name: small
    priority: 10
    condition:
      max_prompt_tokens: 512
    backend: general-small
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Backends[0].Name; got != "general-small" {
		t.Fatalf("backend name = %q", got)
	}
}

func TestLoadExpandsEnvironmentVariables(t *testing.T) {
	t.Setenv("TEST_BACKEND_HOSTPORT", "backend.internal:8000")
	path := filepath.Join(t.TempDir(), "gateway.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  port: 8080
routing:
  default_backend: general-small
backends:
  - name: general-small
    type: openai_compatible
    base_url: http://${TEST_BACKEND_HOSTPORT}/v1
    model: test-model
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.Backends[0].BaseURL, "http://backend.internal:8000/v1"; got != want {
		t.Fatalf("backend base URL = %q, want %q", got, want)
	}
}

func TestLoadRejectsUnknownDefaultBackend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  port: 8080
routing:
  default_backend: missing
backends:
  - name: general-small
    type: openai_compatible
    base_url: http://example.test/v1
    model: test-model
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestSanitizedBackendsDoesNotExposeAPIKeyEnv(t *testing.T) {
	cfg := Config{Backends: []Backend{{Name: "small", APIKeyEnv: "SMALL_MODEL_API_KEY"}}}
	got := cfg.SanitizedBackends()
	if got[0].APIKeyEnv == "SMALL_MODEL_API_KEY" {
		t.Fatal("SanitizedBackends exposed API key environment variable name")
	}
}

func TestBundledConfigsLoad(t *testing.T) {
	t.Setenv("SMALL_BACKEND_HOSTPORT", "small.internal:8000")
	t.Setenv("LARGE_BACKEND_HOSTPORT", "large.internal:8000")
	t.Setenv("CODER_BACKEND_HOSTPORT", "coder.internal:8000")

	for _, path := range []string{
		"../../configs/gateway.yaml",
		"../../configs/gateway.ollama.yaml",
		"../../configs/gateway.render.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := Load(path); err != nil {
				t.Fatalf("Load(%q) error = %v", path, err)
			}
		})
	}
}
