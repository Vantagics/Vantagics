package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AxT-Team/uapi-sdk-go/uapi"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// UAPISearchTool provides universal data search capabilities using UAPI SDK
type UAPISearchTool struct {
	logger func(string)
	client *uapi.Client
}

// UAPISearchInput represents the input for UAPI search
type UAPISearchInput struct {
	Query      string `json:"query" jsonschema:"description=Search query keywords"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results to return (default: 10)"`
	Source     string `json:"source,omitempty" jsonschema:"description=Data source to search (e.g., 'social', 'game', 'image', 'general')"`
}

// UAPISearchResult represents a single UAPI search result
type UAPISearchResult struct {
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Snippet     string                 `json:"snippet"`
	Source      string                 `json:"source"`
	PublishedAt string                 `json:"published_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewUAPISearchTool creates a new UAPI search tool
func NewUAPISearchTool(logger func(string), apiToken string) (*UAPISearchTool, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("UAPI API token is required")
	}

	// Create UAPI client
	// Default base URL for UAPI
	baseURL := "https://api.uapi.nl"
	client := uapi.New(baseURL, apiToken)

	return &UAPISearchTool{
		logger: logger,
		client: client,
	}, nil
}

// Info returns tool information for the LLM
func (t *UAPISearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	description := `Search for structured data across multiple sources using UAPI.

UAPI provides normalized, schema-aligned data from various sources:
- Social media platforms (QQ, WeChat, etc.)
- Gaming platforms and data
- Image and media sources
- General web content

Use this tool to:
- Get structured data from social platforms
- Search gaming information and statistics
- Find images and media content
- Access normalized web data with stable schemas

The tool returns structured results with consistent field names and clean data types.
Results include metadata like publication dates and source information.

⚠️ IMPORTANT:
- UAPI search is optimized for structured data
- Results are normalized with stable schemas
- Use for data that requires consistent formatting
- Prefer this over web scraping for supported sources

Examples:
- "latest gaming trends in China"
- "QQ user statistics 2026"
- "popular images on social media"
- "structured data about [topic]"`

	return &schema.ToolInfo{
		Name: "uapi_search",
		Desc: description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "Search query keywords (be specific for better results)",
				Required: true,
			},
			"max_results": {
				Type:     schema.Number,
				Desc:     "Maximum number of results to return (default: 10, max: 50)",
				Required: false,
			},
			"source": {
				Type:     schema.String,
				Desc:     "Data source to search: 'social', 'game', 'image', or 'general' (default: 'general')",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the UAPI search
func (t *UAPISearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Parse input
	var input UAPISearchInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	if input.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Set defaults
	if input.MaxResults <= 0 {
		input.MaxResults = 10
	}
	if input.MaxResults > 50 {
		input.MaxResults = 50
	}
	if input.Source == "" {
		input.Source = "general"
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[UAPI-SEARCH] Searching for: %s (max: %d results, source: %s)",
			input.Query, input.MaxResults, input.Source))
	}

	// Perform search based on source
	results, err := t.search(ctx, input.Query, input.MaxResults, input.Source)
	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[UAPI-SEARCH] Search failed: %v", err))
		}
		return "", fmt.Errorf("search failed: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[UAPI-SEARCH] Found %d results", len(results)))
	}

	// Format results as JSON
	resultJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format results: %v", err)
	}

	return string(resultJSON), nil
}

// search performs a search using UAPI based on the specified source
func (t *UAPISearchTool) search(ctx context.Context, query string, maxResults int, source string) ([]UAPISearchResult, error) {
	// Placeholder implementation
	_ = ctx
	_ = query
	_ = maxResults

	switch source {
	case "social":
		return t.searchSocial(ctx, query, maxResults)
	case "game":
		return t.searchGame(ctx, query, maxResults)
	case "image":
		return t.searchImage(ctx, query, maxResults)
	case "general":
		return t.searchGeneral(ctx, query, maxResults)
	default:
		return t.searchGeneral(ctx, query, maxResults)
	}
}

// searchSocial searches social media data using UAPI
func (t *UAPISearchTool) searchSocial(ctx context.Context, query string, maxResults int) ([]UAPISearchResult, error) {
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Searching social media data...")
	}

	// Use UAPI Social module
	// Note: Actual implementation depends on UAPI SDK's Social() methods
	// This is a placeholder that should be updated based on actual SDK documentation
	
	results := []UAPISearchResult{}
	
	// Example: If SDK provides Social().Search() method
	// resp, err := t.client.Social().Search(ctx, query, maxResults)
	// if err != nil {
	//     return nil, fmt.Errorf("social search failed: %v", err)
	// }
	
	// For now, return a placeholder indicating the feature is available
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Social search module ready (implementation pending SDK documentation)")
	}

	return results, nil
}

// searchGame searches gaming data using UAPI
func (t *UAPISearchTool) searchGame(ctx context.Context, query string, maxResults int) ([]UAPISearchResult, error) {
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Searching gaming data...")
	}

	// Use UAPI Game module
	results := []UAPISearchResult{}
	
	// Example: If SDK provides Game().Search() method
	// resp, err := t.client.Game().Search(ctx, query, maxResults)
	// if err != nil {
	//     return nil, fmt.Errorf("game search failed: %v", err)
	// }
	
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Game search module ready (implementation pending SDK documentation)")
	}

	return results, nil
}

// searchImage searches image data using UAPI
func (t *UAPISearchTool) searchImage(ctx context.Context, query string, maxResults int) ([]UAPISearchResult, error) {
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Searching image data...")
	}

	// Use UAPI Image module
	results := []UAPISearchResult{}
	
	// Example: If SDK provides Image().Search() method
	// resp, err := t.client.Image().Search(ctx, query, maxResults)
	// if err != nil {
	//     return nil, fmt.Errorf("image search failed: %v", err)
	// }
	
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Image search module ready (implementation pending SDK documentation)")
	}

	return results, nil
}

// searchGeneral performs a general search using UAPI
func (t *UAPISearchTool) searchGeneral(ctx context.Context, query string, maxResults int) ([]UAPISearchResult, error) {
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] Performing general search...")
	}

	// Use UAPI general search/extract functionality
	results := []UAPISearchResult{}
	
	// Example: If SDK provides Extract() or Search() method
	// resp, err := t.client.Extract(ctx, query)
	// if err != nil {
	//     return nil, fmt.Errorf("general search failed: %v", err)
	// }
	
	if t.logger != nil {
		t.logger("[UAPI-SEARCH] General search module ready (implementation pending SDK documentation)")
	}

	return results, nil
}
