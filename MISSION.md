# Mission: Heimdall Agentic DevOps Architecture & Systems Engineering

## Why
Master the first-principles architecture, deterministic security gating, and polyglot microservice design of Heimdall to confidently contribute to, audit, and extend its Go and Python subsystems.

## Success looks like
- Explain the end-to-end ReAct loop and why the Go Gateway / Python Worker split was chosen over a monolith.
- Articulate the role of deterministic code-enforced safety gating vs. probabilistic LLM generation.
- Trace how Ticket 01's gRPC IPC contract connects the stateful Gateway with the stateless Python reasoner.
- Confidently implement and review downstream tickets (02, 03, etc.) adhering to all repository invariants.

## Constraints
- Systems-first, grounded in the actual codebase files and contracts.
- High signal, zero hallucination, zero fluff.

## Out of scope
- Theoretical transformer model pre-training mechanics.
- Generic CI/CD automation pipelines unrelated to Heimdall's runtime triage engine.
