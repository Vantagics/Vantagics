package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// convertMarkdownToHTML converts markdown content to HTML
func convertMarkdownToHTML(content string) string {
	// Convert Markdown images to HTML tags first (before line processing)
	reImage := regexp.MustCompile(`!\[.*?\]\((data:image\/.*?;base64,.*?)\)`)
	content = reImage.ReplaceAllString(content, `<img src="$1" alt="Chart" />`)

	// Convert code blocks ```code``` first (to protect content inside)
	reCodeBlock := regexp.MustCompile("```([\\s\\S]*?)```")
	codeBlocks := reCodeBlock.FindAllString(content, -1)
	for i, block := range codeBlocks {
		placeholder := fmt.Sprintf("__CODE_BLOCK_%d__", i)
		content = strings.Replace(content, block, placeholder, 1)
	}

	// Process line by line for headers and lists
	lines := strings.Split(content, "\n")
	var result []string
	inList := false
	listType := "" // "ul" or "ol"

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for headers (must be at start of line)
		if strings.HasPrefix(trimmedLine, "#### ") {
			if inList {
				result = append(result, "</"+listType+">")
				inList = false
			}
			result = append(result, "<h4>"+strings.TrimPrefix(trimmedLine, "#### ")+"</h4>")
			continue
		}
		if strings.HasPrefix(trimmedLine, "### ") {
			if inList {
				result = append(result, "</"+listType+">")
				inList = false
			}
			result = append(result, "<h3>"+strings.TrimPrefix(trimmedLine, "### ")+"</h3>")
			continue
		}
		if strings.HasPrefix(trimmedLine, "## ") {
			if inList {
				result = append(result, "</"+listType+">")
				inList = false
			}
			result = append(result, "<h2>"+strings.TrimPrefix(trimmedLine, "## ")+"</h2>")
			continue
		}
		if strings.HasPrefix(trimmedLine, "# ") {
			if inList {
				result = append(result, "</"+listType+">")
				inList = false
			}
			result = append(result, "<h1>"+strings.TrimPrefix(trimmedLine, "# ")+"</h1>")
			continue
		}

		// Check for horizontal rule
		if trimmedLine == "---" || trimmedLine == "***" || trimmedLine == "___" {
			if inList {
				result = append(result, "</"+listType+">")
				inList = false
			}
			result = append(result, "<hr>")
			continue
		}

		// Check for unordered list items (- or *)
		if strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "* ") {
			if !inList || listType != "ul" {
				if inList {
					result = append(result, "</"+listType+">")
				}
				result = append(result, "<ul>")
				inList = true
				listType = "ul"
			}
			itemContent := strings.TrimPrefix(strings.TrimPrefix(trimmedLine, "- "), "* ")
			result = append(result, "<li>"+itemContent+"</li>")
			continue
		}

		// Check for ordered list items (1. 2. etc)
		reOrderedList := regexp.MustCompile(`^\d+\.\s+(.*)$`)
		if matches := reOrderedList.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			if !inList || listType != "ol" {
				if inList {
					result = append(result, "</"+listType+">")
				}
				result = append(result, "<ol>")
				inList = true
				listType = "ol"
			}
			result = append(result, "<li>"+matches[1]+"</li>")
			continue
		}

		// Close list if we hit a non-list line
		if inList && trimmedLine != "" {
			result = append(result, "</"+listType+">")
			inList = false
		}

		// Empty line
		if trimmedLine == "" {
			if inList {
				result = append(result, "</"+listType+">")
				inList = false
			}
			result = append(result, "")
			continue
		}

		// Regular paragraph
		result = append(result, line)
	}

	// Close any open list
	if inList {
		result = append(result, "</"+listType+">")
	}

	content = strings.Join(result, "\n")

	// Restore code blocks
	for i, block := range codeBlocks {
		placeholder := fmt.Sprintf("__CODE_BLOCK_%d__", i)
		// Extract code content from block
		codeContent := reCodeBlock.ReplaceAllString(block, "$1")
		content = strings.Replace(content, placeholder, "<pre><code>"+codeContent+"</code></pre>", 1)
	}

	// Convert bold text **text**
	reBold := regexp.MustCompile(`\*\*(.*?)\*\*`)
	content = reBold.ReplaceAllString(content, `<strong>$1</strong>`)

	// Convert italic text *text* (but not inside bold)
	reItalic := regexp.MustCompile(`\*([^*]+)\*`)
	content = reItalic.ReplaceAllString(content, `<em>$1</em>`)

	// Convert inline code `code`
	reInlineCode := regexp.MustCompile("`([^`]+)`")
	content = reInlineCode.ReplaceAllString(content, `<code>$1</code>`)

	// Convert remaining line breaks to <br> for non-block elements
	// But preserve structure for headers, lists, etc.
	lines = strings.Split(content, "\n")
	var finalResult []string
	for i, line := range lines {
		// Don't add <br> after block elements
		if strings.HasPrefix(line, "<h") || strings.HasPrefix(line, "</h") ||
			strings.HasPrefix(line, "<ul") || strings.HasPrefix(line, "</ul") ||
			strings.HasPrefix(line, "<ol") || strings.HasPrefix(line, "</ol") ||
			strings.HasPrefix(line, "<li") || strings.HasPrefix(line, "</li") ||
			strings.HasPrefix(line, "<pre") || strings.HasPrefix(line, "</pre") ||
			strings.HasPrefix(line, "<hr") || line == "" {
			finalResult = append(finalResult, line)
		} else if i < len(lines)-1 {
			// Add <br> for regular text lines
			nextLine := ""
			if i+1 < len(lines) {
				nextLine = lines[i+1]
			}
			// Don't add <br> if next line is a block element
			if strings.HasPrefix(nextLine, "<h") || strings.HasPrefix(nextLine, "<ul") ||
				strings.HasPrefix(nextLine, "<ol") || strings.HasPrefix(nextLine, "<pre") ||
				strings.HasPrefix(nextLine, "<hr") || nextLine == "" {
				finalResult = append(finalResult, line)
			} else {
				finalResult = append(finalResult, line+"<br>")
			}
		} else {
			finalResult = append(finalResult, line)
		}
	}

	// Wrap non-block content in paragraphs
	content = strings.Join(finalResult, "\n")
	
	// Clean up empty lines
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")

	return content
}

