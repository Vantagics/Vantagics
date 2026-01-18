package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"rapidbi/config"
)

// extractDomainName extracts a clean domain name from URL for file naming
func extractDomainName(baseURL string) string {
	// Remove protocol if present
	domain := strings.TrimPrefix(baseURL, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	
	// Remove path and query
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, "?"); idx != -1 {
		domain = domain[:idx]
	}
	
	// Replace dots with underscores for filename
	domain = strings.ReplaceAll(domain, ".", "_")
	
	// Remove any invalid filename characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	domain = reg.ReplaceAllString(domain, "_")
	
	return domain
}

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
		// Add User-Agent to avoid bot detection
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"),
		// Disable automation flags
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	}

	// Add proxy configuration if enabled
	// CRITICAL: Only use proxy if it's explicitly enabled, tested, and has valid configuration
	if t.proxyConfig != nil && 
	   t.proxyConfig.Enabled && 
	   t.proxyConfig.Tested && 
	   t.proxyConfig.Host != "" && 
	   t.proxyConfig.Port > 0 {
		proxyURL := fmt.Sprintf("%s://%s:%d", 
			t.proxyConfig.Protocol, t.proxyConfig.Host, t.proxyConfig.Port)
		opts = append(opts, chromedp.ProxyServer(proxyURL))
		
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] Using proxy: %s", proxyURL))
		}
	} else if t.logger != nil {
		t.logger("[WEB-SEARCH-GOOGLE] Using direct connection (no proxy)")
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
	// Use proper URL encoding for query parameters
	searchURL := fmt.Sprintf("https://%s/search?q=%s&num=%d", 
		baseURL, url.QueryEscape(query), maxResults)

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] Navigating to: %s", searchURL))
	}

	var htmlContent string
	var pageTitle string
	
	// Improved waiting strategy: wait for body instead of specific element
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(searchURL),
		// Wait for body element (always exists)
		chromedp.WaitReady("body", chromedp.ByQuery),
		// Extra wait for JavaScript execution
		chromedp.Sleep(2*time.Second),
		// Get page title for debugging
		chromedp.Title(&pageTitle),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] Navigation failed: %v", err))
		}
		return nil, fmt.Errorf("failed to load search results: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] Page loaded, title: %s, HTML size: %d bytes", 
			pageTitle, len(htmlContent)))
		
		// Check for captcha or bot detection
		if strings.Contains(strings.ToLower(htmlContent), "captcha") || 
		   strings.Contains(strings.ToLower(htmlContent), "unusual traffic") {
			t.logger("[WEB-SEARCH-GOOGLE] WARNING: Captcha or bot detection page detected")
		}
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Try multiple selectors (Google frequently changes structure)
	results := []SearchResult{}
	
	// Selector priority list
	selectors := []string{
		"div.g",           // Traditional selector
		"div.MjjYud",      // Newer selector
		"div[data-sokoban-container]", // Another possible selector
	}
	
	var foundSelector string
	for _, selector := range selectors {
		found := doc.Find(selector)
		if found.Length() > 0 {
			foundSelector = selector
			if t.logger != nil {
				t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] Using selector: %s (found %d elements)", 
					selector, found.Length()))
			}
			
			found.Each(func(i int, s *goquery.Selection) {
				if len(results) >= maxResults {
					return
				}

				// Extract title and URL
				titleElem := s.Find("h3")
				linkElem := s.Find("a")
				
				title := strings.TrimSpace(titleElem.Text())
				resultURL, _ := linkElem.Attr("href")
				
				// Extract snippet - try multiple selectors
				var snippet string
				snippetSelectors := []string{
					"div[data-sncf]",
					"div.VwiC3b",
					"div[style*='-webkit-line-clamp']",
				}
				for _, snippetSel := range snippetSelectors {
					snippetElem := s.Find(snippetSel)
					if snippetElem.Length() > 0 {
						snippet = strings.TrimSpace(snippetElem.First().Text())
						if snippet != "" {
							break
						}
					}
				}

				// Extract display link
				displayLink := ""
				citeLinkElem := s.Find("cite")
				if citeLinkElem.Length() > 0 {
					displayLink = strings.TrimSpace(citeLinkElem.Text())
				}

				// Only add if we have at least title and URL
				if title != "" && resultURL != "" {
					results = append(results, SearchResult{
						Title:       title,
						URL:         resultURL,
						Snippet:     snippet,
						DisplayLink: displayLink,
					})
				}
			})
			break
		}
	}
	
	if foundSelector == "" && t.logger != nil {
		t.logger("[WEB-SEARCH-GOOGLE] WARNING: No matching selector found")
		// Save HTML for debugging
		domainName := extractDomainName(baseURL)
		debugFile := fmt.Sprintf("debug_%s_%d.html", domainName, time.Now().Unix())
		if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err == nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] HTML saved to: %s", debugFile))
		}
	}

	if len(results) == 0 {
		// Save HTML for debugging when no results found
		if t.logger != nil {
			domainName := extractDomainName(baseURL)
			debugFile := fmt.Sprintf("debug_%s_%d.html", domainName, time.Now().Unix())
			if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err == nil {
				t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] No results found. HTML saved to: %s", debugFile))
			}
		}
		return nil, fmt.Errorf("no search results found (page loaded but no results extracted)")
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-GOOGLE] Successfully extracted %d results", len(results)))
	}

	return results, nil
}

