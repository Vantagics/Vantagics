package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/AxT-Team/uapi-sdk-go/uapi"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"vantagedata/config"
)

// SearchAPITool provides unified search capabilities using various API services
type SearchAPITool struct {
	logger    func(string)
	apiConfig *config.SearchAPIConfig
}

// SearchAPIInput represents the input for API search
type SearchAPIInput struct {
	Query      string `json:"query" jsonschema:"description=Search query keywords"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results to return (default: 10)"`
}

// SearchAPIResult represents a single search result
type SearchAPIResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at,omitempty"`
}

// NewSearchAPITool creates a new search API tool
func NewSearchAPITool(logger func(string), apiConfig *config.SearchAPIConfig) (*SearchAPITool, error) {
	if apiConfig == nil {
		return nil, fmt.Errorf("API configuration is required")
	}

	// Validate configuration based on API type
	switch apiConfig.ID {
	case "uapi_pro":
		// UAPI Pro API key is optional for now (placeholder implementation)
		// When actual UAPI Pro integration is complete, uncomment the validation below:
		// if apiConfig.APIKey == "" {
		//     return nil, fmt.Errorf("UAPI Pro requires an API key")
		// }
	case "duckduckgo":
		// No API key required
	case "serper":
		if apiConfig.APIKey == "" {
			return nil, fmt.Errorf("Serper.dev requires an API key")
		}
	default:
		return nil, fmt.Errorf("unsupported search API: %s", apiConfig.ID)
	}

	return &SearchAPITool{
		logger:    logger,
		apiConfig: apiConfig,
	}, nil
}

// Info returns tool information for the LLM
func (t *SearchAPITool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	description := fmt.Sprintf(`Search the web using %s API.

Use this tool to:
- Find current market data, prices, and trends
- Research competitors and industry information
- Get latest news and updates
- Compare products, services, or companies
- Gather real-time business intelligence

The tool returns structured search results with titles, URLs, and snippets.
Use the web_fetch tool to read full content from specific URLs.

⚠️ IMPORTANT:
- Web search may take 10-30 seconds
- Use ONLY when external data is truly needed
- Prefer database queries for internal data
- Consider if the information is critical before searching

Current API: %s
%s`, t.apiConfig.Name, t.apiConfig.Name, t.apiConfig.Description)

	return &schema.ToolInfo{
		Name: "web_search",
		Desc: description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "Search query keywords (be specific for better results)",
				Required: true,
			},
			"max_results": {
				Type:     schema.Number,
				Desc:     "Maximum number of results to return (default: 10, max: 20)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the search
func (t *SearchAPITool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Parse input
	var input SearchAPIInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	if input.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Set default max results
	if input.MaxResults <= 0 {
		input.MaxResults = 10
	}
	if input.MaxResults > 20 {
		input.MaxResults = 20
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[SEARCH-API] Searching with %s: %s (max: %d results)",
			t.apiConfig.Name, input.Query, input.MaxResults))
	}

	// Route to appropriate search function
	var results []SearchAPIResult
	var err error

	switch t.apiConfig.ID {
	case "duckduckgo":
		results, err = t.searchDuckDuckGo(ctx, input.Query, input.MaxResults)
	case "serper":
		results, err = t.searchSerper(ctx, input.Query, input.MaxResults)
	case "uapi_pro":
		results, err = t.searchUAPIPro(ctx, input.Query, input.MaxResults)
	default:
		return "", fmt.Errorf("unsupported search API: %s", t.apiConfig.ID)
	}

	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[SEARCH-API] Search failed: %v", err))
		}
		return "", fmt.Errorf("search failed: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[SEARCH-API] Found %d results", len(results)))
	}

	// Format results as JSON
	resultJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format results: %v", err)
	}

	return string(resultJSON), nil
}

