package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// NetHTTPProbeTool tests HTTP endpoint reachability, status code, latency, and response headers.
type NetHTTPProbeTool struct {
	client *http.Client
}

// NewNetHTTPProbeTool constructs a NetHTTPProbeTool with client controlled by request context.
func NewNetHTTPProbeTool() *NetHTTPProbeTool {
	return NewNetHTTPProbeToolWithClient(&http.Client{})
}

// NewNetHTTPProbeToolWithClient constructs a NetHTTPProbeTool with a custom HTTP client.
func NewNetHTTPProbeToolWithClient(client *http.Client) *NetHTTPProbeTool {
	return &NetHTTPProbeTool{
		client: client,
	}
}

// Name returns the unique tool identifier matching permission matrix.
func (t *NetHTTPProbeTool) Name() string {
	return "net_http_probe"
}

// Description explains the tool's diagnostic purpose.
func (t *NetHTTPProbeTool) Description() string {
	return "Probe an HTTP/HTTPS endpoint to inspect reachability, status code, latency in milliseconds, and headers."
}

// JSONSchema specifies the expected JSON schema for input arguments.
func (t *NetHTTPProbeTool) JSONSchema() string {
	return `{
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "description": "The HTTP or HTTPS URL to probe (e.g. 'http://localhost:8080/health')."
    },
    "timeout_seconds": {
      "type": "integer",
      "description": "Probe timeout in seconds (default: 5)."
    }
  },
  "required": ["url"]
}`
}

// RiskLevel returns the deterministic safety classification.
func (t *NetHTTPProbeTool) RiskLevel() permissions.RiskLevel {
	return permissions.RiskLevelReadOnly
}

// Execute performs an HTTP GET probe against the target URL.
func (t *NetHTTPProbeTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		URL            string `json:"url"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("malformed JSON arguments: %w", err)
	}
	targetURL := strings.TrimSpace(args.URL)
	if targetURL == "" {
		return "", fmt.Errorf("missing required argument: 'url'")
	}
	if err := ValidateSafeArgument("url", targetURL); err != nil {
		return "", err
	}
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	timeout := 5 * time.Second
	if args.TimeoutSeconds > 0 {
		timeout = time.Duration(args.TimeoutSeconds) * time.Second
	}
	if timeout > 30*time.Second {
		timeout = 30 * time.Second
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid HTTP request: %w", err)
	}

	start := time.Now()
	resp, err := t.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		// Connection failed/refused/timed out — return structured diagnostic observation
		unreachableRes := map[string]any{
			"url":         targetURL,
			"reachable":   false,
			"error":       err.Error(),
			"latency_ms":  latency,
			"status_code": 0,
		}
		bytes, _ := json.MarshalIndent(unreachableRes, "", "  ")
		return string(bytes), nil
	}
	defer resp.Body.Close()

	// Read body snippet (up to 2KB)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))

	headerMap := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headerMap[k] = v[0]
		}
	}

	res := map[string]any{
		"url":          targetURL,
		"reachable":    true,
		"status_code":  resp.StatusCode,
		"status":       resp.Status,
		"latency_ms":   latency,
		"headers":      headerMap,
		"body_snippet": string(bodyBytes),
	}

	bytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format http probe result: %w", err)
	}

	return string(bytes), nil
}
