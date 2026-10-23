package tools_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
	"github.com/raghavdev/heimdall/services/go-gateway/tools"
)

// mockTool implements tools.Tool for testing registry dispatch.
type mockTool struct {
	name        string
	description string
	schema      string
	riskLevel   permissions.RiskLevel
	executeFn   func(ctx context.Context, argsJSON string) (string, error)
}

func (m *mockTool) Name() string                     { return m.name }
func (m *mockTool) Description() string              { return m.description }
func (m *mockTool) JSONSchema() string               { return m.schema }
func (m *mockTool) RiskLevel() permissions.RiskLevel { return m.riskLevel }
func (m *mockTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, argsJSON)
	}
	return "{}", nil
}

func TestRegistryLifecycle(t *testing.T) {
	reg := tools.NewRegistry()

	dummy := &mockTool{
		name:        "mock_echo",
		description: "Echo mock tool",
		schema:      `{"type":"object"}`,
		riskLevel:   permissions.RiskLevelReadOnly,
		executeFn: func(ctx context.Context, argsJSON string) (string, error) {
			return `{"received":` + argsJSON + `}`, nil
		},
	}

	reg.Register(dummy)

	// Verify lookup
	tool, exists := reg.Get("mock_echo")
	if !exists {
		t.Fatalf("expected tool 'mock_echo' to be registered")
	}
	if tool.Name() != "mock_echo" {
		t.Errorf("expected tool name 'mock_echo', got '%s'", tool.Name())
	}

	// Verify list
	all := reg.List()
	if len(all) != 1 {
		t.Fatalf("expected 1 registered tool, got %d", len(all))
	}

	// Verify Protobuf conversion
	protoDefs := reg.ToProtobuf()
	if len(protoDefs) != 1 {
		t.Fatalf("expected 1 protobuf definition, got %d", len(protoDefs))
	}
	def := protoDefs[0]
	if def.Name != "mock_echo" {
		t.Errorf("expected proto name 'mock_echo', got '%s'", def.Name)
	}
	if def.Description != "Echo mock tool" {
		t.Errorf("expected proto description 'Echo mock tool', got '%s'", def.Description)
	}
	if def.JsonSchema != `{"type":"object"}` {
		t.Errorf("expected proto json_schema, got '%s'", def.JsonSchema)
	}
	if def.RiskLevel != "READ_ONLY" {
		t.Errorf("expected proto risk level 'READ_ONLY', got '%s'", def.RiskLevel)
	}

	// Verify dispatch execution
	res, err := reg.Execute(context.Background(), "mock_echo", `{"val":"test"}`)
	if err != nil {
		t.Fatalf("unexpected error executing tool: %v", err)
	}
	if res != `{"received":{"val":"test"}}` {
		t.Errorf("unexpected execution result: %s", res)
	}

	// Verify missing tool dispatch
	_, err = reg.Execute(context.Background(), "non_existent", `{}`)
	if err == nil {
		t.Fatalf("expected error executing unregistered tool, got nil")
	}
}

