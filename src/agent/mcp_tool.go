package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"rapidbi/config"
)

// MCPTool provides access to configured MCP services
type MCPTool struct {
	services []config.MCPService
	logger   func(string)
}

// MCPToolInput represents the input for MCP tool calls
type MCPToolInput struct {
	ServiceName string                 `json:"service_name" jsonschema:"description=Name of the MCP service to use"`
	Method      string                 `json:"method" jsonschema:"description=HTTP method (GET, POST, etc.)"`
	Endpoint    string                 `json:"endpoint" jsonschema:"description=API endpoint path (will be appended to service URL)"`
	Body        map[string]interface{} `json:"body,omitempty" jsonschema:"description=Request body for POST/PUT requests"`
	Headers     map[string]string      `json:"headers,omitempty" jsonschema:"description=Additional HTTP headers"`
}

// NewMCPTool creates a new MCP tool with configured services
func NewMCPTool(services []config.MCPService, logger func(string)) *MCPTool {
	// Filter only enabled AND tested services
	enabledServices := []config.MCPService{}
	skippedServices := []string{}
	
	for _, svc := range services {
		if svc.Enabled && svc.Tested {
			enabledServices = append(enabledServices, svc)
		} else if svc.Enabled && !svc.Tested {
			// Service is enabled but not tested - skip it and log warning
			skippedServices = append(skippedServices, svc.Name)
		}
	}

	// Log warning about skipped services
	if len(skippedServices) > 0 && logger != nil {
		logger(fmt.Sprintf("[MCP-WARNING] Skipped %d enabled but untested service(s): %s. Please test these services before use.",
			len(skippedServices), strings.Join(skippedServices, ", ")))
	}

	return &MCPTool{
		services: enabledServices,
		logger:   logger,
	}
}

// Info returns tool information for the LLM
func (t *MCPTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	// Build description with available services
	serviceList := ""
	for i, svc := range t.services {
		serviceList += fmt.Sprintf("\n%d. %s: %s (URL: %s)", i+1, svc.Name, svc.Description, svc.URL)
	}

	description := fmt.Sprintf(`Call external MCP (Model Context Protocol) services for extended capabilities.

Available services:%s

Use this tool to:
- Search the web for current information
- Access external APIs and services
- Retrieve real-time data
- Perform operations not available in the local database

Important:
- Specify the service_name exactly as listed above
- Use appropriate HTTP method (GET for retrieval, POST for submission)
- Provide endpoint path relative to the service URL
- Include necessary headers and body data`, serviceList)

	if len(t.services) == 0 {
		description = "No MCP services are currently configured and enabled. This tool is not available."
	}

	return &schema.ToolInfo{
		Name: "mcp_service",
		Desc: description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"service_name": {
				Type:     schema.String,
				Desc:     "Name of the MCP service to use (must match one of the available services)",
				Required: true,
			},
			"method": {
				Type:     schema.String,
				Desc:     "HTTP method: GET, POST, PUT, DELETE, etc.",
				Required: true,
			},
			"endpoint": {
				Type:     schema.String,
				Desc:     "API endpoint path (will be appended to service base URL)",
				Required: true,
			},
			"body": {
				Type:     schema.Object,
				Desc:     "Request body as JSON object (for POST/PUT requests)",
				Required: false,
			},
			"headers": {
				Type:     schema.Object,
				Desc:     "Additional HTTP headers as key-value pairs",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the MCP service call
func (t *MCPTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if len(t.services) == 0 {
		return "", fmt.Errorf("no MCP services are configured")
	}

	// Parse input
	var input MCPToolInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	// Find the service
	var service *config.MCPService
	for i := range t.services {
		if strings.EqualFold(t.services[i].Name, input.ServiceName) {
			service = &t.services[i]
			break
		}
	}

	if service == nil {
		availableServices := []string{}
		for _, svc := range t.services {
			availableServices = append(availableServices, svc.Name)
		}
		return "", fmt.Errorf("service '%s' not found. Available services: %s", 
			input.ServiceName, strings.Join(availableServices, ", "))
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[MCP] Calling service: %s, method: %s, endpoint: %s", 
			service.Name, input.Method, input.Endpoint))
	}

	// Build full URL
	baseURL := strings.TrimRight(service.URL, "/")
	endpoint := strings.TrimLeft(input.Endpoint, "/")
	fullURL := fmt.Sprintf("%s/%s", baseURL, endpoint)

	// Prepare request body
	var bodyReader io.Reader
	if input.Body != nil && len(input.Body) > 0 {
		bodyBytes, err := json.Marshal(input.Body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(input.Method), fullURL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RapidBI-Agent/1.0")

	// Add custom headers
	for key, value := range input.Headers {
		req.Header.Set(key, value)
	}

	// Execute request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[MCP] Response from %s: status=%d, duration=%v, size=%d bytes", 
			service.Name, resp.StatusCode, duration, len(respBody)))
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("service returned error status %d: %s", resp.StatusCode, string(respBody))
	}

	// Try to format JSON response nicely
	var jsonResp interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		// Valid JSON, format it
		formatted, err := json.MarshalIndent(jsonResp, "", "  ")
		if err == nil {
			return string(formatted), nil
		}
	}

	// Return raw response if not JSON
	return string(respBody), nil
}

// GetAvailableServices returns a list of available service names
func (t *MCPTool) GetAvailableServices() []string {
	services := []string{}
	for _, svc := range t.services {
		services = append(services, svc.Name)
	}
	return services
}

// HasServices returns true if there are any enabled services
func (t *MCPTool) HasServices() bool {
	return len(t.services) > 0
}
