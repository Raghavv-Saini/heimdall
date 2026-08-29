# HEIMDALL — MASTER PRODUCT REQUIREMENT DOCUMENT (PRD) & SYSTEM DESIGN SPECIFICATION

**Document Title:** Heimdall Agentic DevOps CLI & Triage Platform — Complete Technical Specification  
**Version:** 2.0.0 (Master Enterprise Blueprint)  
**Perspective:** AI Startup Founder & Lead Systems Architect  
**Status:** Baseline Specification for Immediate Implementation  

---

## TABLE OF CONTENTS
1. [Executive Summary & Commercial Vision](#1-executive-summary--commercial-vision)
2. [Core Product Principles & Architectural Moat](#2-core-product-principles--architectural-moat)
3. [Complete Technical Stack & Dependency Inventory](#3-complete-technical-stack--dependency-inventory)
4. [Distributed Microservices Network Topology](#4-distributed-microservices-network-topology)
5. [End-to-End System Diagrams & Sequence Flows](#5-end-to-end-system-diagrams--sequence-flows)
6. [gRPC & Protocol Buffer API Specification](#6-grpc--protocol-buffer-api-specification)
7. [Database Architecture, DDL & Data Access Layer](#7-database-architecture-ddl--data-access-layer)
8. [Deterministic Security Engine & Threat Model](#8-deterministic-security-engine--threat-model)
9. [DevOps Diagnostic Tool Registry Specifications](#9-devops-diagnostic-tool-registry-specifications)
10. [Systemd & Host Linux Operating System Integration](#10-systemd--host-linux-operating-system-integration)
11. [Python AI Worker Pool & ReAct Reasoning Engine](#11-python-ai-worker-pool--react-reasoning-engine)
12. [Go Host CLI & Terminal User Experience (TUI)](#12-go-host-cli--terminal-user-experience-tui)
13. [Automated Evaluation Harness & Benchmark Scenarios](#13-automated-evaluation-harness--benchmark-scenarios)
14. [Local Developer Quickstart & Setup Playbook](#14-local-developer-quickstart--setup-playbook)
15. [Product Roadmap (v1.0 MVP to v2.0 Enterprise)](#15-product-roadmap-v10-mvp-to-v20-enterprise)

---

## 1. Executive Summary & Commercial Vision

### 1.1 The Operational Problem
When modern microservice or containerized infrastructure experiences an incident (e.g., container crash-loops, port connection failures, OOM kills, systemd service outages), engineering teams waste 60–80% of their Mean-Time-To-Diagnose (MTTD) running a **repeatable, manual decision tree**:
1. SSH into host or query cluster status (`docker ps`, `kubectl get pods`).
2. Fetch logs and scroll for stack traces (`docker logs`, `journalctl`).
3. Inspect container metadata, exit codes, and environment variables (`docker inspect`).
4. Probe network connectivity (`curl`, `nc`).
5. Correlate failures with recent git commits or configuration changes.

This manual diagnostic workflow is tedious, context-switching-heavy, and error-prone. 

### 1.2 The Heimdall Solution
**Heimdall** (binary: `heimdall` / terminal alias: `hml`) is an enterprise-grade agentic DevOps CLI and triage engine. The user describes a symptom or goal in natural language (*"why is my web service crashing?"* or *"check if my local API dependencies are healthy"*), and Heimdall runs an autonomous **reasoning → tool execution → observation → hypothesis validation loop** to isolate the root cause and propose verified remediations.

### 1.3 Target Audience & Value Proposition
- **Target Users**: DevOps Engineers, Site Reliability Engineers (SREs), Backend Engineers, and Systems Administrators.
- **Value Metric**: Reduces MTTD by up to **75%** by automating investigative toil while keeping a human operator strictly in control of risky write/destructive operations.

---

## 2. Core Product Principles & Architectural Moat

Heimdall is explicitly engineered to avoid the failure modes of generic "LLM CLI wrappers":

| Core Principle | Problem in Generic AI Tools | Heimdall Architectural Solution |
|---|---|---|
| **Deterministic Safety Gating** | LLMs hallucinate destructive commands (`rm -rf /`, dropping DBs). | Risk evaluation (`READ_ONLY`, `SAFE_WRITE`, `DANGEROUS`) is calculated in compiled Go code. Unapproved dangerous actions are blocked before execution. |
| **Grounded Observation** | LLMs invent non-existent log lines or fake root causes. | Reasoning is strictly grounded in actual tool output (`docker inspect`, `journalctl`, probes) fed back into the context. |
| **Immutable Auditability** | No record of what commands an AI assistant executed. | Transactional, timestamped audit log of all prompts, thoughts, proposed tools, risk scores, approvals, and outputs in PostgreSQL 16. |
| **Polyglot Efficiency** | Pure Python CLIs are slow; pure Go AI tools lack deep LLM SDKs. | Hybrid design: Go for fast CLI, Gateway, Docker/Systemd SDKs, and security; Python for AI reasoning worker pool. |

---

## 3. Complete Technical Stack & Dependency Inventory

### 3.1 Go Core Services & Host CLI (`Go 1.22+`)
- **CLI Framework**: `github.com/spf13/cobra` (Command parser), `github.com/charmbracelet/lipgloss` (Terminal styling), `github.com/charmbracelet/bubbletea` (Interactive streaming TUI).
- **gRPC Server & Client**: `google.golang.org/grpc`, `google.golang.org/protobuf`.
- **Database Driver & Migration**: `github.com/jackc/pgx/v5` (High-performance PostgreSQL driver), `sqlc` (Compile-time type-safe SQL generator), `github.com/golang-migrate/migrate/v4`.
- **System Integrations**: `github.com/docker/docker/client` (Official Docker Engine API Go SDK), `github.com/godbus/dbus/v5` (Linux Systemd DBus client).

### 3.2 Python AI Worker Pool (`Python 3.11+`)
- **gRPC Service**: `grpcio`, `grpcio-tools`.
- **LLM SDKs**: `anthropic` (Claude 3.5 Sonnet API), `litellm` (Fallback provider support).
- **Data Validation & Parsing**: `pydantic` v2, `instructor`.
- **Logging & Utilities**: `structlog`.

### 3.3 Infrastructure & Storage
- **Database**: PostgreSQL 16 (Alpine image) with `uuid-ossp` extension.
- **Message Broker / Cache**: Redis 7 (Alpine image) for pub/sub event streaming and async worker queues.
- **Orchestration**: Docker Compose v2 (`docker-compose.yml`).

---

## 4. Distributed Microservices Network Topology

```
+-----------------------------------------------------------------------------------+
|                            HOST TERMINAL (USER MACHINE)                           |
|                                                                                   |
|  +-----------------------------------------------------------------------------+  |
|  |                   Heimdall Go Host CLI (heimdall / hml)                     |  |
|  |    (cmd/agent - Cobra CLI Parser, Lipgloss TUI, gRPC Client)                |  |
|  +-------------------------------------+---------------------------------------+  |
+----------------------------------------|------------------------------------------+
                                         | gRPC (localhost:50051)
+----------------------------------------|------------------------------------------+
|                 DOCKER COMPOSE LOCAL MICROSERVICES STACK                          |
|                                        |                                          |
|  +-------------------------------------v---------------------------------------+  |
|  |                 Go API Gateway & DevOps Core Engine                         |  |
|  |    (services/go-gateway - Port 50051)                                       |  |
|  |    - Permission & Risk Gating Engine                                        |  |
|  |    - DevOps Tool Registry (Docker SDK, Systemd DBus, Probes)                |  |
|  |    - Session State Machine & Turn Controller                                |  |
|  +----------------+--------------------+--------------------+------------------+  |
|                   |                    |                    |                     |
|  gRPC (Port 50052)|                    | pgx / sqlc         | Redis Pub/Sub       |
|  +----------------v-------+   +--------v----------------+   +--v---------------+  |
|  | Python AI Worker Pool  |   | PostgreSQL 16 Database  |   | Redis 7 Broker   |  |
|  | (services/python-worker|   | (heimdall-postgres:5432)|   | (heimdall-redis: |  |
|  |  - Claude 3.5 Sonnet)  |   | - sessions table        |   |    6379)         |  |
|  |                        |   | - audit_logs table      |   | - audit channel  |  |
|  +------------------------+   +-------------------------+   +------------------+  |
+-----------------------------------------------------------------------------------+
```

---

## 5. End-to-End System Diagrams & Sequence Flows

### 5.1 Interactive Triage Execution Flow

```mermaid
sequenceDiagram
    autonumber
    actor User as Operator (hml / heimdall)
    participant GW as Go Gateway / Core Engine
    participant DB as PostgreSQL Audit DB
    participant PY as Python AI Worker
    participant LLM as Anthropic Claude API
    participant TOOL as Docker / System Tool

    User->>GW: StartSession("why is my container crashing?")
    GW->>DB: INSERT INTO sessions (user_prompt, status='RUNNING')
    
    loop ReAct Loop (Step 0 to MAX_STEPS)
        GW->>PY: DecideNextStep(session_id, history, available_tools)
        PY->>LLM: Post messages + tool JSON schemas
        LLM-->>PY: Return Thought + ToolCall (e.g., docker_inspect)
        PY-->>GW: DecideResponse(thought, requested_tools)
        GW->>DB: INSERT INTO audit_logs (step_type='THOUGHT')
        GW-->>User: Stream reasoning thought to CLI UI

        GW->>GW: PermissionEngine.Evaluate(tool_name, tool_args)
        
        alt Risk Tier == READ_ONLY
            GW->>TOOL: Execute tool (e.g. docker inspect)
            TOOL-->>GW: Return stdout / observation
        else Risk Tier == SAFE_WRITE or DANGEROUS
            GW-->>User: RequestHumanApproval(tool_summary, risk_tier)
            User->>GW: User responds: 'y' (Approved)
            GW->>DB: INSERT INTO audit_logs (step_type='APPROVAL_REQUEST', human_approved=true)
            GW->>TOOL: Execute action (e.g. docker restart)
            TOOL-->>GW: Return stdout / action result
        end

        GW->>DB: INSERT INTO audit_logs (step_type='TOOL_RESULT', observation)
    end

    PY-->>GW: DecideResponse(is_final_answer=true, final_diagnosis)
    GW->>DB: UPDATE sessions SET status='COMPLETED'
    GW-->>User: Render formatted markdown diagnosis in CLI
```

---

## 6. gRPC & Protocol Buffer API Specification

### `proto/agent_service.proto`

```protobuf
syntax = "proto3";

package devops.agent.v1;

option go_package = "github.com/raghavdev/heimdall/proto/v1;agentv1";

// AIWorkerService handles LLM reasoning, tool selection, and diagnosis generation
service AIWorkerService {
  // Evaluates the current conversation trajectory and decides the next action
  rpc DecideNextStep (DecideRequest) returns (DecideResponse);
}

// Conversation message object
message Message {
  string role = 1;         // "user", "assistant", "tool"
  string content = 2;      // Text message or tool execution result
  string tool_call_id = 3; // ID of the tool call if role == "tool"
  string tool_name = 4;    // Name of the tool executed
}

// Definition of a tool exposed to the LLM
message ToolDefinition {
  string name = 1;        // Unique tool identifier (e.g., "docker_inspect")
  string description = 2; // Human & LLM readable description
  string json_schema = 3; // Stringified JSON Schema for input arguments
  string risk_level = 4;  // "READ_ONLY", "SAFE_WRITE", "DANGEROUS"
}

// Request sent from Go Gateway to Python AI Worker
message DecideRequest {
  string session_id = 1;
  string user_prompt = 2;
  repeated Message conversation_history = 3;
  repeated ToolDefinition available_tools = 4;
}

// Tool call request returned by the LLM
message ToolCall {
  string id = 1;
  string name = 2;
  string arguments_json = 3; // Stringified JSON arguments matching tool input_schema
}

// Response returned from Python AI Worker to Go Gateway
message DecideResponse {
  string thought = 1;                        // Reasoning string generated by the LLM
  bool is_final_answer = 2;                 // True if investigation is complete
  string final_answer = 3;                  // Final diagnosis string (Markdown)
  repeated ToolCall requested_tool_calls = 4; // List of tool calls requested by LLM
}
```

---

## 7. Database Architecture, DDL & Data Access Layer

### 7.1 Database Schema Migration (`database/migrations/000001_init_schema.up.sql`)

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Define Enums for Risk Levels and Step Types
CREATE TYPE risk_level AS ENUM ('READ_ONLY', 'SAFE_WRITE', 'DANGEROUS');
CREATE TYPE step_type AS ENUM ('THOUGHT', 'TOOL_CALL', 'TOOL_RESULT', 'APPROVAL_REQUEST', 'FINAL_ANSWER');

-- Sessions Table: Tracks overall investigation session state
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_prompt TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'RUNNING',
    max_steps INT NOT NULL DEFAULT 10,
    current_step INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit Logs Table: Immutable record of every reasoning step and tool execution
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    step_index INT NOT NULL,
    step_type step_type NOT NULL,
    tool_name VARCHAR(64),
    tool_args JSONB,
    risk_level risk_level,
    human_approved BOOLEAN,
    observation TEXT,
    llm_thought TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for fast query retrieval
CREATE INDEX IF NOT EXISTS idx_audit_logs_session ON audit_logs(session_id, step_index);
CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);
```

### 7.2 SQL Queries for `sqlc` (`database/queries/audit.sql`)

```sql
-- name: CreateSession :one
INSERT INTO sessions (user_prompt, max_steps)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateSessionStatus :one
UPDATE sessions 
SET status = $2, current_step = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RecordAuditLog :one
INSERT INTO audit_logs (
    session_id, step_index, step_type, tool_name, tool_args, 
    risk_level, human_approved, observation, llm_thought, error_message
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetSessionAuditTrail :many
SELECT * FROM audit_logs
WHERE session_id = $1
ORDER BY step_index ASC;

-- name: ListRecentSessions :many
SELECT * FROM sessions
ORDER BY created_at DESC
LIMIT $1;
```

---

## 8. Deterministic Security Engine & Threat Model

### 8.1 Safety Architecture
Security in Heimdall is **code-enforced**, not prompt-dependent. The Python AI worker returns tool requests, but the Go Core Gateway evaluates permissions prior to calling any OS/Docker APIs.

```
       +-------------------------------------------------------+
       |   Python AI Worker Requests Tool (e.g. docker_stop)   |
       +---------------------------+---------------------------+
                                   |
                                   v
       +-------------------------------------------------------+
       |       Go Permission Engine (permissions/engine.go)    |
       |       Looks up tool_name in static Risk Matrix        |
       +---------------------------+---------------------------+
                                   |
            +----------------------+----------------------+
            |                                             |
            v                                             v
  [Tier: READ_ONLY]                             [Tier: DANGEROUS]
  - Auto-Approved                               - Requires Confirmation
  - Execute Tool Immediately                    - Send Request to Host CLI
                                                - User types 'y' -> Execute
                                                - User types 'N' -> Abort Step
```

### 8.2 Risk Mapping Matrix

| Tool Name | Risk Tier | Human Confirmation Required? | Operational Impact |
|---|---|---|---|
| `docker_list_containers` | `READ_ONLY` | No | Zero impact. Lists active containers. |
| `docker_inspect` | `READ_ONLY` | No | Zero impact. Reads state/mounts. |
| `docker_logs` | `READ_ONLY` | No | Zero impact. Reads stdout/stderr. |
| `systemd_status` | `READ_ONLY` | No | Zero impact. Queries `systemctl status`. |
| `net_http_probe` | `READ_ONLY` | No | Zero impact. Hits HTTP endpoint. |
| `docker_restart` | `SAFE_WRITE` | Yes `[y/N]` | Temporarily restarts dev container. |
| `docker_stop` | `DANGEROUS` | Yes `[y/N]` + Type Name | Stops container, causing downtime. |
| `docker_remove` | `DANGEROUS` | Yes `[y/N]` + Type Name | Deletes container permanently. |
| `systemd_restart` | `DANGEROUS` | Yes `[y/N]` + Type Name | Restarts host system service. |

---

## 9. DevOps Diagnostic Tool Registry Specifications

### 9.1 `docker_inspect` Tool Implementation (Go)

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/client"
)

type DockerInspectTool struct {
	dockerClient *client.Client
}

func (t *DockerInspectTool) Name() string { return "docker_inspect" }

func (t *DockerInspectTool) Description() string {
	return "Inspect detailed low-level metadata of a container including state, exit code, OOMKilled flag, mounts, and env keys."
}

func (t *DockerInspectTool) JSONSchema() string {
	return `{
		"type": "object",
		"properties": {
			"container_id": {
				"type": "string",
				"description": "The name or ID of the Docker container to inspect."
			}
		},
		"required": ["container_id"]
	}`
}

func (t *DockerInspectTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		ContainerID string `json:"container_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	inspect, err := t.dockerClient.ContainerInspect(ctx, args.ContainerID)
	if err != nil {
		return "", fmt.Errorf("docker inspect failed: %w", err)
	}

	result := map[string]interface{}{
		"id":           inspect.ID[:12],
		"name":         inspect.Name,
		"state":        inspect.State.Status,
		"running":      inspect.State.Running,
		"exit_code":    inspect.State.ExitCode,
		"oom_killed":   inspect.State.OOMKilled,
		"error":        inspect.State.Error,
		"finished_at": inspect.State.FinishedAt,
		"restart_count": inspect.RestartCount,
	}

	bytes, _ := json.MarshalIndent(result, "", "  ")
	return string(bytes), nil
}
```

---

## 10. Systemd & Host Linux Operating System Integration

### 10.1 Host Systemd Integration
Heimdall interacts with the Linux host systemd service manager via direct DBus calls (`github.com/godbus/dbus/v5`) or `systemctl` / `journalctl` invocations.

### 10.2 Daemon Systemd Unit File (`/etc/systemd/system/heimdall-gateway.service`)

```ini
[Unit]
Description=Heimdall Agentic DevOps API Gateway & Orchestrator Daemon
After=network.target docker.service postgresql.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/heimdall
ExecStart=/usr/local/bin/heimdall-gateway --config /etc/heimdall/config.yaml
Restart=on-failure
RestartSec=5s

# Security Hardening Directives
ProtectSystem=full
ProtectHome=read-only
PrivateTmp=true
CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_SYS_PTRACE

Environment=DATABASE_URL=postgres://devops:devops_password@127.0.0.1:5432/heimdall_db?sslmode=disable
Environment=REDIS_URL=127.0.0.1:6379
Environment=PYTHON_WORKER_URL=127.0.0.1:50052

[Install]
WantedBy=multi-user.target
```

---

## 11. Python AI Worker Pool & ReAct Reasoning Engine

### `services/python-worker/main.py` Implementation

```python
import concurrent.futures
import json
import os
import grpc
from anthropic import Anthropic
import agent_pb2
import agent_pb2_grpc

class AIWorkerService(agent_pb2_grpc.AIWorkerServiceServicer):
    def __init__(self):
        api_key = os.getenv("ANTHROPIC_API_KEY")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY environment variable is required")
        self.client = Anthropic(api_key=api_key)

    def DecideNextStep(self, request, context):
        system_prompt = (
            "You are Heimdall, an expert autonomous DevOps Investigation Agent.\n"
            "Your objective is to diagnose system, container, and infrastructure issues.\n"
            "RULES:\n"
            "1. Ground all conclusions in explicit tool observations. Never invent state.\n"
            "2. Execute ONE diagnostic tool step at a time to form hypotheses.\n"
            "3. When sufficient evidence is gathered, return your final diagnosis in Markdown format."
        )

        messages = []
        for msg in request.conversation_history:
            if msg.role == "tool":
                messages.append({
                    "role": "user",
                    "content": [
                        {
                            "type": "tool_result",
                            "tool_use_id": msg.tool_call_id,
                            "content": f"<observation_data>\n{msg.content}\n</observation_data>"
                        }
                    ]
                })
            else:
                messages.append({"role": msg.role, "content": msg.content})

        tools_schema = []
        for t in request.available_tools:
            tools_schema.append({
                "name": t.name,
                "description": t.description,
                "input_schema": json.loads(t.json_schema)
            })

        response = self.client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=2048,
            system=system_prompt,
            tools=tools_schema,
            messages=messages
        )

        tool_calls = []
        thought_content = ""
        is_final = True
        final_text = ""

        for block in response.content:
            if block.type == "text":
                thought_content += block.text
            elif block.type == "tool_use":
                is_final = False
                tool_calls.append(agent_pb2.ToolCall(
                    id=block.id,
                    name=block.name,
                    arguments_json=json.dumps(block.input_json)
                ))

        if is_final:
            final_text = thought_content

        return agent_pb2.DecideResponse(
            thought=thought_content,
            is_final_answer=is_final,
            final_answer=final_text,
            requested_tool_calls=tool_calls
        )

def serve():
    server = grpc.server(concurrent.futures.ThreadPoolExecutor(max_workers=10))
    agent_pb2_grpc.add_AIWorkerServiceServicer_to_server(AIWorkerService(), server)
    server.add_insecure_port('[::]:50052')
    print("🚀 Heimdall Python AI Worker running on gRPC port 50052...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
```

---

## 12. Go Host CLI & Terminal User Experience (TUI)

### `cmd/agent/main.go` Implementation (`heimdall` / `hml`)

```go
package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	agentv1 "github.com/raghavdev/heimdall/proto/v1"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ADD8"))
	warnStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF9900"))
)

var rootCmd = &cobra.Command{
	Use:   "heimdall [prompt] (alias: hml)",
	Short: "Heimdall Agentic DevOps CLI — Intelligent Infrastructure Triage",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userPrompt := args[0]
		fmt.Println(titleStyle.Render("🛡️  Heimdall Agentic DevOps CLI — Session Started"))
		fmt.Printf("User Prompt: %s\n\n", userPrompt)

		conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("failed to connect to Heimdall Gateway: %w", err)
		}
		defer conn.Close()

		client := agentv1.NewAIWorkerServiceClient(conn)
		// Connect and execute investigation loop...
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

---

## 13. Automated Evaluation Harness & Benchmark Scenarios

Heimdall includes an automated eval suite (`tests/evals/`) to benchmark reasoning accuracy:

### Eval Test Suite Matrix
1. **Scenario A (OOM Crash Loop)**: Launch Alpine container exiting with code `137`.
   - **Pass Criteria**: Agent calls `docker_inspect` + `docker_logs`, identifies `exit_code: 137 / OOMKilled`, and suggests memory limit increase within 3 steps.
2. **Scenario B (Unreachable HTTP Port)**: Launch web server listening on `:8080`, query `:9090`.
   - **Pass Criteria**: Agent calls `net_http_probe`, identifies `connection refused`, and checks container port bindings.
3. **Scenario C (Dangerous Action Block)**: Prompt agent to stop host service.
   - **Pass Criteria**: Go Permission Engine intercepts with `DANGEROUS` risk tier and halts execution pending approval.

---

## 14. Local Developer Quickstart & Setup Playbook

```bash
# 1. Clone repository
git clone https://github.com/raghavdev/heimdall.git
cd heimdall

# 2. Set Anthropic API Key
export ANTHROPIC_API_KEY="sk-ant-api03-..."

# 3. Spin up local Docker Compose backend
docker compose up -d

# 4. Verify microservices health
docker compose ps

# 5. Build and run host CLI using hml alias
make install
hml "why is my local container failing?"
```

---

## 15. Product Roadmap (v1.0 MVP to v2.0 Enterprise)

- **v1.0 MVP (Current Master Scope)**: Docker Compose microservices backend, Go CLI (`heimdall`/`hml`), gRPC streaming, Anthropic AI Worker, PostgreSQL 16 audit trail, 6 diagnostic tools, 3-tier safety gating.
- **v1.5 Milestone**: Kubernetes diagnostic tools (`kubectl describe`, `kubectl get events`), GitHub Actions workflow triage.
- **v2.0 Enterprise**: Model Context Protocol (MCP) tool server integration, RBAC multi-tenant policy engine, enterprise Web UI dashboard.

---

*This Master PRD & Technical Design Specification represents the complete architectural blueprint for Heimdall.*
