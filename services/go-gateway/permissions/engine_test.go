package permissions_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

func TestStaticRiskMatrix(t *testing.T) {
	engine := permissions.NewEngine()

	tests := []struct {
		name             string
		toolName         string
		argsJSON         string
		expectedRisk     permissions.RiskLevel
		expectedApproval bool
	}{
		{
			name:             "docker_list_containers is READ_ONLY and auto-approved",
			toolName:         "docker_list_containers",
			argsJSON:         "{}",
			expectedRisk:     permissions.RiskLevelReadOnly,
			expectedApproval: false,
		},
		{
			name:             "docker_inspect is READ_ONLY and auto-approved",
			toolName:         "docker_inspect",
			argsJSON:         `{"container_id": "web-api"}`,
			expectedRisk:     permissions.RiskLevelReadOnly,
			expectedApproval: false,
		},
		{
			name:             "docker_logs is READ_ONLY and auto-approved",
			toolName:         "docker_logs",
			argsJSON:         `{"container_id": "web-api"}`,
			expectedRisk:     permissions.RiskLevelReadOnly,
			expectedApproval: false,
		},
		{
			name:             "systemd_status is READ_ONLY and auto-approved",
			toolName:         "systemd_status",
			argsJSON:         `{"service_name": "nginx"}`,
			expectedRisk:     permissions.RiskLevelReadOnly,
			expectedApproval: false,
		},
		{
			name:             "net_http_probe is READ_ONLY and auto-approved",
			toolName:         "net_http_probe",
			argsJSON:         `{"url": "http://localhost:8080/health"}`,
			expectedRisk:     permissions.RiskLevelReadOnly,
			expectedApproval: false,
		},
		{
			name:             "docker_restart is SAFE_WRITE and requires approval",
			toolName:         "docker_restart",
			argsJSON:         `{"container_id": "web-api"}`,
			expectedRisk:     permissions.RiskLevelSafeWrite,
			expectedApproval: true,
		},
		{
			name:             "docker_stop is DANGEROUS and requires approval",
			toolName:         "docker_stop",
			argsJSON:         `{"container_id": "web-api"}`,
			expectedRisk:     permissions.RiskLevelDangerous,
			expectedApproval: true,
		},
		{
			name:             "docker_remove is DANGEROUS and requires approval",
			toolName:         "docker_remove",
			argsJSON:         `{"container_id": "web-api"}`,
			expectedRisk:     permissions.RiskLevelDangerous,
			expectedApproval: true,
		},
		{
			name:             "systemd_restart is DANGEROUS and requires approval",
			toolName:         "systemd_restart",
			argsJSON:         `{"service_name": "postgresql"}`,
			expectedRisk:     permissions.RiskLevelDangerous,
			expectedApproval: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify GetRiskLevel method
			risk := engine.GetRiskLevel(tt.toolName)
			if risk != tt.expectedRisk {
				t.Errorf("GetRiskLevel(%s) = %v, expected %v", tt.toolName, risk, tt.expectedRisk)
			}

			// Verify Evaluate method
			decision, err := engine.Evaluate(tt.toolName, tt.argsJSON)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.RiskLevel != tt.expectedRisk {
				t.Errorf("decision.RiskLevel = %v, expected %v", decision.RiskLevel, tt.expectedRisk)
			}
			if decision.RequiresApproval != tt.expectedApproval {
				t.Errorf("decision.RequiresApproval = %v, expected %v", decision.RequiresApproval, tt.expectedApproval)
			}
			if decision.ToolName != tt.toolName {
				t.Errorf("decision.ToolName = %v, expected %v", decision.ToolName, tt.toolName)
			}
			if decision.Summary == "" {
				t.Errorf("decision.Summary is empty")
			}
		})
	}
}

func TestZeroTrustUnknownTools(t *testing.T) {
	engine := permissions.NewEngine()

	unknownTools := []string{
		"arbitrary_exec",
		"rm_rf",
		"custom_script",
		"docker_prune",
		"drop_database",
		"host_reboot",
		"unregistered_tool",
		"",
	}

	for _, tool := range unknownTools {
		t.Run("zero-trust for "+tool, func(t *testing.T) {
			risk := engine.GetRiskLevel(tool)
			if risk != permissions.RiskLevelDangerous {
				t.Errorf("GetRiskLevel(%s) = %v, expected %v (zero-trust)", tool, risk, permissions.RiskLevelDangerous)
			}

			decision, err := engine.Evaluate(tool, "{}")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.RiskLevel != permissions.RiskLevelDangerous {
				t.Errorf("decision.RiskLevel = %v, expected %v", decision.RiskLevel, permissions.RiskLevelDangerous)
			}
			if !decision.RequiresApproval {
				t.Errorf("expected RequiresApproval to be true for unknown tool %s", tool)
			}
			if !strings.Contains(decision.Summary, "zero-trust") && !strings.Contains(decision.Summary, "Unknown tool") {
				t.Errorf("summary does not mention zero-trust or unknown tool: %s", decision.Summary)
			}
		})
	}
}