// searchBing performs a Bing search using chromedp
func (t *WebSearchTool) searchBing(ctx context.Context, query string, maxResults int, baseURL string) ([]SearchResult, error) {
	// Create chromedp context with proxy support
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		// Add User-Agent to avoid bot detection
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	}

	// Add proxy configuration if enabled
	// CRITICAL: Only use proxy if it's explicitly enabled, tested, and has valid configuration
	if t.proxyConfig != nil && 
	   t.proxyConfig.Enabled && 
	   t.proxyConfig.Tested && 
	   t.proxyConfig.Host != "" && 
	   t.proxyConfig.Port > 0 {
		proxyURL := fmt.Sprintf("%s://%s:%d", 
			t.proxyConfig.Protocol, t.proxyConfig.Host, t.proxyConfig.Port)
		opts = append(opts, chromedp.ProxyServer(proxyURL))
		
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-BING] Using proxy: %s", proxyURL))
		}
	} else if t.logger != nil {
		t.logger("[WEB-SEARCH-BING] Using direct connection (no proxy)")
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctxWithCancel, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctxWithCancel, 90*time.Second)
	defer timeoutCancel()

	searchURL := fmt.Sprintf("https://%s/search?q=%s&count=%d", 
		baseURL, url.QueryEscape(query), maxResults)

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-BING] Navigating to: %s", searchURL))
	}

	var htmlContent string
	var pageTitle string
	
	// Improved waiting strategy
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Title(&pageTitle),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-BING] Navigation failed: %v", err))
		}
		return nil, fmt.Errorf("failed to load search results: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-BING] Page loaded, title: %s, HTML size: %d bytes", 
			pageTitle, len(htmlContent)))
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	results := []SearchResult{}
	
	// Try multiple selectors
	selectors := []string{
		"li.b_algo",
		"li.b_algo_group",
		"div.b_algo",
	}
	
	var foundSelector string
	for _, selector := range selectors {
		found := doc.Find(selector)
		if found.Length() > 0 {
			foundSelector = selector
			if t.logger != nil {
				t.logger(fmt.Sprintf("[WEB-SEARCH-BING] Using selector: %s (found %d elements)", 
					selector, found.Length()))
			}
			
			found.Each(func(i int, s *goquery.Selection) {
				if len(results) >= maxResults {
					return
				}

				titleElem := s.Find("h2 a, h3 a, a")
				title := strings.TrimSpace(titleElem.First().Text())
				resultURL, _ := titleElem.First().Attr("href")

				snippetElem := s.Find("p, .b_caption p, .b_caption, div.b_caption")
				snippet := strings.TrimSpace(snippetElem.First().Text())

				if title != "" && resultURL != "" {
					results = append(results, SearchResult{
						Title:   title,
						URL:     resultURL,
						Snippet: snippet,
					})
				}
			})
			break
		}
	}
	
	if foundSelector == "" && t.logger != nil {
		t.logger("[WEB-SEARCH-BING] WARNING: No matching selector found")
		domainName := extractDomainName(baseURL)
		debugFile := fmt.Sprintf("debug_%s_%d.html", domainName, time.Now().Unix())
		if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err == nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-BING] HTML saved to: %s", debugFile))
		}
	}

	if len(results) == 0 {
		// Save HTML for debugging when no results found
		if t.logger != nil {
			domainName := extractDomainName(baseURL)
			debugFile := fmt.Sprintf("debug_%s_%d.html", domainName, time.Now().Unix())
			if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err == nil {
				t.logger(fmt.Sprintf("[WEB-SEARCH-BING] No results found. HTML saved to: %s", debugFile))
			}
		}
		return nil, fmt.Errorf("no search results found (page loaded but no results extracted)")
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-BING] Successfully extracted %d results", len(results)))
	}

	return results, nil
}

