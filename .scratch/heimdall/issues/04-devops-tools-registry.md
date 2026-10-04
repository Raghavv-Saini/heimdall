# 04: DevOps Diagnostic Tool Registry (Docker & Systemd)

**What to build:** A suite of uniform diagnostic tools in Go implementing the `Tool` interface (`Name()`, `Description()`, `JSONSchema()`, `RiskLevel()`, `Execute()`). Covers container inspection (`docker_inspect`), container logs (`docker_logs`), host Linux service health (`systemd_status`), and HTTP port probes (`net_http_probe`). Commands execute using structured argument slices (`exec.Command`) to avoid shell interpretation, with observation output returned as formatted JSON strings.

**Blocked by:** 03: Deterministic Safety Gating & Risk Matrix in Go

**Assigned to:** Agent A (Systems & Go Track)

**Status:** ready-for-agent

- [ ] `services/go-gateway/tools/tool.go` defines uniform `Tool` interface.
- [ ] `docker_inspect.go` retrieves container state, exit codes, and OOMKilled flag.
- [ ] `docker_logs.go` retrieves recent stdout/stderr lines with configurable tail limits.
- [ ] `systemd_status.go` queries systemctl service units with timeout protection.
- [ ] `net_http_probe.go` checks HTTP status code, latency, and response headers.
- [ ] `registry.go` packages tools into Protobuf `ToolDefinition` list for the AI worker.
