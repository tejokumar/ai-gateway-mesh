import json
import os
import time
from http.server import BaseHTTPRequestHandler, HTTPServer

BACKEND_NAME = os.getenv("BACKEND_NAME", "mock-backend")
MOCK_STATUS = int(os.getenv("MOCK_STATUS", "200"))
MOCK_DELAY_MS = int(os.getenv("MOCK_DELAY_MS", "0"))
PORT = int(os.getenv("PORT", "8000"))

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_response(404)
            self.end_headers()
            return
        length = int(self.headers.get("content-length", "0"))
        body = self.rfile.read(length)
        try:
            payload = json.loads(body or b"{}")
        except Exception:
            payload = {}

        if MOCK_DELAY_MS > 0:
            time.sleep(MOCK_DELAY_MS / 1000)

        if MOCK_STATUS >= 400:
            out = json.dumps({
                "error": {
                    "message": f"{BACKEND_NAME} forced status {MOCK_STATUS}",
                    "type": "mock_error",
                }
            }).encode()
            self.send_response(MOCK_STATUS)
            self.send_header("content-type", "application/json")
            self.send_header("content-length", str(len(out)))
            self.end_headers()
            self.wfile.write(out)
            return

        if payload.get("stream"):
            self.send_response(200)
            self.send_header("content-type", "text/event-stream")
            self.send_header("cache-control", "no-cache")
            self.end_headers()
            chunks = [
                {
                    "id": "chatcmpl-mock",
                    "object": "chat.completion.chunk",
                    "created": 0,
                    "model": payload.get("model", BACKEND_NAME),
                    "choices": [
                        {
                            "index": 0,
                            "delta": {"role": "assistant"},
                            "finish_reason": None
                        }
                    ],
                },
                {
                    "id": "chatcmpl-mock",
                    "object": "chat.completion.chunk",
                    "created": 0,
                    "model": payload.get("model", BACKEND_NAME),
                    "choices": [
                        {
                            "index": 0,
                            "delta": {"content": f"Response from {BACKEND_NAME}."},
                            "finish_reason": None
                        }
                    ],
                },
                {
                    "id": "chatcmpl-mock",
                    "object": "chat.completion.chunk",
                    "created": 0,
                    "model": payload.get("model", BACKEND_NAME),
                    "choices": [
                        {
                            "index": 0,
                            "delta": {},
                            "finish_reason": "stop"
                        }
                    ],
                },
            ]
            for chunk in chunks:
                self.wfile.write(f"data: {json.dumps(chunk)}\n\n".encode())
                self.wfile.flush()
            self.wfile.write(b"data: [DONE]\n\n")
            self.wfile.flush()
            return

        content = f"Response from {BACKEND_NAME}. Model={payload.get('model', 'unknown')}"
        resp = {
            "id": "chatcmpl-mock",
            "object": "chat.completion",
            "created": 0,
            "model": payload.get("model", BACKEND_NAME),
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": content},
                    "finish_reason": "stop"
                }
            ],
            "usage": {"prompt_tokens": 10, "completion_tokens": 10, "total_tokens": 20}
        }
        out = json.dumps(resp).encode()
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(out)))
        self.end_headers()
        self.wfile.write(out)

    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
            return
        self.send_response(404)
        self.end_headers()

HTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
