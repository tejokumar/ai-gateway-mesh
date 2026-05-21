import os
import json
import requests
import gradio as gr

BASE_URL = os.getenv("AI_GATEWAY_BASE_URL", "http://localhost:8080")
API_KEY = os.getenv("AI_GATEWAY_API_KEY", "demo-key")

def chat(prompt, model, stream):
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
    gr.Markdown("Production-style LLM inference gateway with routing, fallback, and observability.")

    prompt = gr.Textbox(label="Prompt", lines=6, value="Explain service mesh for AI inference traffic.")
    model = gr.Dropdown(label="Model", choices=["auto", "general-small", "general-large", "coding-model"], value="auto")
    stream = gr.Checkbox(label="Stream", value=False)

    btn = gr.Button("Send")
    output = gr.Textbox(label="Response", lines=12)
    backend = gr.Textbox(label="Selected Backend")
    fallback = gr.Textbox(label="Fallback Used")

    btn.click(chat, inputs=[prompt, model, stream], outputs=[output, backend, fallback])

demo.launch()
