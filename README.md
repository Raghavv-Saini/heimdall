# Heimdall (`hml`) — Enterprise Agentic DevOps CLI & Triage Engine

[![Architecture: Polyglot Microservices](https://img.shields.io/badge/Architecture-Distributed%20Microservices-blue.svg)](docs/PRD.md)
[![Language: Go & Python](https://img.shields.io/badge/Languages-Go%20%7C%20Python-00ADD8.svg)](docs/PRD.md)
[![Database: PostgreSQL 16](https://img.shields.io/badge/Database-PostgreSQL%2016-336791.svg)](database/migrations/)
[![IPC: gRPC & Redis](https://img.shields.io/badge/IPC-gRPC%20%7C%20Redis-2496ED.svg)](proto/agent_service.proto)

**Heimdall** (terminal binary: `heimdall` / official short alias: `hml`) is an autonomous, agentic DevOps CLI and triage engine built for software engineers, SREs, and DevOps teams. Named after the all-seeing guardian who guards Asgard's gates and signals incidents, **Heimdall automates repetitive investigative toil** during local infrastructure failures, container crash-loops, network misconfigurations, and systemd service outages.

Rather than running rigid scripts or forcing humans to manually execute complex decision trees across `docker`, `journalctl`, `curl`, and `kubectl`, **Heimdall runs an adaptive reasoning → tool execution → observation → hypothesis validation loop** to isolate root causes and propose verified remediations.

---

## 🌟 Terminal Ergonomics & `hml` Alias

Heimdall compiles to a fast Go binary `heimdall` and sets up **`hml`** (**H**ei**m**da**l**l) as its official 3-letter terminal alias:

```bash
# Full command:
heimdall "why is my web container crash-looping?"

# Official 3-letter terminal alias:
hml "why is my web container crash-looping?"
```

---

## 🌟 Architectural Moat & Engineering Guarantees

1. **Deterministic Safety Gating (Code-Level, Non-Bypassable)**: The AI reasoning worker *cannot* execute system commands directly. Risk classification (`READ_ONLY`, `SAFE_WRITE`, `DANGEROUS`) is evaluated by a compiled Go permission engine. Unapproved destructive actions are physically blocked before reaching the host shell or Docker daemon.
2. **Polyglot High-Performance Architecture**: 
   - **Go**: Powers the fast host CLI client (`hml`/`heimdall`), API Gateway, permission engine, Docker Go SDK, Linux systemd DBus/subprocess execution, and `sqlc` database layer.
   - **Python**: Handles AI/LLM integrations, prompt engineering, structured Pydantic schemas, and provider-agnostic agent reasoning loops via `litellm` (supporting OpenAI, Anthropic Claude, Google Gemini, Groq, or local Ollama/vLLM models).
3. **Grounded Observation (Zero Hallucination)**: System state is strictly derived from actual tool outputs (`docker inspect`, `journalctl`, HTTP probes), logged and verified turn-by-turn.
4. **Immutable Audit Trail**: Every prompt, intermediate thought, proposed tool call, risk score, human confirmation, tool stdout/stderr, and final diagnosis is transactionally committed to **PostgreSQL 16**.

---

## 📐 System Architecture

```mermaid
graph TD
    subgraph Host Terminal Environment
        CLI[Heimdall Host CLI Binary<br/>- Command: heimdall / hml<br/>- Cobra CLI Parser<br/>- Lipgloss TUI<br/>- gRPC Client]
    end

    subgraph Docker Compose Local Microservices Stack
        GW[Go API Gateway & DevOps Core<br/>- Permission & Risk Gating Engine<br/>- Tool Execution Registry<br/>- Systemd & Docker Integration<br/>- Session State Machine]
        DB[(PostgreSQL 16<br/>- Session Store<br/>- Immutable Audit Log)]
        REDIS[(Redis 7<br/>- Audit Event Bus<br/>- Async Task Queue)]
        PY[Python AI Worker Pool<br/>- ReAct Agent Loop<br/>- System Prompt Builder<br/>- LiteLLM Multi-Provider BYOK Router]
    end

    subgraph Host Operating System & Diagnostic Surfaces
        DOCKER[Local Docker Daemon<br/>/var/run/docker.sock]
        SYSTEMD[Linux Systemd / Journald<br/>DBus & Subprocess]
        NET[Network Socket Probes<br/>TCP / HTTP / DNS]
        GIT[Git Workspace Repository]
        LLM[LLM API / Local Endpoint<br/>OpenAI / Anthropic / Gemini / Ollama]
    end

    CLI <-->|gRPC Port 50051<br/>Stream Thoughts & Approvals| GW
    GW <-->|gRPC Port 50052<br/>DecideNextStep RPC| PY
    GW -->|pgx / sqlc| DB
    GW <-->|Redis Pub/Sub| REDIS
    PY <-->|Async Event Logs| REDIS
    PY <-->|HTTPS / HTTP API| LLM
    GW -->|Docker Go SDK| DOCKER
    GW -->|systemctl / journalctl| SYSTEMD
    GW -->|HTTP Probes| NET
    GW -->|Git CLI| GIT
```

---

## 🛠️ Tech Stack & Microservices Breakdown

- **Host CLI**: Go 1.22+ (`spf13/cobra`, `charmbracelet/lipgloss`) — Binary: `heimdall`, Alias: `hml`
- **API Gateway & Core Orchestrator**: Go 1.22+ (`gRPC`, `net/http`, `jackc/pgx/v5`, `docker/docker/client`)
- **AI Worker Pool**: Python 3.11+ (`grpcio`, `anthropic`, `pydantic`, `litellm`)
- **Inter-Service Communication**: gRPC over HTTP/2 with Protocol Buffers v3 (`proto/agent_service.proto`)
- **Database & Data Access**: PostgreSQL 16 with `golang-migrate` versioning and `sqlc` type-safe query generation
- **Message Broker & Queue**: Redis 7 Alpine
- **Containerization**: Docker & Docker Compose v2

---

## 🔒 3-Tier Permission & Safety Model

| Tier | Behavior | Example Diagnostic Tools |
|---|---|---|
| **`READ_ONLY`** | Auto-approved, logged to DB | `docker_list_containers`, `docker_inspect`, `docker_logs`, `systemd_status`, `net_http_probe` |
| **`SAFE_WRITE`** | Requires interactive confirmation `[y/N]` | `docker_restart`, scaling dev deployments, clearing cache |
| **`DANGEROUS`** | Requires explicit confirmation + impact summary | `docker_remove`, stopping host systemd services, dropping volumes |

---

## 🚀 Quickstart & Local Setup

### 1. Prerequisites
- Docker & Docker Compose v2 installed
- Go 1.22+ installed
- Python 3.11+ installed
- LLM API Key or Local Endpoint (e.g. `export OPENAI_API_KEY="your-key"` or `export ANTHROPIC_API_KEY="your-key"`, or `export OPENAI_API_BASE="http://localhost:11434/v1"` for local Ollama)

### 2. Start the Local Microservices Stack
```bash
# Clone repository
git clone https://github.com/raghavdev/heimdall.git
cd heimdall

# Start PostgreSQL, Redis, Python AI Worker, and Go Gateway
docker compose up -d

# Check stack health
docker compose ps
```

### 3. Run an Investigation via `hml`
```bash
# Build and install heimdall & hml alias
make install

# Run investigation using official 3-letter alias hml
hml "why is my container crashing?"
```

---

## 📚 Documentation Sitemap

- 📄 **[PRD & Technical Specification](docs/PRD.md)** — Complete Product Requirement Document for Heimdall written from an AI Startup Founder perspective.
- 📄 **[Architecture & Threat Model](docs/architecture.md)** — In-depth microservices architecture design, threat modeling, and gRPC interface specifications.
- 📄 **[Mental Model & Roadmap](docs/agentic-devops-cli.md)** — Core mentor-style guide explaining the paradigm shift from traditional CLI tools to agentic loops.

---

## 📄 License
Apache 2.0 License. See [LICENSE](LICENSE) for details.
