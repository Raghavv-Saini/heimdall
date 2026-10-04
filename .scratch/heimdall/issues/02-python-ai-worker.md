# 02: Python AI Worker ReAct Reasoning Engine & Tool Calling

**What to build:** A runnable Python gRPC service on port `50052` that implements `AIWorkerServiceServicer`. When the Go Gateway sends a `DecideRequest`, the service translates conversation history and tool definitions into Anthropic Claude 3.5 Sonnet Messages API payloads, returns Claude's internal thought and requested `tool_use` blocks, and signals `is_final_answer: true` when the investigation is complete. It also includes an offline deterministic mock reasoning mode so development and testing proceed without incurring API costs.

**Blocked by:** 01: Shared Protocol Buffer & gRPC IPC Contract

**Assigned to:** Agent B (AI & Reasoning Track)

**Status:** ready-for-agent

- [ ] `services/python-worker/requirements.txt` specifies `grpcio`, `anthropic`, and `pydantic`.
- [ ] `services/python-worker/main.py` starts gRPC server on port `50052`.
- [ ] Converts incoming `Message` history to Anthropic format (`user`, `assistant`, `tool_result`).
- [ ] When `ANTHROPIC_API_KEY` is present, calls Claude 3.5 Sonnet with tools schema and system prompt.
- [ ] When `ANTHROPIC_API_KEY` is absent, runs deterministic mock reasoning for crash investigations.
- [ ] Turn-by-turn tests pass for both tool call generation and final answer delivery.
