package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"rapidbi/config"
)

// WebSearchTool provides web search capabilities using chromedp
type WebSearchTool struct {
	logger       func(string)
	searchEngine *config.SearchEngine
	proxyConfig  *config.ProxyConfig
}

// WebSearchInput represents the input for web search
type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"description=Search query keywords"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results to return (default: 10)"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	DisplayLink string `json:"display_link,omitempty"`
}

// NewWebSearchTool creates a new web search tool
func NewWebSearchTool(logger func(string), searchEngine *config.SearchEngine, proxyConfig *config.ProxyConfig) *WebSearchTool {
	return &WebSearchTool{
		logger:       logger,
		searchEngine: searchEngine,
		proxyConfig:  proxyConfig,
	}
}

// Info returns tool information for the LLM
func (t *WebSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	description := `Search the web for current information and data.

Use this tool to:
- Find current market data, prices, and trends
- Research competitors and industry information
- Get latest news and updates
- Compare products, services, or companies
- Gather real-time business intelligence

The tool returns structured search results with titles, URLs, and snippets.
Use the web_fetch tool to read full content from specific URLs.

⚠️ IMPORTANT:
- Web search is SLOW (may take 60-90 seconds)
- Use ONLY when external data is truly needed
- Prefer database queries for internal data
- Consider if the information is critical before searching

Examples:
- "latest smartphone market share 2026"
- "Tesla vs BYD sales comparison"
- "average SaaS pricing models"
- "top e-commerce platforms in China"`

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
				Desc:     "Maximum number of results to return (default: 5, max: 10)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the web search
func (t *WebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Parse input
	var input WebSearchInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	if input.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Set default max results (reduced to 5 for faster response)
	if input.MaxResults <= 0 {
		input.MaxResults = 5
	}
	if input.MaxResults > 10 {
		input.MaxResults = 10
	}

	if t.logger != nil {
		engineName := "Google" // Default
		if t.searchEngine != nil {
			engineName = t.searchEngine.Name
		}
		t.logger(fmt.Sprintf("[WEB-SEARCH] Searching for: %s (max: %d results, engine: %s)", 
			input.Query, input.MaxResults, engineName))
	}

	// Perform search using configured engine
	results, err := t.search(ctx, input.Query, input.MaxResults)
	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH] Search failed: %v", err))
		}
		return "", fmt.Errorf("search failed: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH] Found %d results", len(results)))
	}

	// Format results as JSON
	resultJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format results: %v", err)
	}

	return string(resultJSON), nil
}

// search performs a search using the configured search engine
func (t *WebSearchTool) search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// Determine which search engine to use
	engineURL := "www.google.com" // Default
	if t.searchEngine != nil && t.searchEngine.URL != "" {
		engineURL = t.searchEngine.URL
	}

	// Route to appropriate search function based on engine
	if strings.Contains(engineURL, "google.com") {
		return t.searchGoogle(ctx, query, maxResults, engineURL)
	} else if strings.Contains(engineURL, "bing.com") {
		return t.searchBing(ctx, query, maxResults, engineURL)
	} else if strings.Contains(engineURL, "baidu.com") {
		return t.searchBaidu(ctx, query, maxResults, engineURL)
	}

	// Default to Google-style search for custom engines
	return t.searchGoogle(ctx, query, maxResults, engineURL)
}

