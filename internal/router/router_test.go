package router

import (
	"testing"

	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/openai"
)

func TestRouteExplicitBackendByModelName(t *testing.T) {
	r := New(testConfig())
	got, ok := r.Route(Input{RequestedModel: "coding-model"})
	if !ok {
		t.Fatal("Route() ok = false")
	}
	if got.Backend.Name != "coding-model" || got.Policy != "explicit-model" {
		t.Fatalf("Route() = %+v", got)
	}
}

func TestRouteAutoEvaluatesPoliciesByPriority(t *testing.T) {
	r := New(testConfig())
	got, ok := r.Route(Input{
		RequestedModel:        "auto",
		EstimatedPromptTokens: 100,
		Messages:              []openai.Message{{Role: "user", Content: "please write golang code"}},
	})
	if !ok {
		t.Fatal("Route() ok = false")
	}
	if got.Backend.Name != "coding-model" {
		t.Fatalf("backend = %q, want coding-model", got.Backend.Name)
	}
}

func TestRouteAutoFallsBackToDefault(t *testing.T) {
	r := New(testConfig())
	got, ok := r.Route(Input{
		RequestedModel:        "auto",
		EstimatedPromptTokens: 900,
		Messages:              []openai.Message{{Role: "user", Content: "hello"}},
	})
	if !ok {
		t.Fatal("Route() ok = false")
	}
	if got.Backend.Name != "general-small" || got.Policy != "default" {
		t.Fatalf("Route() = %+v", got)
	}
}

func testConfig() config.Config {
	return config.Config{
		Routing: config.RoutingConfig{DefaultBackend: "general-small"},
		Backends: []config.Backend{
			{Name: "general-small", Type: "openai_compatible", BaseURL: "http://small/v1", Model: "small"},
			{Name: "coding-model", Type: "openai_compatible", BaseURL: "http://coder/v1", Model: "coder"},
		},
		Policies: []config.Policy{
			{
				Name:     "small",
				Priority: 20,
				Condition: config.Condition{
					MaxPromptTokens: 512,
				},
				Backend: "general-small",
			},
			{
				Name:     "coding",
				Priority: 10,
				Condition: config.Condition{
					ContainsAny: []string{"golang", "kubernetes"},
				},
				Backend: "coding-model",
			},
		},
	}
}
