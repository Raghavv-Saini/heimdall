# Repository Agent Guidelines & Persistent Memory (`AGENTS.md`)

This repository is developed concurrently by multiple agents and human engineers. All agents operating in this workspace must adhere to the following rules, workflows, and invariants.

---

## 📌 Critical Workflow Invariant: Ticket Completion Protocol

> [!IMPORTANT]
> **After every implementation, you MUST mark the ticket as complete.**
>
> 1. **Check the Frontier**: Before starting work, consult [**`TICKETS.md`**](TICKETS.md). Only pick up a ticket whose blockers are all checked off (`[x]`).
> 2. **Context Scope**: Read the specific ticket requirements in [`.scratch/heimdall/issues/<NN>-*.md`](.scratch/heimdall/issues/). Keep your context focused on that single ticket.
> 3. **Verify Implementation**: Verify that all acceptance criteria are met using automated tests (`make test`, `make eval`, `go test ./...`).
> 4. **Update Ticket State (MANDATORY)**:
>    - In [`.scratch/heimdall/issues/<NN>-*.md`](.scratch/heimdall/issues/): Mark all acceptance criteria checkboxes as checked `[x]`, and update `Status: complete`.
>    - In the master tracker [**`TICKETS.md`**](TICKETS.md): Toggle the ticket's checkbox from `[ ]` to `[x]`, and update its status column to `complete`.
> 5. **Do not end the turn or close the session** without marking the ticket as complete in [**`TICKETS.md`**](TICKETS.md).

---

## 🏗️ Architectural Invariants

1. **Deterministic Safety Gating (Code-Enforced)**:
   - The Python AI worker NEVER executes system commands directly. It only emits structured JSON tool requests.
   - All tool executions are gated by the compiled Go permission engine (`services/go-gateway/permissions/`).
   - Risk Tiers:
     - `READ_ONLY`: Auto-approved inspection (e.g. `docker_inspect`, `systemd_status`, `net_http_probe`).
     - `SAFE_WRITE`: Requires interactive operator `[y/N]` confirmation (e.g. `docker_restart`).
     - `DANGEROUS`: Requires explicit confirmation and impact notice (e.g. `docker_stop`, `docker_remove`).
     - Unrecognized tools default to `DANGEROUS` (zero-trust).

2. **Grounded Observation (Zero Hallucination)**:
   - Diagnostic findings must cite real tool output from the audit trail (`audit_logs` / gRPC messages).
   - Never invent exit codes, container names, or log lines.

3. **Command Injection Immunization**:
   - Never concatenate user or LLM strings into shell commands (`bash -c`, `sh -c`).
   - All tools execute via explicit argument slices: `exec.Command("docker", "inspect", containerID)`.
   - Arguments must be validated against dangerous shell metacharacters (`;`, `&&`, `||`, `` ` ``, `$()`, `rm -rf`).

---

## 🛠️ Verification & Build Commands

- `make build`: Builds the host CLI binary `bin/heimdall` and creates the `bin/hml` symlink.
- `make proto`: Compiles Protocol Buffers for both Go and Python.
- `make test`: Runs all Go unit tests and validates the Python worker.
- `make eval`: Runs the automated evaluation benchmark suite (`tests/evals/runner.py`).
- `make install`: Installs `heimdall` and `hml` into `$HOME/.local/bin`.
