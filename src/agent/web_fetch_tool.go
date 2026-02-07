package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"vantagedata/config"
)

// WebFetchTool provides web page fetching and parsing capabilities using HTTP client
type WebFetchTool struct {
	logger      func(string)
	proxyConfig *config.ProxyConfig
	httpClient  *http.Client
}

// WebFetchInput represents the input for web fetch
type WebFetchInput struct {
	URL      string `json:"url" jsonschema:"description=URL of the web page to fetch"`
	Selector string `json:"selector,omitempty" jsonschema:"description=CSS selector to extract specific content (optional)"`
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

// NewWebFetchTool creates a new web fetch tool using HTTP client
func NewWebFetchTool(logger func(string), proxyConfig *config.ProxyConfig) *WebFetchTool {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Configure proxy if enabled
	if proxyConfig != nil && proxyConfig.Enabled && proxyConfig.Tested &&
		proxyConfig.Host != "" && proxyConfig.Port > 0 {
		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d",
			proxyConfig.Protocol, proxyConfig.Host, proxyConfig.Port))
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			if logger != nil {
				logger(fmt.Sprintf("[WEB-FETCH] Using proxy: %s", proxyURL.String()))
			}
		}
	}

	return &WebFetchTool{
		logger:      logger,
		proxyConfig: proxyConfig,
		httpClient:  client,
	}
}

// Info returns tool information for the LLM
func (t *WebFetchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	description := `Fetch and parse web page content into structured data.

**IMPORTANT: Use web_search FIRST to find URLs, then use this tool to read specific pages.**

Use this tool to:
- Read full content from URLs obtained via web_search results
- Extract structured data (headings, tables, lists) from specific pages
- Parse product pages, pricing tables, or feature comparisons

The tool returns structured content including:
- Title and description
- Main text content (up to 10KB)
- Headings hierarchy
- Tables (great for pricing, specs)
- Links and images
- Metadata

**Limitations:**
- Static HTML only — JavaScript-rendered content may not be available
- Max content: 10KB main text, 50 links, 20 images, 10 tables
- Requires full URL starting with http:// or https://

**Do NOT use this tool to:**
- ❌ Guess URLs without searching first
- ❌ Fetch internal database data (use execute_sql instead)
- ❌ Download files (use export_data instead)`

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
	content, err := t.fetchPage(ctx, input.URL, input.Selector)
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

// fetchPage fetches and parses a web page using HTTP client
func (t *WebFetchTool) fetchPage(ctx context.Context, pageURL, selector string) (*WebPageContent, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN,zh;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	// Fetch the page
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// If selector is provided, narrow down to that element
	if selector != "" {
		selection := doc.Find(selector)
		if selection.Length() == 0 {
			return nil, fmt.Errorf("selector '%s' not found in page", selector)
		}
		// Create a new document from the selection
		html, err := selection.Html()
		if err != nil {
			return nil, fmt.Errorf("failed to extract selection HTML: %v", err)
		}
		doc, err = goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, fmt.Errorf("failed to parse selection: %v", err)
		}
	}

	// Extract structured content
	content := &WebPageContent{
		URL:      pageURL,
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
			// Convert relative URLs to absolute
			if strings.HasPrefix(href, "/") {
				parsedURL, err := url.Parse(pageURL)
				if err == nil {
					href = fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, href)
				}
			}
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
			// Convert relative URLs to absolute
			if strings.HasPrefix(src, "/") {
				parsedURL, err := url.Parse(pageURL)
				if err == nil {
					src = fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, src)
				}
			}
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
