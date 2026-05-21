---
title: AI Gateway Mesh Demo
emoji: 🚦
colorFrom: blue
colorTo: green
sdk: gradio
sdk_version: 5.0.0
app_file: app.py
pinned: false
---

# AI Gateway Mesh Demo

The Space can run in two modes.

Without `AI_GATEWAY_BASE_URL`, it runs in self-contained demo mode and simulates
the gateway routing decision.

To connect it to a live gateway, set these environment variables in Space
settings:

```text
AI_GATEWAY_BASE_URL=https://your-gateway-url
AI_GATEWAY_API_KEY=demo-key
```

For the Render demo blueprint in this repository, use the public URL of the
`ai-gateway-mesh` Render web service as `AI_GATEWAY_BASE_URL`.