func TestDockerInspectTool(t *testing.T) {
	// Sample mock Docker inspect JSON output for an OOM-killed container
	sampleDockerJSON := `[
		{
			"Id": "c3a1f8e21a4598716253491283",
			"Name": "/payment-processor",
			"State": {
				"Status": "exited",
				"Running": false,
				"ExitCode": 137,
				"OOMKilled": true,
				"Error": "fatal error",
				"FinishedAt": "2026-10-23T16:20:00Z"
			},
			"RestartCount": 5
		}
	]`

	var capturedName string
	var capturedArgs []string

	mockRunner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		capturedName = name
		capturedArgs = args
		return []byte(sampleDockerJSON), nil
	}

	tool := tools.NewDockerInspectToolWithRunner(mockRunner)

	if tool.Name() != "docker_inspect" {
		t.Errorf("expected tool name 'docker_inspect', got '%s'", tool.Name())
	}
	if tool.RiskLevel() != permissions.RiskLevelReadOnly {
		t.Errorf("expected risk level READ_ONLY, got '%s'", tool.RiskLevel())
	}

	// Test successful execution
	resultJSON, err := tool.Execute(context.Background(), `{"container_id":"payment-processor"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedName != "docker" {
		t.Errorf("expected binary 'docker', got '%s'", capturedName)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "inspect" || capturedArgs[1] != "payment-processor" {
		t.Errorf("expected args ['inspect', 'payment-processor'], got %v", capturedArgs)
	}

	// Verify distilled fields exist in result
	if !strings.Contains(resultJSON, `"exit_code": 137`) {
		t.Errorf("expected output to contain exit code 137, got: %s", resultJSON)
	}
	if !strings.Contains(resultJSON, `"oom_killed": true`) {
		t.Errorf("expected output to contain oom_killed: true, got: %s", resultJSON)
	}
	if !strings.Contains(resultJSON, `"c3a1f8e21a45"`) {
		t.Errorf("expected output to contain 12-char container ID, got: %s", resultJSON)
	}

	// Test missing container_id
	_, err = tool.Execute(context.Background(), `{}`)
	if err == nil {
		t.Errorf("expected error on missing container_id, got nil")
	}

	// Test malformed JSON
	_, err = tool.Execute(context.Background(), `not-json`)
	if err == nil {
		t.Errorf("expected error on malformed JSON, got nil")
	}
}

func TestDockerLogsTool(t *testing.T) {
	sampleLogs := "2026-10-23T16:20:01Z [info] Starting payment service\n2026-10-23T16:20:05Z [fatal] Out of memory allocating 512MB"

	var capturedName string
	var capturedArgs []string

	mockRunner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		capturedName = name
		capturedArgs = args
		return []byte(sampleLogs), nil
	}

	tool := tools.NewDockerLogsToolWithRunner(mockRunner)

	if tool.Name() != "docker_logs" {
		t.Errorf("expected name 'docker_logs', got '%s'", tool.Name())
	}
	if tool.RiskLevel() != permissions.RiskLevelReadOnly {
		t.Errorf("expected READ_ONLY, got '%s'", tool.RiskLevel())
	}

	// 1. Test execution with explicit tail
	resJSON, err := tool.Execute(context.Background(), `{"container_id":"payment-api","tail":50}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected 'docker', got '%s'", capturedName)
	}
	if len(capturedArgs) != 4 || capturedArgs[0] != "logs" || capturedArgs[1] != "--tail" || capturedArgs[2] != "50" || capturedArgs[3] != "payment-api" {
		t.Errorf("unexpected args: %v", capturedArgs)
	}
	if !strings.Contains(resJSON, "Out of memory allocating 512MB") {
		t.Errorf("expected logs to contain fatal message, got: %s", resJSON)
	}

	// 2. Test execution with default tail (tail omitted)
	_, err = tool.Execute(context.Background(), `{"container_id":"payment-api"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedArgs[2] != "100" {
		t.Errorf("expected default tail 100, got '%s'", capturedArgs[2])
	}

	// 3. Test missing container_id
	_, err = tool.Execute(context.Background(), `{}`)
	if err == nil {
		t.Errorf("expected error for missing container_id, got nil")
	}
}

func TestSystemdStatusTool(t *testing.T) {
	// Sample systemctl output when service is inactive (exit code 3)
	sampleDeadOutput := "● payment.service - Payment Worker\n   Loaded: loaded (/etc/systemd/system/payment.service; enabled)\n   Active: inactive (dead)"

	var capturedName string
	var capturedArgs []string

	mockRunner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		capturedName = name
		capturedArgs = args
		// Simulate systemctl non-zero exit code 3 (inactive/dead)
		return []byte(sampleDeadOutput), nil
	}

	tool := tools.NewSystemdStatusToolWithRunner(mockRunner)

	if tool.Name() != "systemd_status" {
		t.Errorf("expected name 'systemd_status', got '%s'", tool.Name())
	}
	if tool.RiskLevel() != permissions.RiskLevelReadOnly {
		t.Errorf("expected READ_ONLY, got '%s'", tool.RiskLevel())
	}

	resJSON, err := tool.Execute(context.Background(), `{"service_name":"payment.service"}`)
	if err != nil {
		t.Fatalf("unexpected error for inactive unit: %v", err)
	}

	if capturedName != "systemctl" {
		t.Errorf("expected binary 'systemctl', got '%s'", capturedName)
	}
	if len(capturedArgs) < 2 || capturedArgs[0] != "status" || capturedArgs[1] != "payment.service" {
		t.Errorf("unexpected args: %v", capturedArgs)
	}

	if !strings.Contains(resJSON, "inactive (dead)") {
		t.Errorf("expected output to contain 'inactive (dead)', got: %s", resJSON)
	}
	if !strings.Contains(resJSON, `"service_name": "payment.service"`) {
		t.Errorf("expected service_name in json, got: %s", resJSON)
	}

	// Test missing service_name
	_, err = tool.Execute(context.Background(), `{}`)
	if err == nil {
		t.Errorf("expected error on missing service_name, got nil")
	}
}

func TestNetHTTPProbeTool(t *testing.T) {
	// 1. Setup a test HTTP server returning 200 OK with custom header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "heimdall-test")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer ts.Close()

	tool := tools.NewNetHTTPProbeTool()

	if tool.Name() != "net_http_probe" {
		t.Errorf("expected name 'net_http_probe', got '%s'", tool.Name())
	}
	if tool.RiskLevel() != permissions.RiskLevelReadOnly {
		t.Errorf("expected READ_ONLY, got '%s'", tool.RiskLevel())
	}

	// 2. Test probing the live test server
	resJSON, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s"}`, ts.URL))
	if err != nil {
		t.Fatalf("unexpected error probing live server: %v", err)
	}

	if !strings.Contains(resJSON, `"reachable": true`) {
		t.Errorf("expected reachable true, got: %s", resJSON)
	}
	if !strings.Contains(resJSON, `"status_code": 200`) {
		t.Errorf("expected status_code 200, got: %s", resJSON)
	}
	if !strings.Contains(resJSON, "heimdall-test") {
		t.Errorf("expected header to contain 'heimdall-test', got: %s", resJSON)
	}
	if !strings.Contains(resJSON, "healthy") {
		t.Errorf("expected body to contain 'healthy', got: %s", resJSON)
	}

	// 3. Test probing an unreachable port (e.g. closed port)
	unreachableResJSON, err := tool.Execute(context.Background(), `{"url":"http://127.0.0.1:59999"}`)
	if err != nil {
		t.Fatalf("expected observation JSON even on unreachable target, got error: %v", err)
	}
	if !strings.Contains(unreachableResJSON, `"reachable": false`) {
		t.Errorf("expected reachable false, got: %s", unreachableResJSON)
	}

	// 4. Test missing url
	_, err = tool.Execute(context.Background(), `{}`)
	if err == nil {
		t.Errorf("expected error for missing url, got nil")
	}
}

