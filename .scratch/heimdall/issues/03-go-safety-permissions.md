# 03: Deterministic Safety Gating & Risk Matrix in Go

**What to build:** A compiled Go permission engine (`services/go-gateway/permissions/`) that evaluates risk prior to executing any system tool requested by the AI. It statically classifies operations into `READ_ONLY` (auto-approved), `SAFE_WRITE` (requires operator confirmation), and `DANGEROUS` (requires explicit confirmation). It validates JSON arguments and strictly rejects shell injection sequences (`;`, `&&`, `||`, `` ` ``, `$()`, `rm -rf`), adopting a zero-trust policy that defaults unknown tools to `DANGEROUS`.

**Blocked by:** 01: Shared Protocol Buffer & gRPC IPC Contract

**Assigned to:** Agent A (Systems & Go Track)

**Status:** ready-for-agent

- [ ] `services/go-gateway/permissions/engine.go` implements static risk mapping and argument sanitization.
- [ ] Unknown tool requests default to `DANGEROUS`.
- [ ] Attempted shell injections in arguments return a validation error.
- [ ] Unit tests in `engine_test.go` cover `READ_ONLY`, `SAFE_WRITE`, `DANGEROUS`, zero-trust, and injection rejection with 100% pass rate.
