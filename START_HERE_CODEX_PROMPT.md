# Initial Codex Prompt

You are building a production-quality open-source portfolio project called **AI Gateway Mesh**.

## Goal

Build a Kubernetes-native LLM inference gateway in Go that exposes an OpenAI-compatible `/v1/chat/completions` endpoint and routes requests across multiple OpenAI-compatible model backends such as vLLM, TGI, Ollama, or mock servers.

This project is for an infrastructure engineer with deep service mesh experience. Prioritize clean architecture, production-quality code, tests, observability, Kubernetes, and Istio integration.

## Start Here

First inspect these files:

- `README.md`
- `PROJECT_SPEC.md`
- `CODEX_TASKS.md`
- `configs/gateway.yaml`

Then implement the project incrementally.

## Core Requirements

1. Go service under `cmd/gateway`
2. HTTP server on port `8080`
3. Endpoints:
   - `GET /healthz`
   - `GET /readyz`
   - `GET /metrics`
   - `GET /admin/backends`
   - `GET /admin/policies`
   - `POST /v1/chat/completions`
4. YAML config loader
5. Backend registry
6. Policy-based router
7. OpenAI-compatible backend proxy
8. Non-streaming response support first
9. Streaming SSE support second
10. Retry and fallback support
11. Prometheus metrics
12. Dockerfile
13. Docker Compose with mock OpenAI-compatible model servers
14. Kubernetes manifests
15. Istio Gateway and VirtualService
16. Hugging Face Gradio demo under `demo/huggingface-space`

## Engineering Standards

- Use idiomatic Go.
- Keep packages small and testable.
- Add unit tests for router, config, and backend client.
- Avoid high-cardinality metric labels.
- Never log prompt content by default.
- Do not expose API keys in admin endpoints.
- Use structured logging.
- Add clear README instructions.
- Keep implementation simple first, then add advanced features.

## Execution Order

Start by creating the full repo skeleton and implementing **Task 1** from `CODEX_TASKS.md`.

After Task 1 is complete and tests pass, continue task by task.

Do not jump to advanced multi-cluster or EnvoyFilter features until the local Docker Compose demo works.