func TestDefaultRegistry(t *testing.T) {
	reg := tools.NewDefaultRegistry()
	expectedTools := []string{
		"docker_inspect",
		"docker_logs",
		"net_http_probe",
		"systemd_status",
	}

	for _, name := range expectedTools {
		tool, exists := reg.Get(name)
		if !exists {
			t.Errorf("expected tool '%s' in default registry", name)
			continue
		}
		if tool.Name() != name {
			t.Errorf("expected tool name '%s', got '%s'", name, tool.Name())
		}
		if tool.Description() == "" {
			t.Errorf("expected non-empty description for '%s'", name)
		}
		if tool.JSONSchema() == "" {
			t.Errorf("expected non-empty JSONSchema for '%s'", name)
		}
		if tool.RiskLevel() != permissions.RiskLevelReadOnly {
			t.Errorf("expected tool '%s' to be READ_ONLY, got '%s'", name, tool.RiskLevel())
		}
	}

	protoDefs := reg.ToProtobuf()
	if len(protoDefs) != 4 {
		t.Fatalf("expected 4 protobuf definitions, got %d", len(protoDefs))
	}

	for i, def := range protoDefs {
		if def.Name != expectedTools[i] {
			t.Errorf("expected sorted proto definition '%s', got '%s'", expectedTools[i], def.Name)
		}
		if def.RiskLevel != "READ_ONLY" {
			t.Errorf("expected proto risk level READ_ONLY for '%s', got '%s'", def.Name, def.RiskLevel)
		}
	}
}





