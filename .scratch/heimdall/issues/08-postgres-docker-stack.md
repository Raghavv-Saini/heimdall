# 08: PostgreSQL 16 Audit Persistence & Docker Stack Integration

**What to build:** The immutable audit persistence layer and containerized deployment stack. Provides PostgreSQL 16 schema migrations (`database/migrations/000001_init_schema.up.sql`) creating `sessions` and `audit_logs` tables to record every intermediate thought, tool invocation, risk tier, and human confirmation. Configures `docker-compose.yml` to orchestrate PostgreSQL, Redis, and the Python AI Worker, and standardizes developer workflows in `Makefile`.

**Blocked by:** 05: Gateway Turn Orchestrator & Session Loop, 06: Automated Evaluation Benchmark Suite (OOM & Safety Gating)

**Assigned to:** Agent B (or jointly)

**Status:** ready-for-agent

- [ ] `database/migrations/000001_init_schema.up.sql` establishes `sessions` and `audit_logs` tables.
- [ ] `docker-compose.yml` launches `postgres:16-alpine`, `redis:7-alpine`, and `python-worker`.
- [ ] `services/python-worker/Dockerfile` builds container image with generated proto stubs.
- [ ] `Makefile` provides targets: `build`, `proto`, `test`, `eval`, `install`.
- [ ] Whole stack can be brought up with `docker compose up -d`.
