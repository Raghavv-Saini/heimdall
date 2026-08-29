# Heimdall — Architecture & System Design Specification

## 1. System Overview

Heimdall is built as a **polyglot microservices system** deployed locally via Docker Compose, controlled via a lightweight native Go CLI binary (`heimdall` / short alias: `hml`) on the host machine.

```
┌────────────────────────────────────────────────────────────────────────┐
│                          HOST TERMINAL ENVIRONMENT                     │
│                                                                        │
│   ┌────────────────────────────────────────────────────────────────┐   │
│   │                    Heimdall Go Host CLI                        │   │
│   │    (cmd/agent - Binary: heimdall / Alias: hml)                 │   │
│   └───────────────────────────────┬────────────────────────────────┘   │
└───────────────────────────────────┼────────────────────────────────────┘
                                    │ gRPC (Port 50051)
┌───────────────────────────────────┼────────────────────────────────────┘
│                    DOCKER COMPOSE LOCAL ENVIRONMENT                │
│                                   │                                    │
│   ┌───────────────────────────────▼────────────────────────────────┐   │
│   │               Go API Gateway & Core Engine                     │   │
│   │    - Permission & Risk Gating Engine                           │   │
│   │    - DevOps Tool Registry (Docker, Systemd, Net, Git)          │   │
│   │    - Session ReAct Loop Controller                             │   │
│   └───────────────┬───────────────────┬───────────────────┬────────┘   │
│                   │                   │                   │            │
│   gRPC (Port 50052)│                   │ pgx / sqlc        │ Pub/Sub    │
│   ┌───────────────▼────────┐  ┌───────▼─────────────┐  ┌──▼─────────┐ │
│   │ Python AI Worker Pool  │  │ PostgreSQL 16 DB    │  │ Redis 7    │ │
│   │ - Agent Prompt Builder │  │ - Sessions Table    │  │ - Event Bus│ │
│   │ - Claude 3.5 Sonnet API│  │ - Audit Logs Table  │  │ - Task Queue││
│   └────────────────────────┘  └─────────────────────┘  └────────────┘ │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Component Design & Responsibilities

### 2.1 Host CLI (`cmd/agent/`)
- Written in **Go**.
- Executable binary: `heimdall`, Short shell alias: `hml`.
- Parses command line arguments using `spf13/cobra`.
- Formats streaming output with `charmbracelet/lipgloss`.
- Connects to the Go API Gateway via gRPC (`localhost:50051`).
- Handles interactive human-in-the-loop approval prompts (`[y/N]`) when requested by the Gateway for `SAFE_WRITE` or `DANGEROUS` actions.

### 2.2 Go API Gateway & DevOps Core Engine (`services/go-gateway/`)
- Written in **Go**.
- Exposes gRPC server on `:50051` for the CLI.
- Communicates with Python AI Worker via gRPC (`python-worker:50052`).
- Hosts the **Permission & Risk Evaluation Engine**:
  - Tags every tool call with a risk level (`READ_ONLY`, `SAFE_WRITE`, `DANGEROUS`).
  - Intercepts requests and enforces human confirmation gates deterministically in Go.
- Executes diagnostic tools directly:
  - Docker daemon calls via Official Docker Go SDK (`/var/run/docker.sock`).
  - Linux process & systemd service status via `systemctl`/`journalctl` subprocess execution.
  - Network reachability checks via HTTP/TCP socket probes.
- Writes structured, timestamped audit records to PostgreSQL using `sqlc` generated code and `pgx/v5`.

### 2.3 Python AI Worker (`services/python-worker/`)
- Written in **Python 3.11+**.
- Exposes gRPC server on `:50052`.
- Formats messages and JSON schemas for available tools.
- Invokes Anthropic Claude 3.5 Sonnet API via the `anthropic` SDK.
- Parses Claude's reasoning thoughts and structured `tool_use` requests, returning them via gRPC to the Go Gateway.

### 2.4 Persistence Layer (PostgreSQL 16 & Redis 7)
- **PostgreSQL**: Stores session state and audit logs. Schema managed via `golang-migrate` versioned SQL files (`000001_init_schema.up.sql`). Type-safe query code compiled into Go using `sqlc`.
- **Redis**: Used as an event bus for real-time audit streaming and asynchronous task queuing.

---

## 3. Communication Protocols & Contracts

### 3.1 gRPC Service Interface (`proto/agent_service.proto`)
```protobuf
syntax = "proto3";

package devops.agent.v1;

option go_package = "github.com/raghavdev/heimdall/proto/v1;agentv1";

service AIWorkerService {
  rpc DecideNextStep (DecideRequest) returns (DecideResponse);
}

message Message {
  string role = 1;
  string content = 2;
  string tool_call_id = 3;
  string tool_name = 4;
}

message ToolDefinition {
  string name = 1;
  string description = 2;
  string json_schema = 3;
  string risk_level = 4;
}

message DecideRequest {
  string session_id = 1;
  string user_prompt = 2;
  repeated Message conversation_history = 3;
  repeated ToolDefinition available_tools = 4;
}

message ToolCall {
  string id = 1;
  string name = 2;
  string arguments_json = 3;
}

message DecideResponse {
  string thought = 1;
  bool is_final_answer = 2;
  string final_answer = 3;
  repeated ToolCall requested_tool_calls = 4;
}
```

---

## 4. Security & Threat Modeling

1. **Strict Code-Level Risk Gating**: The LLM cannot grant itself elevated permissions. Permission evaluation happens inside compiled Go code prior to tool dispatch.
2. **Command Injection Immunization**: Tools do not construct free-form shell strings. Arguments are validated against Pydantic / JSON schemas and executed using explicit command argument arrays (`exec.Command("docker", "inspect", containerID)`).
3. **Secrets Protection**: Credentials and environment variables containing `KEY`, `PASS`, `TOKEN`, or `SECRET` are automatically masked (`***REDACTED***`) before saving to database or sending to external APIs.
