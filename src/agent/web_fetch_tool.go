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

// WebFetchTool provides web page fetching and parsing capabilities
type WebFetchTool struct {
	logger      func(string)
	proxyConfig *config.ProxyConfig
}

// WebFetchInput represents the input for web fetch
type WebFetchInput struct {
	URL      string `json:"url" jsonschema:"description=URL of the web page to fetch"`
	Selector string `json:"selector,omitempty" jsonschema:"description=CSS selector to extract specific content (optional)"`
	WaitFor  string `json:"wait_for,omitempty" jsonschema:"description=CSS selector to wait for before extracting (optional)"`
}

// WebPageContent represents structured web page content
type WebPageContent struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	MainContent string            `json:"main_content"`
	Headings    []string          `json:"headings,omitempty"`
	Links       []Link            `json:"links,omitempty"`
	Images      []Image           `json:"images,omitempty"`
	Tables      []TableData       `json:"tables,omitempty"`
	Lists       []string          `json:"lists,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Link represents a hyperlink
type Link struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

// Image represents an image
type Image struct {
	Alt string `json:"alt"`
	Src string `json:"src"`
}

// TableData represents a table
type TableData struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// NewWebFetchTool creates a new web fetch tool
func NewWebFetchTool(logger func(string), proxyConfig *config.ProxyConfig) *WebFetchTool {
	return &WebFetchTool{
		logger:      logger,
		proxyConfig: proxyConfig,
	}
}

// Info returns tool information for the LLM
func (t *WebFetchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	description := `Fetch and parse web page content into structured data.

Use this tool to:
- Read full content from specific URLs (from search results)
- Extract structured data (headings, tables, lists)
- Parse competitor websites and product pages
- Analyze pricing pages and feature comparisons
- Extract business data from company websites

The tool returns structured content including:
- Title and description
- Main text content
- Headings hierarchy
- Tables (perfect for pricing, features, specs)
- Links and images
- Metadata