// generateMessageHTML creates HTML content for a single message
func generateMessageHTML(content string, messageID string) string {
	var html strings.Builder

	// Filter out technical code blocks before processing
	// Remove json:echarts, json:table, json:metrics, json:dashboard code blocks
	filteredContent := content
	filteredContent = regexp.MustCompile("```[ \t]*json:echarts[\\s\\S]*?```").ReplaceAllString(filteredContent, "")
	filteredContent = regexp.MustCompile("```[ \t]*json:table[\\s\\S]*?```").ReplaceAllString(filteredContent, "")
	filteredContent = regexp.MustCompile("```[ \t]*json:metrics[\\s\\S]*?```").ReplaceAllString(filteredContent, "")
	filteredContent = regexp.MustCompile("```[ \t]*json:dashboard[\\s\\S]*?```").ReplaceAllString(filteredContent, "")
	// Remove SQL and Python code blocks
	filteredContent = regexp.MustCompile("```[ \t]*(sql|SQL)[\\s\\S]*?```").ReplaceAllString(filteredContent, "")
	filteredContent = regexp.MustCompile("```[ \t]*(python|Python|py)[\\s\\S]*?```").ReplaceAllString(filteredContent, "")

	// Convert Markdown to HTML
	contentHTML := convertMarkdownToHTML(filteredContent)

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>分析结果</title>
<style>
body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
    max-width: 900px;
    margin: 0 auto;
    padding: 40px 20px;
    line-height: 1.8;
    color: #1e293b;
    background-color: #f8fafc;
}
.container {
    background: white;
    border-radius: 12px;
    padding: 40px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.header {
    margin-bottom: 30px;
    padding-bottom: 20px;
    border-bottom: 2px solid #e2e8f0;
}
h1 {
    color: #0f172a;
    font-size: 28px;
    margin: 0 0 10px 0;
    font-weight: 700;
}
h2 {
    color: #1e293b;
    font-size: 22px;
    margin: 24px 0 12px 0;
    font-weight: 600;
    border-bottom: 1px solid #e2e8f0;
    padding-bottom: 8px;
}
h3 {
    color: #334155;
    font-size: 18px;
    margin: 20px 0 10px 0;
    font-weight: 600;
}
h4 {
    color: #475569;
    font-size: 16px;
    margin: 16px 0 8px 0;
    font-weight: 600;
}
.meta {
    color: #64748b;
    font-size: 14px;
}
.content {
    font-size: 16px;
    color: #334155;
}
.content p {
    margin: 16px 0;
}
ul, ol {
    margin: 16px 0;
    padding-left: 24px;
}
li {
    margin: 8px 0;
}
img {
    max-width: 100%;
    height: auto;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    margin: 20px 0;
    display: block;
}
pre {
    background: #1e293b;
    color: #e2e8f0;
    padding: 16px;
    border-radius: 8px;
    overflow-x: auto;
    font-family: "Consolas", "Monaco", "Courier New", monospace;
    font-size: 14px;
    line-height: 1.5;
    margin: 20px 0;
}
code {
    background: #f1f5f9;
    color: #0f172a;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: "Consolas", "Monaco", "Courier New", monospace;
    font-size: 14px;
}
pre code {
    background: transparent;
    color: inherit;
    padding: 0;
}
strong {
    font-weight: 600;
    color: #0f172a;
}
hr {
    border: none;
    border-top: 1px solid #e2e8f0;
    margin: 24px 0;
}
.footer {
    margin-top: 40px;
    padding-top: 20px;
    border-top: 1px solid #e2e8f0;
    text-align: center;
    color: #94a3b8;
    font-size: 12px;
}
@media print {
    body { background: white; padding: 0; }
    .container { box-shadow: none; }
}
</style>
</head>
<body>
<div class="container">
<div class="header">
<h1>📊 分析结果</h1>
<div class="meta">导出时间: ` + time.Now().Format("2006-01-02 15:04:05") + `</div>
</div>
<div class="content">
` + contentHTML + `
</div>
<div class="footer">
Generated by VantageData
</div>
</div>
</body>
</html>`)

	return html.String()
}

// ExportSessionHTML exports the session trace as an HTML file
func (a *App) ExportSessionHTML(threadID string) error {
	// Load thread from chat service to get complete message data including charts
	threads, err := a.chatService.LoadThreads()
	if err != nil {
		return fmt.Errorf("failed to load threads: %v", err)
	}
	
	var targetThread *ChatThread
	for i := range threads {
		if threads[i].ID == threadID {
			targetThread = &threads[i]
			break
		}
	}
	
	if targetThread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}
	
	if len(targetThread.Messages) == 0 {
		return fmt.Errorf("no messages found in thread")
	}

	// Generate HTML
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Analysis Session Export - ` + targetThread.Title + `</title>
<style>
body { 
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; 
    max-width: 1200px; 
    margin: 0 auto; 
    padding: 20px; 
    line-height: 1.6; 
    background-color: #f8fafc;
}
.header {
    background: linear-gradient(135deg, #3b82f6, #6366f1);
    color: white;
    padding: 30px;
    border-radius: 12px;
    margin-bottom: 30px;
    text-align: center;
}
.header h1 {
    margin: 0 0 10px 0;
    font-size: 2em;
}
.header p {
    margin: 0;
    opacity: 0.9;
}
.message { 
    margin-bottom: 25px; 
    padding: 20px; 
    border-radius: 12px; 
    background: white;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.user { 
    border-left: 4px solid #3b82f6; 
}
.assistant { 
    border-left: 4px solid #10b981; 
}
.role { 
    font-weight: 600; 
    margin-bottom: 10px; 
    color: #1e293b; 
    font-size: 0.9em;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}
.content {
    color: #334155;
    font-size: 15px;
}
.content h1 {
    color: #0f172a;
    font-size: 24px;
    margin: 20px 0 12px 0;
    font-weight: 700;
}
.content h2 {
    color: #1e293b;
    font-size: 20px;
    margin: 18px 0 10px 0;
    font-weight: 600;
    border-bottom: 1px solid #e2e8f0;
    padding-bottom: 6px;
}
.content h3 {
    color: #334155;
    font-size: 17px;
    margin: 16px 0 8px 0;
    font-weight: 600;
}
.content h4 {
    color: #475569;
    font-size: 15px;
    margin: 14px 0 6px 0;
    font-weight: 600;
}
.content ul, .content ol {
    margin: 12px 0;
    padding-left: 24px;
}
.content li {
    margin: 6px 0;
}
.content hr {
    border: none;
    border-top: 1px solid #e2e8f0;
    margin: 20px 0;
}
.chart-container {
    margin: 20px 0;
    padding: 20px;
    background: #f8fafc;
    border-radius: 8px;
    border: 1px solid #e2e8f0;
}
.chart-title {
    font-weight: 600;
    color: #475569;
    margin-bottom: 15px;
    font-size: 0.9em;
}
img { 
    max-width: 100%; 
    height: auto; 
    border: 1px solid #e2e8f0; 
    border-radius: 8px; 
    margin: 10px 0;
    display: block;
}
pre { 
    background: #1e293b; 
    color: #e2e8f0; 
    padding: 15px; 
    border-radius: 8px; 
    overflow-x: auto; 
    font-family: "Consolas", "Monaco", monospace;
    font-size: 13px;
}
code {
    background: #f1f5f9;
    color: #0f172a;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: "Consolas", "Monaco", monospace;
    font-size: 13px;
}
pre code {
    background: transparent;
    color: inherit;
    padding: 0;
}
.table-container {
    overflow-x: auto;
    margin: 15px 0;
}
table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
}
th, td {
    padding: 10px;
    text-align: left;
    border: 1px solid #e2e8f0;
}
th {
    background: #f1f5f9;
    font-weight: 600;
    color: #475569;
}
tr:nth-child(even) {
    background: #f8fafc;
}
.footer {
    margin-top: 40px;
    padding: 20px;
    text-align: center;
    color: #64748b;
    font-size: 0.9em;
    border-top: 1px solid #e2e8f0;
}
@media print {
    body { background: white; }
    .message { box-shadow: none; border: 1px solid #e2e8f0; page-break-inside: avoid; }
    .chart-container { page-break-inside: avoid; }
}
</style>
</head>
<body>
<div class="header">
<h1>📊 ` + targetThread.Title + `</h1>
<p>导出时间: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
</div>
`)

	for _, msg := range targetThread.Messages {
		if msg.Role == "system" {
			continue
		}
		
		// Filter out technical code blocks
		content := msg.Content
		content = regexp.MustCompile("```[ \t]*json:echarts[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*json:table[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*json:metrics[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*json:dashboard[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*(sql|SQL)[\\s\\S]*?```").ReplaceAllString(content, "")
		content = regexp.MustCompile("```[ \t]*(python|Python|py)[\\s\\S]*?```").ReplaceAllString(content, "")
		
		// Convert Markdown to HTML using the shared function
		content = convertMarkdownToHTML(content)
		
		divClass := "message " + strings.ToLower(msg.Role)
		roleLabel := strings.ToUpper(msg.Role)
		if msg.Role == "user" {
			roleLabel = "👤 " + roleLabel
		} else {
			roleLabel = "🤖 " + roleLabel
		}

		html.WriteString(fmt.Sprintf(`<div class="%s">
<div class="role">%s</div>
<div class="content">%s</div>
`, divClass, roleLabel, content))

		// Add chart data if present
		if msg.ChartData != nil && len(msg.ChartData.Charts) > 0 {
			for idx, chart := range msg.ChartData.Charts {
				html.WriteString(`<div class="chart-container">`)
				html.WriteString(fmt.Sprintf(`<div class="chart-title">📊 Chart %d - Type: %s</div>`, idx+1, strings.ToUpper(chart.Type)))
				
				switch chart.Type {
				case "image":
					// Direct image data (base64 or data URL)
					if strings.HasPrefix(chart.Data, "data:image") {
						html.WriteString(fmt.Sprintf(`<img src="%s" alt="Chart Image" />`, chart.Data))
					} else {
						html.WriteString(`<p style="color: #64748b; font-style: italic;">Image data not available</p>`)
					}
					
				case "echarts":
					// For ECharts, we need to render it or show a placeholder
					// Since we can't execute JavaScript in static HTML, we show the config
					html.WriteString(`<div style="padding: 20px; background: #f1f5f9; border-radius: 8px; border: 2px dashed #cbd5e1;">`)
					html.WriteString(`<p style="color: #64748b; text-align: center; margin: 0;">`)
					html.WriteString(`📊 ECharts Interactive Chart<br>`)
					html.WriteString(`<small>This chart requires JavaScript to render. Please view in the original application for full interactivity.</small>`)
					html.WriteString(`</p></div>`)
					// Optionally include the config in a collapsible section
					html.WriteString(`<details style="margin-top: 10px;">`)
					html.WriteString(`<summary style="cursor: pointer; color: #64748b; font-size: 0.9em;">View Chart Configuration</summary>`)
					html.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, chart.Data))
					html.WriteString(`</details>`)
					
				case "table", "csv":
					// Parse and render table data
					var tableData [][]interface{}
					if err := json.Unmarshal([]byte(chart.Data), &tableData); err == nil && len(tableData) > 0 {
						html.WriteString(`<div class="table-container"><table>`)
						
						// Header row
						if len(tableData) > 0 {
							html.WriteString(`<thead><tr>`)
							for _, cell := range tableData[0] {
								html.WriteString(fmt.Sprintf(`<th>%v</th>`, cell))
							}
							html.WriteString(`</tr></thead>`)
						}
						
						// Data rows
						if len(tableData) > 1 {
							html.WriteString(`<tbody>`)
							for _, row := range tableData[1:] {
								html.WriteString(`<tr>`)
								for _, cell := range row {
									html.WriteString(fmt.Sprintf(`<td>%v</td>`, cell))
								}
								html.WriteString(`</tr>`)
							}
							html.WriteString(`</tbody>`)
						}
						
						html.WriteString(`</table></div>`)
					} else {
						html.WriteString(`<p style="color: #64748b; font-style: italic;">Table data not available</p>`)
					}
					
				default:
					html.WriteString(fmt.Sprintf(`<p style="color: #64748b; font-style: italic;">Unsupported chart type: %s</p>`, chart.Type))
				}
				
				html.WriteString(`</div>`)
			}
		}

		html.WriteString(`</div>`)
	}

	html.WriteString(`<div class="footer">
<p>Generated by VantageData - Intelligent Business Intelligence Platform</p>
<p>` + time.Now().Format("2006-01-02 15:04:05") + `</p>
</div>
</body></html>`)

	// Save File Dialog
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Export Analysis to HTML",
		DefaultFilename: fmt.Sprintf("analysis_%s_%s.html", targetThread.Title, time.Now().Format("20060102_150405")),
		Filters: []runtime.FileFilter{{DisplayName: "HTML Files", Pattern: "*.html"}},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	return os.WriteFile(savePath, []byte(html.String()), 0644)
}