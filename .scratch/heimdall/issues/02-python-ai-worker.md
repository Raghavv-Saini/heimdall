# 02: Python AI Worker ReAct Reasoning Engine & Tool Calling

**What to build:** A runnable Python gRPC service on port `50052` that implements `AIWorkerServiceServicer`. When the Go Gateway sends a `DecideRequest`, the service translates conversation history and tool definitions into unified LLM payloads via `litellm` (supporting user-supplied API keys for OpenAI, Anthropic, Gemini, Groq, or local Ollama/vLLM models via `OPENAI_API_BASE`), returns the model's internal thought and requested tool calls, and signals `is_final_answer: true` when the investigation is complete. It also includes an offline deterministic mock reasoning mode so development and testing proceed without incurring API costs.

**Blocked by:** 01: Shared Protocol Buffer & gRPC IPC Contract

**Assigned to:** Agent B (AI & Reasoning Track)

**Status:** ready-for-agent

- [ ] `services/python-worker/requirements.txt` specifies `grpcio`, `litellm`, and `pydantic`.
- [ ] `services/python-worker/main.py` starts gRPC server on port `50052`.
- [ ] Converts incoming `Message` history to provider format (`user`, `assistant`, `tool`).
- [ ] When API keys (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, etc.) or `OPENAI_API_BASE` are present, calls the model specified in `LLM_MODEL` via `litellm`.
- [ ] When API keys are absent or `HEIMDALL_MOCK_REASONING=true`, runs deterministic mock reasoning for crash investigations.
- [ ] Turn-by-turn tests pass for both tool call generation and final answer delivery.
