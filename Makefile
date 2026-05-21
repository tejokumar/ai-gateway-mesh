SHELL := /bin/sh

COMPOSE := docker compose -f deploy/docker-compose/docker-compose.yml
COMPOSE_FALLBACK := $(COMPOSE) -f deploy/docker-compose/docker-compose.fallback.yml
COMPOSE_OLLAMA := $(COMPOSE) -f deploy/docker-compose/docker-compose.ollama.yml
IMAGE ?= ai-gateway-mesh:local
GOCACHE ?= $(CURDIR)/.gocache

export GOCACHE

.PHONY: help test vet validate docker-build mock-docker-build compose-config k8s-validate compose-up compose-down compose-fallback compose-ollama demo-traffic demo-traffic-ollama ollama-pull

help:
	@echo "AI Gateway Mesh targets:"
	@echo "  make test                  Run Go tests"
	@echo "  make vet                   Run go vet"
	@echo "  make validate              Validate Go, Python, JSON, and Compose configs"
	@echo "  make docker-build          Build the gateway Docker image"
	@echo "  make mock-docker-build     Build the mock OpenAI backend image"
	@echo "  make k8s-validate          Render Kubernetes and Istio manifests"
	@echo "  make compose-up            Start mock backend demo stack"
	@echo "  make compose-fallback      Start stack with forced fallback demo"
	@echo "  make compose-ollama        Start stack with Ollama overlay"
	@echo "  make compose-down          Stop Compose stack"
	@echo "  make demo-traffic          Send mock demo traffic"
	@echo "  make demo-traffic-ollama   Send Ollama demo traffic"
	@echo "  make ollama-pull           Pull the default local Ollama model"

test:
	go test ./...

vet:
	go vet ./...

validate: test vet compose-config k8s-validate
	python3 -m py_compile tests/mock-openai-server/server.py demo/huggingface-space/app.py
	python3 -m json.tool deploy/docker-compose/grafana/dashboards/ai-gateway-mesh.json >/tmp/ai-gateway-mesh-dashboard.json

docker-build:
	docker build -t $(IMAGE) .

mock-docker-build:
	docker build -t ai-gateway-mock-openai:latest tests/mock-openai-server

compose-config:
	$(COMPOSE) config >/tmp/compose-default.yml
	$(COMPOSE_FALLBACK) config >/tmp/compose-fallback.yml
	$(COMPOSE_OLLAMA) config >/tmp/compose-ollama.yml

k8s-validate:
	@if command -v kubectl >/dev/null 2>&1; then \
		kubectl kustomize deploy/k8s >/tmp/ai-gateway-k8s.yaml; \
		kubectl kustomize deploy/istio >/tmp/ai-gateway-istio.yaml; \
	else \
		echo "kubectl not found; skipping Kubernetes manifest render validation"; \
	fi

compose-up:
	$(COMPOSE) up --build

compose-down:
	$(COMPOSE) down

compose-fallback:
	$(COMPOSE_FALLBACK) up --build

compose-ollama:
	$(COMPOSE_OLLAMA) up --build

demo-traffic:
	sh scripts/demo-traffic.sh

demo-traffic-ollama:
	sh scripts/demo-traffic-ollama.sh

ollama-pull:
	ollama pull llama3.2:1b