// searchBaidu performs a Baidu search using chromedp
func (t *WebSearchTool) searchBaidu(ctx context.Context, query string, maxResults int, baseURL string) ([]SearchResult, error) {
	// Create chromedp context with proxy support
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		// Add User-Agent for Baidu
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// Baidu may need Chinese language support
		chromedp.Flag("accept-language", "zh-CN,zh;q=0.9,en;q=0.8"),
	}

	// Add proxy configuration if enabled
	// CRITICAL: Only use proxy if it's explicitly enabled, tested, and has valid configuration
	if t.proxyConfig != nil && 
	   t.proxyConfig.Enabled && 
	   t.proxyConfig.Tested && 
	   t.proxyConfig.Host != "" && 
	   t.proxyConfig.Port > 0 {
		proxyURL := fmt.Sprintf("%s://%s:%d", 
			t.proxyConfig.Protocol, t.proxyConfig.Host, t.proxyConfig.Port)
		opts = append(opts, chromedp.ProxyServer(proxyURL))
		
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] Using proxy: %s", proxyURL))
		}
	} else if t.logger != nil {
		t.logger("[WEB-SEARCH-BAIDU] Using direct connection (no proxy)")
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctxWithCancel, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctxWithCancel, 90*time.Second)
	defer timeoutCancel()

	searchURL := fmt.Sprintf("https://%s/s?wd=%s&rn=%d", 
		baseURL, url.QueryEscape(query), maxResults)

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] Navigating to: %s", searchURL))
	}

	var htmlContent string
	var pageTitle string
	
	// Improved waiting strategy
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Title(&pageTitle),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] Navigation failed: %v", err))
		}
		return nil, fmt.Errorf("failed to load search results: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] Page loaded, title: %s, HTML size: %d bytes", 
			pageTitle, len(htmlContent)))
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	results := []SearchResult{}
	
	// Try multiple selectors for Baidu
	selectors := []string{
		".result",
		".c-container",
		"div[tpl]",
	}
	
	var foundSelector string
	for _, selector := range selectors {
		found := doc.Find(selector)
		if found.Length() > 0 {
			foundSelector = selector
			if t.logger != nil {
				t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] Using selector: %s (found %d elements)", 
					selector, found.Length()))
			}
			
			found.Each(func(i int, s *goquery.Selection) {
				if len(results) >= maxResults {
					return
				}

				titleElem := s.Find("h3 a, a")
				title := strings.TrimSpace(titleElem.First().Text())
				resultURL, _ := titleElem.First().Attr("href")

				snippetElem := s.Find(".c-abstract, .c-span9, span")
				snippet := strings.TrimSpace(snippetElem.First().Text())

				if title != "" && resultURL != "" {
					results = append(results, SearchResult{
						Title:   title,
						URL:     resultURL,
						Snippet: snippet,
					})
				}
			})
			break
		}
	}
	
	if foundSelector == "" && t.logger != nil {
		t.logger("[WEB-SEARCH-BAIDU] WARNING: No matching selector found")
		domainName := extractDomainName(baseURL)
		debugFile := fmt.Sprintf("debug_%s_%d.html", domainName, time.Now().Unix())
		if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err == nil {
			t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] HTML saved to: %s", debugFile))
		}
	}

	if len(results) == 0 {
		// Save HTML for debugging when no results found
		if t.logger != nil {
			domainName := extractDomainName(baseURL)
			debugFile := fmt.Sprintf("debug_%s_%d.html", domainName, time.Now().Unix())
			if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err == nil {
				t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] No results found. HTML saved to: %s", debugFile))
			}
		}
		return nil, fmt.Errorf("no search results found (page loaded but no results extracted)")
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-SEARCH-BAIDU] Successfully extracted %d results", len(results)))
	}

	return results, nil
}
