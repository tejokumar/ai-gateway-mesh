# Architecture

## Components

### Gateway

The Gateway exposes an OpenAI-compatible API.

Responsibilities:
- request validation
- policy evaluation
- backend selection
- retry/fallback
- streaming proxy
- metrics
- tracing

### Policy Engine

The policy engine chooses a backend based on request metadata.

Inputs:
- requested model
- prompt size
- keywords
- tenant
- backend health
- future: cost and latency

### Backend Registry

Stores configured model backends.

Backend examples:
- vLLM
- Hugging Face TGI
- Ollama
- OpenAI-compatible API
- internal company model endpoint

### Metrics Layer

Emits Prometheus metrics for:
- latency
- errors
- fallback
- token estimates
- inflight requests

### Istio Layer

Istio handles:
- ingress
- mTLS
- retries if desired
- traffic splitting
- rate limiting
- authz
- observability

## Request Flow

```text
Client
  -> Istio Gateway
  -> AI Gateway Mesh
  -> Validate request
  -> Estimate tokens
  -> Evaluate policy
  -> Select backend
  -> Proxy to model
  -> Stream or return response
  -> Emit metrics
```

## Why This Project Matters

AI apps are easy to demo but hard to operate.

This project focuses on:
- reliability
- latency
- fallback
- traffic control
- model rollout
- observability
