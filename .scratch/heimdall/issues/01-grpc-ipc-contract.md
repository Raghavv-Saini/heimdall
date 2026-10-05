# 01: Shared Protocol Buffer & gRPC IPC Contract

**What to build:** The cross-language interface contract between the Go Gateway and Python AI Worker. When an agent or developer invokes the build, it compiles type-safe gRPC client and server stubs for both Go (`agent_service.pb.go`, `agent_service_grpc.pb.go`) and Python (`agent_service_pb2.py`, `agent_service_pb2_grpc.py`). It defines polymorphic tool parameters as stringified JSON so new diagnostic tools can be added without altering the proto schema.

**Blocked by:** None (can start immediately)

**Assigned to:** Agent 1 (or Agent 2)

**Status:** complete

- [x] `proto/v1/agent_service.proto` defines `AIWorkerService.DecideNextStep` with `DecideRequest` and `DecideResponse`.
- [x] `Message` schema supports `user`, `assistant`, and `tool` roles with `tool_call_id`.
- [x] `ToolDefinition` and `ToolCall` represent schemas and parameters as JSON strings.
- [x] Running `make proto` generates valid, error-free stubs for both Go and Python.