func TestShellInjectionRejection(t *testing.T) {
	engine := permissions.NewEngine()

	payloads := []struct {
		name     string
		toolName string
		argsJSON string
	}{
		{
			name:     "semicolon command chaining",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web; rm -rf /"}`,
		},
		{
			name:     "logical AND command chaining",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web && echo hacked"}`,
		},
		{
			name:     "logical OR command chaining",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web || touch /tmp/pwned"}`,
		},
		{
			name:     "backtick command substitution",
			toolName: "docker_inspect",
			argsJSON: "{\"container_id\": \"web`whoami`\"}",
		},
		{
			name:     "subshell dollar paren substitution with command",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web$(id)"}`,
		},
		{
			name:     "subshell dollar paren empty",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web$()"}`,
		},
		{
			name:     "destructive rm -rf sequence",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "rm -rf /var/lib/docker"}`,
		},
		{
			name:     "newline command injection",
			toolName: "docker_inspect",
			argsJSON: "{\"container_id\": \"web\\nrm -rf /\"}",
		},
		{
			name:     "carriage return command injection",
			toolName: "docker_inspect",
			argsJSON: "{\"container_id\": \"web\\rreboot\"}",
		},
		{
			name:     "injection in nested JSON object",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web", "nested": {"param": "safe; id"}}`,
		},
		{
			name:     "injection in nested JSON array",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web", "filters": ["valid=1", "attack$(cat /etc/shadow)"]}`,
		},
		{
			name:     "injection in JSON object key",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web", "inject; rm -rf /": "val"}`,
		},
		{
			name:     "injection in toolName parameter itself",
			toolName: "docker_inspect; rm -rf /",
			argsJSON: `{"container_id": "web"}`,
		},
		{
			name:     "injection on safe write tool",
			toolName: "docker_restart",
			argsJSON: `{"container_id": "web-api && reboot"}`,
		},
		{
			name:     "injection on dangerous tool",
			toolName: "docker_stop",
			argsJSON: "{\"container_id\": \"web`whoami`\"}",
		},
	}

	for _, tt := range payloads {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Evaluate(tt.toolName, tt.argsJSON)
			if err == nil {
				t.Fatalf("expected error for payload '%s', but got decision: %+v", tt.name, decision)
			}
			if !errors.Is(err, permissions.ErrShellInjectionDetected) {
				t.Errorf("expected ErrShellInjectionDetected, got: %v", err)
			}
			if decision != nil {
				t.Errorf("expected decision to be nil on injection failure, got: %+v", decision)
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	engine := permissions.NewEngine()

	malformedCases := []struct {
		name     string
		argsJSON string
	}{
		{
			name:     "unclosed brace",
			argsJSON: `{"container_id": "web"`,
		},
		{
			name:     "truncated json",
			argsJSON: `{"container_id":`,
		},
		{
			name:     "bare string primitive",
			argsJSON: `"just a string"`,
		},
		{
			name:     "bare integer primitive",
			argsJSON: `12345`,
		},
		{
			name:     "bare array primitive",
			argsJSON: `["container_id", "web"]`,
		},
		{
			name:     "unquoted keys and values",
			argsJSON: `{container_id: web}`,
		},
	}

	for _, tt := range malformedCases {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Evaluate("docker_inspect", tt.argsJSON)
			if err == nil {
				t.Fatalf("expected ErrMalformedJSON for '%s', got decision: %+v", tt.name, decision)
			}
			if !errors.Is(err, permissions.ErrMalformedJSON) {
				t.Errorf("expected ErrMalformedJSON, got: %v", err)
			}
			if decision != nil {
				t.Errorf("expected decision to be nil on malformed JSON, got: %+v", decision)
			}
		})
	}
}

