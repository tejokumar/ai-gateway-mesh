# AI Gateway Mesh

AI Gateway Mesh is a Kubernetes-native LLM inference gateway built for production AI infrastructure.

It routes OpenAI-compatible chat completion requests across multiple model backends such as vLLM, Hugging Face TGI, Ollama, or remote APIs.

## Core Idea

This project is like a service mesh gateway for AI inference traffic.

Clients call:

```text
POST /v1/chat/completions
```

The gateway decides which model backend should serve the request.

## MVP Features

- OpenAI-compatible `/v1/chat/completions` API
- Static routing by model name
- Policy-based routing by prompt size and keywords
- Streaming response proxy
- Retry and fallback model
- Request/response metrics
- Token estimate tracking
- Prometheus metrics endpoint
- Docker Compose local dev
- Kubernetes manifests
- Istio Gateway + VirtualService
- Hugging Face Spaces demo UI

## Tech Stack

- Go for gateway/control plane
- Python/Gradio for Hugging Face demo
- Kubernetes
- Istio
- Envoy
- Prometheus
- Grafana
- OpenTelemetry later

## Local Quickstart

Run the gateway directly:

```bash
go test ./...
go run ./cmd/gateway
```

Then check the core endpoints:

```bash
curl http://localhost:8080/
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/admin/backends
curl http://localhost:8080/admin/policies
curl http://localhost:8080/metrics
```

Backend API key environment variable names are redacted from admin responses.

Run the full local demo with mock OpenAI-compatible backends:

```bash
docker compose -f deploy/docker-compose/docker-compose.yml up --build
```

Local service URLs:

- Gateway: http://localhost:8080
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3300 (`admin` / `ai-gateway`)

Grafana is provisioned with the Prometheus datasource and an `AI Gateway Mesh`
dashboard. After logging in, open:

```text
http://localhost:3300/d/ai-gateway-mesh/ai-gateway-mesh
```

Then call:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer demo-key" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "user", "content": "Explain service mesh in simple terms"}
    ],
    "stream": false
  }'
```

For streaming:

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "user", "content": "Explain kubernetes scheduling"}
    ],
    "stream": true
  }'
```

To generate a quick burst of dashboard traffic:

```bash
sh scripts/demo-traffic.sh
```

To demo fallback deterministically, start Compose with the fallback override. It forces the
small backend to return `503`, so the gateway should retry/fallback to `general-large` and
return `x-ai-gateway-fallback-used: true`.

```bash
docker compose \
  -f deploy/docker-compose/docker-compose.yml \
  -f deploy/docker-compose/docker-compose.fallback.yml \
  up --build
```

## Optional Ollama Backend

The default Compose demo uses mock OpenAI-compatible backends. To route through a
real local Ollama model, install Ollama, pull a model, and make sure Ollama is
listening on your host at port `11434`.

```bash
ollama pull llama3.2:1b
ollama serve
```

Then start the gateway with the Ollama overlay:

```bash
docker compose \
  -f deploy/docker-compose/docker-compose.yml \
  -f deploy/docker-compose/docker-compose.ollama.yml \
  up --build
```

The overlay uses `configs/gateway.ollama.yaml`, where:

- `ollama-local` points to `http://host.docker.internal:11434/v1`
- `auto` routes ordinary prompts to `ollama-local`
- coding keywords still route to the mock `coding-model`
- if Ollama is unavailable, `ollama-local` falls back to `general-small`

Try the real backend directly:

```bash
sh scripts/demo-traffic-ollama.sh
```
