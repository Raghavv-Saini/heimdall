package agentv1_test

import (
	"testing"

	agentv1 "github.com/raghavdev/heimdall/proto/v1"
	"google.golang.org/protobuf/proto"
)

func TestDecideRequestSerialization(t *testing.T) {
	req := &agentv1.DecideRequest{
		SessionId:  "session-123",
		UserPrompt: "why is my web service crashing?",
		ConversationHistory: []*agentv1.Message{
			{
				Role:       "user",
				Content:    "why is my web service crashing?",
				ToolCallId: "",
				ToolName:   "",
			},
			{
				Role:       "assistant",
				Content:    "Let me inspect running containers.",
				ToolCallId: "",
				ToolName:   "",
			},
			{
				Role:       "tool",
				Content:    `{"ExitCode": 137, "OOMKilled": true}`,
				ToolCallId: "call-99",
				ToolName:   "docker_inspect",
			},
		},
		AvailableTools: []*agentv1.ToolDefinition{
			{
				Name:        "docker_inspect",
				Description: "Inspect container state and exit code",
				JsonSchema:  `{"type": "object", "properties": {"container_id": {"type": "string"}}}`,
				RiskLevel:   "READ_ONLY",
			},
			{
				Name:        "docker_restart",
				Description: "Restart a container",
				JsonSchema:  `{"type": "object", "properties": {"container_id": {"type": "string"}}}`,
				RiskLevel:   "SAFE_WRITE",
			},
		},
	}

	data, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal DecideRequest: %v", err)
	}

	unmarshaled := &agentv1.DecideRequest{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal DecideRequest: %v", err)
	}

	if unmarshaled.GetSessionId() != "session-123" {
		t.Errorf("expected session_id 'session-123', got '%s'", unmarshaled.GetSessionId())
	}
	if unmarshaled.GetUserPrompt() != "why is my web service crashing?" {
		t.Errorf("expected user_prompt 'why is my web service crashing?', got '%s'", unmarshaled.GetUserPrompt())
	}
	if len(unmarshaled.GetConversationHistory()) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(unmarshaled.GetConversationHistory()))
	}
	if unmarshaled.GetConversationHistory()[2].GetRole() != "tool" {
		t.Errorf("expected 3rd message role 'tool', got '%s'", unmarshaled.GetConversationHistory()[2].GetRole())
	}
	if unmarshaled.GetConversationHistory()[2].GetToolCallId() != "call-99" {
		t.Errorf("expected tool_call_id 'call-99', got '%s'", unmarshaled.GetConversationHistory()[2].GetToolCallId())
	}
	if len(unmarshaled.GetAvailableTools()) != 2 {
		t.Fatalf("expected 2 available tools, got %d", len(unmarshaled.GetAvailableTools()))
	}
	if unmarshaled.GetAvailableTools()[0].GetRiskLevel() != "READ_ONLY" {
		t.Errorf("expected risk_level 'READ_ONLY', got '%s'", unmarshaled.GetAvailableTools()[0].GetRiskLevel())
	}
	if unmarshaled.GetAvailableTools()[1].GetRiskLevel() != "SAFE_WRITE" {
		t.Errorf("expected risk_level 'SAFE_WRITE', got '%s'", unmarshaled.GetAvailableTools()[1].GetRiskLevel())
	}
}

func TestDecideResponseSerialization(t *testing.T) {
	resp := &agentv1.DecideResponse{
		Thought:       "The container was OOMKilled. We should restart it with higher limits.",
		IsFinalAnswer: false,
		FinalAnswer:   "",
		RequestedToolCalls: []*agentv1.ToolCall{
			{
				Id:            "call-100",
				Name:          "docker_restart",
				ArgumentsJson: `{"container_id": "web-api"}`,
			},
		},
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal DecideResponse: %v", err)
	}

	unmarshaled := &agentv1.DecideResponse{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal DecideResponse: %v", err)
	}

	if unmarshaled.GetThought() != "The container was OOMKilled. We should restart it with higher limits." {
		t.Errorf("unexpected thought: '%s'", unmarshaled.GetThought())
	}
	if unmarshaled.GetIsFinalAnswer() != false {
		t.Errorf("expected is_final_answer false, got true")
	}
	if len(unmarshaled.GetRequestedToolCalls()) != 1 {
		t.Fatalf("expected 1 requested tool call, got %d", len(unmarshaled.GetRequestedToolCalls()))
	}
	call := unmarshaled.GetRequestedToolCalls()[0]
	if call.GetId() != "call-100" || call.GetName() != "docker_restart" {
		t.Errorf("unexpected tool call: %+v", call)
	}
}

func TestDecideResponseFinalAnswer(t *testing.T) {
	resp := &agentv1.DecideResponse{
		Thought:       "Analysis complete.",
		IsFinalAnswer: true,
		FinalAnswer:   "Root Cause: Out of Memory (OOM) killer invoked.\nEvidence: Exit code 137.",
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal DecideResponse: %v", err)
	}

	unmarshaled := &agentv1.DecideResponse{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal DecideResponse: %v", err)
	}

	if !unmarshaled.GetIsFinalAnswer() {
		t.Errorf("expected is_final_answer true")
	}
	if len(unmarshaled.GetFinalAnswer()) == 0 {
		t.Errorf("expected non-empty final answer")
	}
}

func TestServiceInterfaces(t *testing.T) {
	var _ agentv1.AIWorkerServiceClient = nil
	var _ agentv1.AIWorkerServiceServer = nil
}