// searchGoogle performs a Google search using chromedp
func (t *WebSearchTool) searchGoogle(ctx context.Context, query string, maxResults int, baseURL string) ([]SearchResult, error) {
	// Create chromedp context with proxy support
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
	}

	// Add proxy configuration if enabled
	if t.proxyConfig != nil && t.proxyConfig.Enabled && t.proxyConfig.Tested {
		proxyURL := fmt.Sprintf("%s://%s:%d", 
			t.proxyConfig.Protocol, t.proxyConfig.Host, t.proxyConfig.Port)
		opts = append(opts, chromedp.ProxyServer(proxyURL))
		
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH] Using proxy: %s", proxyURL))
		}
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create chromedp context
	ctxWithCancel, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout for the entire operation (increased to 90s for slow networks/proxies)
	timeoutCtx, timeoutCancel := context.WithTimeout(ctxWithCancel, 90*time.Second)
	defer timeoutCancel()

	// Navigate to Google and perform search
	searchURL := fmt.Sprintf("https://%s/search?q=%s&num=%d", 
		baseURL, strings.ReplaceAll(query, " ", "+"), maxResults)

	var htmlContent string
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`#search`, chromedp.ByID),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load search results: %v", err)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Extract search results
	results := []SearchResult{}
	doc.Find("div.g").Each(func(i int, s *goquery.Selection) {
		if len(results) >= maxResults {
			return
		}

		// Extract title and URL
		titleElem := s.Find("h3")
		linkElem := s.Find("a")
		snippetElem := s.Find("div[data-sncf]")

		title := strings.TrimSpace(titleElem.Text())
		url, _ := linkElem.Attr("href")
		snippet := strings.TrimSpace(snippetElem.Text())

		// Extract display link
		displayLink := ""
		citeLinkElem := s.Find("cite")
		if citeLinkElem.Length() > 0 {
			displayLink = strings.TrimSpace(citeLinkElem.Text())
		}

		// Only add if we have at least title and URL
		if title != "" && url != "" {
			results = append(results, SearchResult{
				Title:       title,
				URL:         url,
				Snippet:     snippet,
				DisplayLink: displayLink,
			})
		}
	})

	return results, nil
}

// searchBing performs a Bing search using chromedp
func (t *WebSearchTool) searchBing(ctx context.Context, query string, maxResults int, baseURL string) ([]SearchResult, error) {
	// Create chromedp context with proxy support
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
	}

	// Add proxy configuration if enabled
	if t.proxyConfig != nil && t.proxyConfig.Enabled && t.proxyConfig.Tested {
		proxyURL := fmt.Sprintf("%s://%s:%d", 
			t.proxyConfig.Protocol, t.proxyConfig.Host, t.proxyConfig.Port)
		opts = append(opts, chromedp.ProxyServer(proxyURL))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctxWithCancel, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctxWithCancel, 90*time.Second)
	defer timeoutCancel()

	searchURL := fmt.Sprintf("https://%s/search?q=%s&count=%d", 
		baseURL, strings.ReplaceAll(query, " ", "+"), maxResults)

	var htmlContent string
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`#b_results`, chromedp.ByID),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load search results: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	results := []SearchResult{}
	doc.Find("li.b_algo").Each(func(i int, s *goquery.Selection) {
		if len(results) >= maxResults {
			return
		}

		titleElem := s.Find("h2 a")
		snippetElem := s.Find("p, .b_caption p")

		title := strings.TrimSpace(titleElem.Text())
		url, _ := titleElem.Attr("href")
		snippet := strings.TrimSpace(snippetElem.First().Text())

		if title != "" && url != "" {
			results = append(results, SearchResult{
				Title:   title,
				URL:     url,
				Snippet: snippet,
			})
		}
	})

	return results, nil
}

// searchBaidu performs a Baidu search using chromedp
func (t *WebSearchTool) searchBaidu(ctx context.Context, query string, maxResults int, baseURL string) ([]SearchResult, error) {
	// Create chromedp context with proxy support
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
	}

	// Add proxy configuration if enabled
	if t.proxyConfig != nil && t.proxyConfig.Enabled && t.proxyConfig.Tested {
		proxyURL := fmt.Sprintf("%s://%s:%d", 
			t.proxyConfig.Protocol, t.proxyConfig.Host, t.proxyConfig.Port)
		opts = append(opts, chromedp.ProxyServer(proxyURL))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctxWithCancel, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctxWithCancel, 90*time.Second)
	defer timeoutCancel()

	searchURL := fmt.Sprintf("https://%s/s?wd=%s&rn=%d", 
		baseURL, strings.ReplaceAll(query, " ", "+"), maxResults)

	var htmlContent string
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(`#content_left`, chromedp.ByID),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load search results: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	results := []SearchResult{}
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if len(results) >= maxResults {
			return
		}

		titleElem := s.Find("h3 a")
		snippetElem := s.Find(".c-abstract")

		title := strings.TrimSpace(titleElem.Text())
		url, _ := titleElem.Attr("href")
		snippet := strings.TrimSpace(snippetElem.Text())

		if title != "" && url != "" {
			results = append(results, SearchResult{
				Title:   title,
				URL:     url,
				Snippet: snippet,
			})
		}
	})

	return results, nil
}
