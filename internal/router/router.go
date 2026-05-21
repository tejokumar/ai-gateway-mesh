package router

import (
	"sort"
	"strings"

	"github.com/tejokumar/ai-gateway-mesh/internal/config"
	"github.com/tejokumar/ai-gateway-mesh/internal/openai"
)

type Input struct {
	RequestedModel        string
	EstimatedPromptTokens int
	Messages              []openai.Message
	Tenant                string
}

type Result struct {
	Backend config.Backend
	Policy  string
}

type Router struct {
	cfg      config.Config
	policies []config.Policy
}

func New(cfg config.Config) *Router {
	policies := append([]config.Policy(nil), cfg.Policies...)
	sort.SliceStable(policies, func(i, j int) bool {
		return policies[i].Priority < policies[j].Priority
	})
	return &Router{cfg: cfg, policies: policies}
}

func (r *Router) Route(in Input) (Result, bool) {
	if in.RequestedModel != "" && in.RequestedModel != "auto" {
		if backend, ok := r.cfg.BackendByName(in.RequestedModel); ok {
			return Result{Backend: backend, Policy: "explicit-model"}, true
		}
	}

	for _, policy := range r.policies {
		if matches(policy.Condition, in) {
			backend, ok := r.cfg.BackendByName(policy.Backend)
			return Result{Backend: backend, Policy: policy.Name}, ok
		}
	}

	backend, ok := r.cfg.BackendByName(r.cfg.Routing.DefaultBackend)
	return Result{Backend: backend, Policy: "default"}, ok
}

func matches(condition config.Condition, in Input) bool {
	if condition.MinPromptTokens > 0 && in.EstimatedPromptTokens < condition.MinPromptTokens {
		return false
	}
	if condition.MaxPromptTokens > 0 && in.EstimatedPromptTokens > condition.MaxPromptTokens {
		return false
	}
	if condition.Tenant != "" && condition.Tenant != in.Tenant {
		return false
	}
	if len(condition.ContainsAny) > 0 {
		content := strings.ToLower(joinContent(in.Messages))
		for _, term := range condition.ContainsAny {
			if strings.Contains(content, strings.ToLower(term)) {
				return true
			}
		}
		return false
	}
	return true
}

func joinContent(messages []openai.Message) string {
	var b strings.Builder
	for _, message := range messages {
		b.WriteString(message.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

func EstimatePromptTokens(messages []openai.Message) int {
	total := 0
	for _, message := range messages {
		total += len(strings.Fields(message.Content))
	}
	if total == 0 && len(messages) > 0 {
		return 1
	}
	return total
}
