package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig  `yaml:"server" json:"server"`
	Routing  RoutingConfig `yaml:"routing" json:"routing"`
	Backends []Backend     `yaml:"backends" json:"backends"`
	Policies []Policy      `yaml:"policies" json:"policies"`
}

type ServerConfig struct {
	Port           int `yaml:"port" json:"port"`
	ReadTimeoutMS  int `yaml:"read_timeout_ms" json:"read_timeout_ms"`
	WriteTimeoutMS int `yaml:"write_timeout_ms" json:"write_timeout_ms"`
}

type RoutingConfig struct {
	DefaultBackend string `yaml:"default_backend" json:"default_backend"`
}

type Backend struct {
	Name       string `yaml:"name" json:"name"`
	Type       string `yaml:"type" json:"type"`
	BaseURL    string `yaml:"base_url" json:"base_url"`
	APIKeyEnv  string `yaml:"api_key_env" json:"api_key_env,omitempty"`
	Model      string `yaml:"model" json:"model"`
	TimeoutMS  int    `yaml:"timeout_ms" json:"timeout_ms"`
	MaxRetries int    `yaml:"max_retries" json:"max_retries"`
	Fallback   string `yaml:"fallback,omitempty" json:"fallback,omitempty"`
}

type Policy struct {
	Name      string    `yaml:"name" json:"name"`
	Priority  int       `yaml:"priority" json:"priority"`
	Condition Condition `yaml:"condition" json:"condition"`
	Backend   string    `yaml:"backend" json:"backend"`
}

type Condition struct {
	MaxPromptTokens int      `yaml:"max_prompt_tokens,omitempty" json:"max_prompt_tokens,omitempty"`
	MinPromptTokens int      `yaml:"min_prompt_tokens,omitempty" json:"min_prompt_tokens,omitempty"`
	ContainsAny     []string `yaml:"contains_any,omitempty" json:"contains_any,omitempty"`
	Tenant          string   `yaml:"tenant,omitempty" json:"tenant,omitempty"`
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Routing.DefaultBackend == "" {
		return errors.New("routing.default_backend is required")
	}
	if len(c.Backends) == 0 {
		return errors.New("at least one backend is required")
	}

	backends := map[string]Backend{}
	for _, backend := range c.Backends {
		if backend.Name == "" {
			return errors.New("backend.name is required")
		}
		if _, exists := backends[backend.Name]; exists {
			return fmt.Errorf("duplicate backend %q", backend.Name)
		}
		if backend.Type != "openai_compatible" {
			return fmt.Errorf("backend %q has unsupported type %q", backend.Name, backend.Type)
		}
		if backend.BaseURL == "" {
			return fmt.Errorf("backend %q base_url is required", backend.Name)
		}
		if backend.Model == "" {
			return fmt.Errorf("backend %q model is required", backend.Name)
		}
		if backend.TimeoutMS < 0 || backend.MaxRetries < 0 {
			return fmt.Errorf("backend %q timeout_ms and max_retries must be non-negative", backend.Name)
		}
		backends[backend.Name] = backend
	}

	if _, ok := backends[c.Routing.DefaultBackend]; !ok {
		return fmt.Errorf("default backend %q is not configured", c.Routing.DefaultBackend)
	}
	for _, backend := range c.Backends {
		if backend.Fallback != "" {
			if _, ok := backends[backend.Fallback]; !ok {
				return fmt.Errorf("backend %q fallback %q is not configured", backend.Name, backend.Fallback)
			}
		}
	}
	for _, policy := range c.Policies {
		if policy.Name == "" {
			return errors.New("policy.name is required")
		}
		if _, ok := backends[policy.Backend]; !ok {
			return fmt.Errorf("policy %q backend %q is not configured", policy.Name, policy.Backend)
		}
	}
	return nil
}

func (c Config) BackendByName(name string) (Backend, bool) {
	for _, backend := range c.Backends {
		if backend.Name == name {
			return backend, true
		}
	}
	return Backend{}, false
}

func (c Config) SanitizedBackends() []Backend {
	out := slices.Clone(c.Backends)
	for i := range out {
		out[i].APIKeyEnv = redactEnvName(out[i].APIKeyEnv)
	}
	return out
}

func redactEnvName(value string) string {
	if value == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(value), "key") {
		return "configured"
	}
	return value
}