// searchDuckDuckGo performs a search using DuckDuckGo Instant Answer API
func (t *SearchAPITool) searchDuckDuckGo(ctx context.Context, query string, maxResults int) ([]SearchAPIResult, error) {
	if t.logger != nil {
		t.logger("[SEARCH-API] Using DuckDuckGo Instant Answer API...")
	}

	// DuckDuckGo Instant Answer API
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add User-Agent header
	req.Header.Set("User-Agent", "VantageData/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// DuckDuckGo may return 202 (Accepted) or 200 (OK)
	// Both are valid responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse DuckDuckGo response
	var ddgResp struct {
		Abstract       string `json:"Abstract"`
		AbstractText   string `json:"AbstractText"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		Heading        string `json:"Heading"`
		RelatedTopics  []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
			Icon     struct {
				URL string `json:"URL"`
			} `json:"Icon"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	results := []SearchAPIResult{}

	// Add abstract as first result if available
	if ddgResp.AbstractText != "" && ddgResp.AbstractURL != "" {
		title := ddgResp.AbstractSource
		if title == "" {
			title = ddgResp.Heading
		}
		if title == "" {
			title = "DuckDuckGo Result"
		}
		
		results = append(results, SearchAPIResult{
			Title:   title,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.AbstractText,
			Source:  "duckduckgo",
		})
	}

	// Add results from Results array
	for _, result := range ddgResp.Results {
		if len(results) >= maxResults {
			break
		}
		if result.Text != "" && result.FirstURL != "" {
			results = append(results, SearchAPIResult{
				Title:   result.Text[:min(len(result.Text), 100)],
				URL:     result.FirstURL,
				Snippet: result.Text,
				Source:  "duckduckgo",
			})
		}
	}

	// Add related topics
	for _, topic := range ddgResp.RelatedTopics {
		if len(results) >= maxResults {
			break
		}
		if topic.Text != "" && topic.FirstURL != "" {
			results = append(results, SearchAPIResult{
				Title:   topic.Text[:min(len(topic.Text), 100)],
				URL:     topic.FirstURL,
				Snippet: topic.Text,
				Source:  "duckduckgo",
			})
		}
	}

	// If no results found, return a helpful message
	if len(results) == 0 {
		// DuckDuckGo Instant Answer API may not have results for all queries
		// This is normal - it focuses on instant answers rather than web search
		return []SearchAPIResult{
			{
				Title:   "No instant answers available",
				URL:     fmt.Sprintf("https://duckduckgo.com/?q=%s", url.QueryEscape(query)),
				Snippet: fmt.Sprintf("DuckDuckGo Instant Answer API did not return results for '%s'. This API provides instant answers for specific queries. For general web search, visit DuckDuckGo directly.", query),
				Source:  "duckduckgo",
			},
		}, nil
	}

	return results, nil
}

// searchSerper performs a search using Serper.dev (Google Search API)
func (t *SearchAPITool) searchSerper(ctx context.Context, query string, maxResults int) ([]SearchAPIResult, error) {
	if t.logger != nil {
		t.logger("[SEARCH-API] Using Serper.dev (Google Search)...")
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"q":   query,
		"num": maxResults,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Create HTTP request
	apiURL := "https://google.serper.dev/search"
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("X-API-KEY", t.apiConfig.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse Serper.dev response
	var serperResp struct {
		Organic []struct {
			Title    string `json:"title"`
			Link     string `json:"link"`
			Snippet  string `json:"snippet"`
			Date     string `json:"date"`
			Position int    `json:"position"`
		} `json:"organic"`
		AnswerBox struct {
			Answer  string `json:"answer"`
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"answerBox"`
		KnowledgeGraph struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Link        string `json:"link"`
		} `json:"knowledgeGraph"`
	}

	if err := json.Unmarshal(body, &serperResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	results := []SearchAPIResult{}

	// Add answer box if available
	if serperResp.AnswerBox.Answer != "" {
		title := serperResp.AnswerBox.Title
		if title == "" {
			title = "Answer"
		}
		results = append(results, SearchAPIResult{
			Title:   title,
			URL:     serperResp.AnswerBox.Link,
			Snippet: serperResp.AnswerBox.Answer,
			Source:  "serper",
		})
	}

	// Add knowledge graph if available
	if serperResp.KnowledgeGraph.Title != "" {
		results = append(results, SearchAPIResult{
			Title:   serperResp.KnowledgeGraph.Title,
			URL:     serperResp.KnowledgeGraph.Link,
			Snippet: serperResp.KnowledgeGraph.Description,
			Source:  "serper",
		})
	}

	// Add organic results
	for _, result := range serperResp.Organic {
		if len(results) >= maxResults {
			break
		}
		publishedAt := ""
		if result.Date != "" {
			publishedAt = result.Date
		}
		results = append(results, SearchAPIResult{
			Title:       result.Title,
			URL:         result.Link,
			Snippet:     result.Snippet,
			Source:      "serper",
			PublishedAt: publishedAt,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for query: %s", query)
	}

	return results, nil
}

// searchUAPIPro performs a search using UAPI Pro
func (t *SearchAPITool) searchUAPIPro(ctx context.Context, query string, maxResults int) ([]SearchAPIResult, error) {
	if t.logger != nil {
		t.logger("[SEARCH-API] Using UAPI Pro...")
	}

	// Create UAPI client
	baseURL := "https://api.uapi.nl"
	client := uapi.New(baseURL, t.apiConfig.APIKey)

	// Use UAPI's search functionality
	// Note: This is a placeholder - actual implementation depends on UAPI SDK's search methods
	// For now, we'll return a structured response indicating the service is available
	
	results := []SearchAPIResult{
		{
			Title:   fmt.Sprintf("UAPI Pro Search: %s", query),
			URL:     "https://docs.uapi.nl/",
			Snippet: fmt.Sprintf("UAPI Pro search for '%s' - Implementation pending SDK documentation", query),
			Source:  "uapi_pro",
		},
	}

	// TODO: Implement actual UAPI Pro search when SDK documentation is available
	// Example structure (to be updated):
	// resp, err := client.Search().Query(ctx, query, maxResults)
	// if err != nil {
	//     return nil, err
	// }
	// Parse resp and populate results

	_ = client // Use client to avoid unused variable error

	return results, nil
}
