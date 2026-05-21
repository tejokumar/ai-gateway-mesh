import os
import json
import time
import requests
import gradio as gr

BASE_URL = os.getenv("AI_GATEWAY_BASE_URL", "").rstrip("/")
API_KEY = os.getenv("AI_GATEWAY_API_KEY", "demo-key")
DEMO_MODE = os.getenv("AI_GATEWAY_DEMO_MODE", "").lower() in {"1", "true", "yes"} or not BASE_URL
CODING_KEYWORDS = {"code", "golang", "kubernetes", "istio", "envoy", "python"}


def select_demo_backend(prompt, model):
    if model != "auto":
        return model
    lowered = prompt.lower()
    if any(keyword in lowered for keyword in CODING_KEYWORDS):
        return "coding-model"
    if len(prompt.split()) > 80:
        return "general-large"
    return "general-small"


def demo_chat(prompt, model, stream):
    backend = select_demo_backend(prompt, model)
    response = (
        f"Demo response from {backend}.\n\n"
        "This Space is running without a public AI Gateway URL, so it is simulating "
        "the gateway routing decision. Set AI_GATEWAY_BASE_URL to connect this UI "
        "to a live gateway deployment."
    )
    if stream:
        content = ""
        for token in response.split(" "):
            content = f"{content} {token}".strip()
            yield content, backend, "false"
            time.sleep(0.03)
        return
    yield response, backend, "false"

def chat(prompt, model, stream):
    if DEMO_MODE:
        yield from demo_chat(prompt, model, stream)
        return

    url = f"{BASE_URL}/v1/chat/completions"
    payload = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0.2,
        "stream": stream,
    }
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json",
    }
    try:
        resp = requests.post(url, json=payload, headers=headers, timeout=120, stream=stream)
        backend = resp.headers.get("x-ai-gateway-backend", "unknown")
        fallback = resp.headers.get("x-ai-gateway-fallback-used", "false")

        if resp.status_code != 200:
            yield f"Error {resp.status_code}: {resp.text}", backend, fallback
            return

        if stream:
            content = ""
            for line in resp.iter_lines(decode_unicode=True):
                if not line or not line.startswith("data: "):
                    continue
                data = line.removeprefix("data: ")
                if data == "[DONE]":
                    break
                chunk = json.loads(data)
                delta = chunk.get("choices", [{}])[0].get("delta", {})
                content += delta.get("content", "")
                yield content, backend, fallback
            return

        data = resp.json()
        content = data.get("choices", [{}])[0].get("message", {}).get("content", "")
        yield content, backend, fallback
    except Exception as e:
        yield f"Request failed: {e}", "unknown", "unknown"

with gr.Blocks() as demo:
    gr.Markdown("# AI Gateway Mesh")
    if DEMO_MODE:
        gr.Markdown("Demo mode: set `AI_GATEWAY_BASE_URL` to connect a live gateway.")

    prompt = gr.Textbox(label="Prompt", lines=6, value="Explain service mesh for AI inference traffic.")
    model = gr.Dropdown(label="Model", choices=["auto", "general-small", "general-large", "coding-model"], value="auto")
    stream = gr.Checkbox(label="Stream", value=False)

    btn = gr.Button("Send")
    output = gr.Textbox(label="Response", lines=12)
    backend = gr.Textbox(label="Selected Backend")
    fallback = gr.Textbox(label="Fallback Used")

    btn.click(chat, inputs=[prompt, model, stream], outputs=[output, backend, fallback])

demo.launch()
