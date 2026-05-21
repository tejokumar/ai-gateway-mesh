# AI Gateway Mesh

[![CI](https://github.com/tejokumar/ai-gateway-mesh/actions/workflows/ci.yml/badge.svg)](https://github.com/tejokumar/ai-gateway-mesh/actions/workflows/ci.yml)

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
make test
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
make compose-up
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
make demo-traffic
```

To demo fallback deterministically, start Compose with the fallback override. It forces the
small backend to return `503`, so the gateway should retry/fallback to `general-large` and
return `x-ai-gateway-fallback-used: true`.

```bash
make compose-fallback
```

## Optional Ollama Backend

The default Compose demo uses mock OpenAI-compatible backends. To route through a
real local Ollama model, install Ollama, pull a model, and make sure Ollama is
listening on your host at port `11434`.

```bash
make ollama-pull
ollama serve
```

Then start the gateway with the Ollama overlay:

```bash
make compose-ollama
```

The overlay uses `configs/gateway.ollama.yaml`, where:

- `ollama-local` points to `http://host.docker.internal:11434/v1`
- `auto` routes ordinary prompts to `ollama-local`
- coding keywords still route to the mock `coding-model`
- if Ollama is unavailable, `ollama-local` falls back to `general-small`

Try the real backend directly:

```bash
make demo-traffic-ollama
```

Stop the local stack:

```bash
make compose-down
```

## Kubernetes Demo

The Kubernetes manifests are self-contained for a local mock-backend demo. They
deploy the gateway plus three mock OpenAI-compatible backend services:

- `vllm-small`
- `vllm-large`
- `vllm-coder`

Build images for a local cluster:

```bash
make docker-build
make mock-docker-build
```

If you use `kind`, load the images into the cluster:

```bash
kind load docker-image ai-gateway-mesh:local
kind load docker-image ai-gateway-mock-openai:latest
```

Validate and apply manifests:

```bash
make k8s-validate
kubectl apply -k deploy/k8s
kubectl -n ai-gateway-mesh port-forward svc/ai-gateway 8080:8080
```

Then call the gateway:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","messages":[{"role":"user","content":"Explain kubernetes service mesh"}],"stream":false}'
```

Or run the local kind workflow through Make:

```bash
make k8s-up
make k8s-smoke
make k8s-port-forward
```

For a disposable end-to-end demo that creates the kind cluster, deploys the
gateway and mock backends, sends smoke-test traffic, stops the temporary
port-forward, and deletes the cluster:

```bash
make k8s-demo
```

To delete the local kind cluster manually:

```bash
make k8s-down
```

### Kubernetes Observability

Deploy Prometheus and Grafana into the same local Kubernetes namespace:

```bash
make k8s-observability-up
make k8s-observability-smoke
make k8s-grafana-port-forward
```

Open Grafana at `http://127.0.0.1:3301`.

Login:

- username: `admin`
- password: `ai-gateway`

Prometheus can be forwarded separately:

```bash
make k8s-prometheus-port-forward
```

Then open `http://127.0.0.1:19090`.

To remove only Prometheus and Grafana while keeping the gateway running:

```bash
make k8s-observability-down
```

For a disposable end-to-end demo with observability:

```bash
make k8s-demo-observability
```

## Istio Demo

Istio is optional and should be applied after the Kubernetes demo works.

```bash
make k8s-up
make istio-up
make istio-smoke
```

The Istio manifests expose the gateway through `ai-gateway.local` using the
cluster's Istio ingress gateway.

To keep the ingress open locally:

```bash
make istio-port-forward
curl -H "Host: ai-gateway.local" http://127.0.0.1:18081/healthz
```

To remove only Istio while keeping the Kubernetes gateway running:

```bash
make istio-down
```

For a disposable end-to-end Istio demo:

```bash
make k8s-demo-istio
```

## Public Demo Deployment

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/tejokumar/ai-gateway-mesh)

The repository includes a Render Blueprint at `render.yaml` for a public demo
gateway plus three private mock OpenAI-compatible backend services.

Render will deploy:

- `ai-gateway-mesh` as a public Docker web service
- `ai-gateway-mock-small` as a private service
- `ai-gateway-mock-large` as a private service
- `ai-gateway-mock-coder` as a private service

The gateway uses `configs/gateway.render.yaml`, which expands backend private
hostnames from Render environment variables such as `SMALL_BACKEND_HOSTPORT`.

After the Render Blueprint deploys, set the Hugging Face Space variables:

```text
AI_GATEWAY_BASE_URL=https://<your-render-service>.onrender.com
AI_GATEWAY_API_KEY=demo-key
```

Then the Gradio demo in `demo/huggingface-space` can call the public gateway.
