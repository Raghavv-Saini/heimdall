# 05: Gateway Turn Orchestrator & Session Loop

**What to build:** The central Go Gateway coordinator (`services/go-gateway/orchestrator.go`) that executes the ReAct turn loop. Given a user symptom, it generates a unique session UUID, initializes history, and iterates: sending history to Python worker via gRPC `DecideNextStep`, evaluating requested tool permissions, intercepting unapproved actions for operator confirmation, executing authorized tools via the Registry, and appending tool observations to history until a final diagnosis is produced.

**Blocked by:** 02: Python AI Worker ReAct Reasoning Engine & Tool Calling, 04: DevOps Diagnostic Tool Registry (Docker & Systemd)

**Assigned to:** Agent A (Systems & Go Track)

**Status:** ready-for-agent

- [ ] `services/go-gateway/orchestrator.go` implements `Orchestrator.RunInvestigation`.
- [ ] Connects to Python AI Worker over gRPC with timeout contexts.
- [ ] Interleaves thoughts, tool calls, and observations across multi-turn sessions.
- [ ] Halts and triggers `onApprovalRequest` callback when tools require confirmation.
- [ ] Enforces maximum turn limit (default 10) to prevent infinite loops.
- [ ] Returns structured `InvestigationResult` with final diagnosis and turn count.
