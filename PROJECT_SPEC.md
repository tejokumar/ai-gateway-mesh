# Project Specification: AI Gateway Mesh

## Problem

Modern AI systems often call multiple LLMs. Production systems need routing, fallback, observability, cost control, rate limiting, and rollout safety.

Most demo AI apps hide this infrastructure complexity.

AI Gateway Mesh exposes the infrastructure layer.

## Main Use Cases

1. Route requests to the right model.
2. Fall back when a model fails.
3. Control cost per request.
4. Canary new models safely.
5. Monitor latency, errors, token usage, and saturation.
6. Enforce tenant limits.
7. Run in Kubernetes with Istio.

## MVP Scope

### API

Implement:

```text
POST /v1/chat/completions
GET /healthz
GET /readyz
GET /metrics
GET /admin/backends
GET /admin/policies
```

### Routing

Support route policies:

```yaml
routes:
  - name: small-prompt-route
    condition:
      max_prompt_tokens: 512
    backend: general-small

  - name: coding-route
    condition:
      contains_any:
        - code
        - kubernetes
        - golang
    backend: coding-model
```

### Backends

Support OpenAI-compatible backends:

```yaml
backends:
  - name: local-vllm
    type: openai_compatible
    base_url: http://vllm:8000/v1
    api_key_env: VLLM_API_KEY
    model: mistral
    timeout_ms: 60000
    max_retries: 1
```

### Fallback

If primary backend fails:
- retry once
- route to fallback backend
- emit metric

### Metrics

Expose Prometheus metrics:

```text
ai_gateway_requests_total
ai_gateway_request_duration_seconds
ai_gateway_backend_errors_total
ai_gateway_fallback_total
ai_gateway_estimated_prompt_tokens
ai_gateway_estimated_completion_tokens
ai_gateway_inflight_requests
```

## Non-Goals For MVP

- Full auth system
- Real billing
- Fine-tuning
- Vector database
- Complex agents
- Production-grade tokenizer accuracy
