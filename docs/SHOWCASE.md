# Portfolio Showcase

## Project Title

AI Gateway Mesh: Kubernetes-native LLM inference gateway with Istio/Envoy integration

## One-Line Pitch

Built a production-style AI inference gateway that routes, observes, rate-limits, and safely rolls out LLM traffic across multiple model backends.

## Resume Bullet

Built a Kubernetes-native AI inference gateway in Go using Istio/Envoy, OpenAI-compatible model backends, streaming proxying, fallback routing, Prometheus metrics, and policy-based model selection for production LLM workloads.

## Demo Script

1. Show Hugging Face UI.
2. Send a simple prompt.
3. Show backend selected as `general-small`.
4. Send a coding/Istio prompt.
5. Show backend selected as `coding-model`.
6. Kill coding backend.
7. Retry request.
8. Show fallback to `general-large`.
9. Open Grafana dashboard.
10. Show request count, latency, fallback, and backend errors.

## What This Demonstrates

- AI infra knowledge
- Kubernetes experience
- Service mesh experience
- LLM serving understanding
- Observability
- Reliability engineering
- Platform engineering maturity
