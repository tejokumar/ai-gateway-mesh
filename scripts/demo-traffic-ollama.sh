#!/usr/bin/env sh
set -eu

BASE_URL="${AI_GATEWAY_BASE_URL:-http://127.0.0.1:8080}"

echo "Sending Ollama demo traffic to $BASE_URL"

curl -sS "$BASE_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"ollama-local","messages":[{"role":"user","content":"Explain service mesh for AI inference in two short sentences."}],"max_tokens":96,"stream":false}' \
  | python3 -m json.tool

echo
echo "Streaming through Ollama backend:"

curl -N "$BASE_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"ollama-local","messages":[{"role":"user","content":"Count from one to five, comma-separated."}],"max_tokens":32,"stream":true}'
