# 07: Interactive Host CLI (`heimdall` / `hml`) & Lipgloss TUI

**What to build:** The user-facing terminal binary `heimdall` and official 3-letter alias `hml` (`cmd/agent/main.go`). It accepts natural language symptom descriptions, connects to the Gateway, renders streaming thoughts with step indicators in real-time, displays styled warning boxes for intercepted `SAFE_WRITE` / `DANGEROUS` tools with `[y/N]` prompts, and outputs final diagnosis in formatted markdown.

**Blocked by:** 05: Gateway Turn Orchestrator & Session Loop

**Assigned to:** Agent A (Systems & Go Track)

**Status:** ready-for-agent

- [ ] `cmd/agent/main.go` parses arguments and `--ai-worker` flag using Cobra.
- [ ] Streams turn thoughts with styled Lipgloss badges (`[Turn 1]`, `[Turn 2]`).
- [ ] Prompts operator for `[y/N]` confirmation when risky tools are intercepted.
- [ ] Renders final diagnosis in a styled bordered Lipgloss box.
- [ ] Compiles to `bin/heimdall` with `bin/hml` symlink via `make build`.
