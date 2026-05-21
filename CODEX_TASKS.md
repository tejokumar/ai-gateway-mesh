# Codex Implementation Tasks

## Task 1: Bootstrap Go Service

Create a Go service under `cmd/gateway`.

Requirements:
- HTTP server on port 8080
- Graceful shutdown
- Structured JSON logging
- Health endpoints:
  - GET /healthz
  - GET /readyz
- Prometheus endpoint:
  - GET /metrics

Use:
- Go 1.22+
- chi or standard net/http
- prometheus/client_golang
- slog for logging

Acceptance:
- `go test ./...` passes
- `go run ./cmd/gateway` starts server
- `/healthz` returns 200

## Task 2: Add OpenAI-Compatible API Types

Create request/response structs for:

POST /v1/chat/completions

Support fields:
- model
- messages
- temperature
- max_tokens
- stream

Message fields:
- role
- content

Acceptance:
- JSON request parses correctly
- Invalid JSON returns 400
- Missing messages returns 400

## Task 3: Backend Registry

Create backend config loader from YAML.

Backend fields:
- name
- type
- base_url
- api_key_env
- model
- timeout_ms
- max_retries
- fallback

Acceptance:
- config loads on startup
- invalid config fails fast
- `/admin/backends` returns configured backends without secrets

## Task 4: Router

Implement routing engine.

Routing inputs:
- requested model
- estimated prompt tokens
- message content
- tenant header

Routing behavior:
- if model is explicit and backend exists, use it
- if model is `auto`, evaluate policies in order
- fallback to default backend

Acceptance:
- unit tests for routing
- deterministic policy ordering

## Task 5: OpenAI-Compatible Backend Client

Implement backend client that forwards requests to `{base_url}/chat/completions`.

Behavior:
- replace request model with backend model
- forward Authorization header using backend API key
- preserve stream value
- timeout support
- retry support

Acceptance:
- mock backend test passes
- non-streaming response works

## Task 6: Streaming Proxy

If `stream=true`, proxy SSE response from backend to client.

Requirements:
- preserve `text/event-stream`
- flush chunks immediately
- handle client disconnect
- record metrics

Acceptance:
- test with mock SSE backend
- curl receives streamed chunks

## Task 7: Fallback

When backend returns:
- 429
- 500
- 502
- 503
- timeout

Then route to fallback backend if configured.

Acceptance:
- fallback metric increments
- response includes header:
  - `x-ai-gateway-backend`
  - `x-ai-gateway-fallback-used`

## Task 8: Metrics

Add Prometheus metrics:
- request count by route/backend/status
- latency histogram by backend
- fallback count
- backend error count
- inflight requests
- estimated prompt tokens

Acceptance:
- `/metrics` shows all metrics
- labels do not include high-cardinality prompt text

## Task 9: Docker Compose

Create:
- gateway service
- mock-vllm-small service
- mock-vllm-large service
- mock-vllm-coder service
- prometheus service
- grafana service

Acceptance:
- `docker compose -f deploy/docker-compose/docker-compose.yml up --build`
- gateway reachable on localhost:8080
- prometheus reachable on localhost:9090
- grafana reachable on localhost:3000

## Task 10: Kubernetes Manifests

Create:
- Namespace
- Deployment
- Service
- ConfigMap
- Secret placeholder
- ServiceAccount
- HPA optional

Acceptance:
- `kubectl apply -k deploy/k8s`
- gateway pod starts

## Task 11: Istio Integration

Create:
- Gateway
- VirtualService
- DestinationRule
- optional EnvoyFilter for local rate limit

Acceptance:
- traffic enters through Istio ingress gateway
- routing to gateway service works

## Task 12: Hugging Face Spaces Demo

Create a Gradio app in `demo/huggingface-space`.

UI:
- Prompt input
- Model selector: auto, small, large, coding
- Streaming checkbox
- Output text
- Shows selected backend returned by response header

Acceptance:
- can run locally with `python app.py`
- deployable as Hugging Face Space