Examples:
- Fetch competitor pricing page
- Extract product specifications table
- Read company about page
- Parse market research report`

	return &schema.ToolInfo{
		Name: "web_fetch",
		Desc: description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Desc:     "Full URL of the web page to fetch (must start with http:// or https://)",
				Required: true,
			},
			"selector": {
				Type:     schema.String,
				Desc:     "CSS selector to extract specific content (e.g., 'article', '.pricing-table', '#main-content')",
				Required: false,
			},
			"wait_for": {
				Type:     schema.String,
				Desc:     "CSS selector to wait for before extracting (useful for dynamic content)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the web fetch
func (t *WebFetchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Parse input
	var input WebFetchInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	if input.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	// Validate URL
	if !strings.HasPrefix(input.URL, "http://") && !strings.HasPrefix(input.URL, "https://") {
		return "", fmt.Errorf("url must start with http:// or https://")
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-FETCH] Fetching: %s", input.URL))
	}

	// Fetch and parse the page
	content, err := t.fetchPage(ctx, input.URL, input.Selector, input.WaitFor)
	if err != nil {
		if t.logger != nil {
			t.logger(fmt.Sprintf("[WEB-FETCH] Fetch failed: %v", err))
		}
		return "", fmt.Errorf("fetch failed: %v", err)
	}

	if t.logger != nil {
		t.logger(fmt.Sprintf("[WEB-FETCH] Successfully fetched: %s (content length: %d)", 
			content.Title, len(content.MainContent)))
	}

	// Format result as JSON
	resultJSON, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format result: %v", err)
	}

	return string(resultJSON), nil
}

// fetchPage fetches and parses a web page
func (t *WebFetchTool) fetchPage(ctx context.Context, url, selector, waitFor string) (*WebPageContent, error) {
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
			t.logger(fmt.Sprintf("[WEB-FETCH] Using proxy: %s", proxyURL))
		}
	} else if t.logger != nil {
		t.logger("[WEB-FETCH] Using direct connection (no proxy)")
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create chromedp context
	ctxWithCancel, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout for the entire operation (increased to 60s for slow networks)
	timeoutCtx, timeoutCancel := context.WithTimeout(ctxWithCancel, 60*time.Second)
	defer timeoutCancel()

	// Navigate to URL and wait for content
	var htmlContent string
	tasks := []chromedp.Action{
		chromedp.Navigate(url),
	}

	// Wait for specific element if specified
	if waitFor != "" {
		tasks = append(tasks, chromedp.WaitVisible(waitFor, chromedp.ByQuery))
	} else {
		// Default wait for body
		tasks = append(tasks, chromedp.WaitVisible(`body`, chromedp.ByQuery))
	}

	// Extract HTML
	if selector != "" {
		tasks = append(tasks, chromedp.OuterHTML(selector, &htmlContent, chromedp.ByQuery))
	} else {
		tasks = append(tasks, chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery))
	}

	err := chromedp.Run(timeoutCtx, tasks...)
	if err != nil {
		return nil, fmt.Errorf("failed to load page: %v", err)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Extract structured content
	content := &WebPageContent{
		URL:      url,
		Metadata: make(map[string]string),
	}

	// Extract title
	content.Title = strings.TrimSpace(doc.Find("title").Text())

	// Extract meta description
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if desc, exists := s.Attr("content"); exists {
			content.Description = strings.TrimSpace(desc)
		}
	})

	// Extract main content (try common content selectors)
	mainSelectors := []string{"article", "main", ".content", "#content", ".main-content", "body"}
	for _, sel := range mainSelectors {
		mainElem := doc.Find(sel).First()
		if mainElem.Length() > 0 {
			content.MainContent = strings.TrimSpace(mainElem.Text())
			if len(content.MainContent) > 100 { // Only use if substantial content
				break
			}
		}
	}

	// Fallback to body if no main content found
	if content.MainContent == "" {
		content.MainContent = strings.TrimSpace(doc.Find("body").Text())
	}

	// Limit main content length
	if len(content.MainContent) > 10000 {
		content.MainContent = content.MainContent[:10000] + "... [truncated]"
	}

	// Extract headings
	doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
		heading := strings.TrimSpace(s.Text())
		if heading != "" && len(content.Headings) < 20 {
			content.Headings = append(content.Headings, heading)
		}
	})

	// Extract links
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if len(content.Links) >= 50 { // Limit links
			return
		}
		text := strings.TrimSpace(s.Text())
		href, _ := s.Attr("href")
		if text != "" && href != "" {
			content.Links = append(content.Links, Link{
				Text: text,
				URL:  href,
			})
		}
	})

	// Extract images
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if len(content.Images) >= 20 { // Limit images
			return
		}
		alt, _ := s.Attr("alt")
		src, _ := s.Attr("src")
		if src != "" {
			content.Images = append(content.Images, Image{
				Alt: alt,
				Src: src,
			})
		}
	})

	// Extract tables
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		if len(content.Tables) >= 10 { // Limit tables
			return
		}
		table := t.parseTable(s)
		if len(table.Headers) > 0 || len(table.Rows) > 0 {
			content.Tables = append(content.Tables, table)
		}
	})

	// Extract lists
	doc.Find("ul, ol").Each(func(i int, s *goquery.Selection) {
		if len(content.Lists) >= 20 { // Limit lists
			return
		}
		s.Find("li").Each(func(j int, li *goquery.Selection) {
			item := strings.TrimSpace(li.Text())
			if item != "" {
				content.Lists = append(content.Lists, item)
			}
		})
	})

	// Extract metadata
	doc.Find("meta[property], meta[name]").Each(func(i int, s *goquery.Selection) {
		var key string
		if prop, exists := s.Attr("property"); exists {
			key = prop
		} else if name, exists := s.Attr("name"); exists {
			key = name
		}
		if metaContent, exists := s.Attr("content"); exists && key != "" {
			content.Metadata[key] = metaContent
		}
	})

	return content, nil
}

// parseTable extracts table data
func (t *WebFetchTool) parseTable(table *goquery.Selection) TableData {
	data := TableData{
		Headers: []string{},
		Rows:    [][]string{},
	}

	// Extract headers
	table.Find("thead tr th, thead tr td").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Text())
		data.Headers = append(data.Headers, header)
	})

	// If no thead, try first tr
	if len(data.Headers) == 0 {
		table.Find("tr").First().Find("th, td").Each(func(i int, s *goquery.Selection) {
			header := strings.TrimSpace(s.Text())
			data.Headers = append(data.Headers, header)
		})
	}

	// Extract rows
	table.Find("tbody tr, tr").Each(func(i int, s *goquery.Selection) {
		// Skip if this is the header row
		if i == 0 && len(data.Headers) > 0 && table.Find("thead").Length() == 0 {
			return
		}

		row := []string{}
		s.Find("td, th").Each(func(j int, cell *goquery.Selection) {
			cellText := strings.TrimSpace(cell.Text())
			row = append(row, cellText)
		})

		if len(row) > 0 && len(data.Rows) < 100 { // Limit rows
			data.Rows = append(data.Rows, row)
		}
	})

	return data
}