func TestRequiredArgumentValidation(t *testing.T) {
	engine := permissions.NewEngine()

	cases := []struct {
		name        string
		toolName    string
		argsJSON    string
		expectedErr error
	}{
		{
			name:        "docker_inspect missing container_id",
			toolName:    "docker_inspect",
			argsJSON:    `{}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "docker_inspect empty container_id",
			toolName:    "docker_inspect",
			argsJSON:    `{"container_id": ""}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "docker_inspect whitespace container_id",
			toolName:    "docker_inspect",
			argsJSON:    `{"container_id": "   "}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "docker_inspect container_id as integer",
			toolName:    "docker_inspect",
			argsJSON:    `{"container_id": 12345}`,
			expectedErr: permissions.ErrInvalidArgumentType,
		},
		{
			name:        "docker_inspect container_id as boolean",
			toolName:    "docker_inspect",
			argsJSON:    `{"container_id": true}`,
			expectedErr: permissions.ErrInvalidArgumentType,
		},
		{
			name:        "docker_inspect container_id as object",
			toolName:    "docker_inspect",
			argsJSON:    `{"container_id": {"name": "web"}}`,
			expectedErr: permissions.ErrInvalidArgumentType,
		},
		{
			name:        "docker_restart missing container_id",
			toolName:    "docker_restart",
			argsJSON:    `{}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "systemd_status missing service_name",
			toolName:    "systemd_status",
			argsJSON:    `{}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "systemd_status service_name as number",
			toolName:    "systemd_status",
			argsJSON:    `{"service_name": 42}`,
			expectedErr: permissions.ErrInvalidArgumentType,
		},
		{
			name:        "systemd_restart missing service_name",
			toolName:    "systemd_restart",
			argsJSON:    `{}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "net_http_probe missing url",
			toolName:    "net_http_probe",
			argsJSON:    `{}`,
			expectedErr: permissions.ErrMissingRequiredArgument,
		},
		{
			name:        "net_http_probe url as array",
			toolName:    "net_http_probe",
			argsJSON:    `{"url": ["http://localhost"]}`,
			expectedErr: permissions.ErrInvalidArgumentType,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Evaluate(tt.toolName, tt.argsJSON)
			if err == nil {
				t.Fatalf("expected error %v, got success decision: %+v", tt.expectedErr, decision)
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
			if decision != nil {
				t.Errorf("expected decision to be nil on argument error, got: %+v", decision)
			}
		})
	}
}

func TestLegitimateSafeInputs(t *testing.T) {
	engine := permissions.NewEngine()

	safeCases := []struct {
		name     string
		toolName string
		argsJSON string
	}{
		{
			name:     "standard hyphenated container name",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "web-api-gateway"}`,
		},
		{
			name:     "container name with dots",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "my-service.internal.v1"}`,
		},
		{
			name:     "container UUID identifier",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "f2c38214-f42f-436a-a319-f0ea94e05bfc"}`,
		},
		{
			name:     "container tag with colon",
			toolName: "docker_inspect",
			argsJSON: `{"container_id": "redis:7-alpine"}`,
		},
		{
			name:     "URL with query parameters containing single ampersand",
			toolName: "net_http_probe",
			argsJSON: `{"url": "http://localhost:8080/health?env=prod&tier=1"}`,
		},
		{
			name:     "HTTPS URL with custom port and path",
			toolName: "net_http_probe",
			argsJSON: `{"url": "https://api.example.com:8443/v1/metrics"}`,
		},
		{
			name:     "systemd service name with dot",
			toolName: "systemd_status",
			argsJSON: `{"service_name": "docker.service"}`,
		},
		{
			name:     "docker_list_containers with empty json object",
			toolName: "docker_list_containers",
			argsJSON: `{}`,
		},
		{
			name:     "docker_list_containers with empty string",
			toolName: "docker_list_containers",
			argsJSON: ``,
		},
		{
			name:     "docker_list_containers with whitespace string",
			toolName: "docker_list_containers",
			argsJSON: `   `,
		},
	}

	for _, tt := range safeCases {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Evaluate(tt.toolName, tt.argsJSON)
			if err != nil {
				t.Fatalf("unexpected error for legitimate input '%s': %v", tt.name, err)
			}
			if decision == nil {
				t.Fatalf("expected non-nil decision for '%s'", tt.name)
			}
			if decision.ToolName != tt.toolName {
				t.Errorf("decision.ToolName = %s, expected %s", decision.ToolName, tt.toolName)
			}
		})
	}
}

func TestDeterminism(t *testing.T) {
	engine := permissions.NewEngine()

	tool := "docker_restart"
	args := `{"container_id": "web-api"}`

	initialDecision, err := engine.Evaluate(tool, args)
	if err != nil {
		t.Fatalf("initial Evaluate failed: %v", err)
	}

	// Repeatedly evaluate the same input across 100 iterations
	for i := 0; i < 100; i++ {
		decision, err := engine.Evaluate(tool, args)
		if err != nil {
			t.Fatalf("iteration %d Evaluate failed: %v", i, err)
		}
		if decision.RiskLevel != initialDecision.RiskLevel {
			t.Fatalf("iteration %d RiskLevel mismatch: %v != %v", i, decision.RiskLevel, initialDecision.RiskLevel)
		}
		if decision.RequiresApproval != initialDecision.RequiresApproval {
			t.Fatalf("iteration %d RequiresApproval mismatch: %v != %v", i, decision.RequiresApproval, initialDecision.RequiresApproval)
		}
		if decision.Summary != initialDecision.Summary {
			t.Fatalf("iteration %d Summary mismatch: %s != %s", i, decision.Summary, initialDecision.Summary)
		}
	}
}
