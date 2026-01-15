package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportSessionHTML exports the session trace as an HTML file
func (a *App) ExportSessionHTML(threadID string) error {
	cfg, _ := a.GetConfig()
	tracePath := filepath.Join(cfg.DataCacheDir, "sessions", threadID, "trace.json")
	
	// Check if trace exists, if not, try history
	var messages []struct {
		Role    string          `json:"role"`
		Content string          `json:"content"`
		ToolCalls []interface{} `json:"tool_calls,omitempty"`
	}

	data, err := os.ReadFile(tracePath)
	if err != nil {
		// Fallback to history
		threads, _ := a.chatService.LoadThreads()
		for _, t := range threads {
			if t.ID == threadID {
				// Convert ChatMessage to struct compatible with trace
				for _, m := range t.Messages {
					messages = append(messages, struct {
						Role    string          `json:"role"`
						Content string          `json:"content"`
						ToolCalls []interface{} `json:"tool_calls,omitempty"`
					}{
						Role:    m.Role,
						Content: m.Content,
					})
				}
				break
			}
		}
	} else {
		json.Unmarshal(data, &messages)
	}

	if len(messages) == 0 {
		return fmt.Errorf("no session data found to export")
	}

	// Generate HTML
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Analysis Session Export</title>
<style>
body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; }
.message { margin-bottom: 20px; padding: 15px; border-radius: 8px; }
.user { background-color: #f0f9ff; border-left: 4px solid #0284c7; }
.assistant { background-color: #f8fafc; border-left: 4px solid #64748b; }
.tool { background-color: #fefce8; border-left: 4px solid #ca8a04; font-family: monospace; white-space: pre-wrap; font-size: 0.9em; }
.role { font-weight: bold; margin-bottom: 5px; color: #334155; }
img { max-width: 100%; height: auto; border: 1px solid #e2e8f0; border-radius: 4px; margin-top: 10px; }
pre { background: #1e293b; color: #e2e8f0; padding: 10px; border-radius: 4px; overflow-x: auto; }
</style>
</head>
<body>
<h1>Analysis Session Export</h1>
`)

	for _, msg := range messages {
		// Convert Markdown images to HTML tags
		content := msg.Content
		reImage := regexp.MustCompile(`!\[.*?]\((data:image\/.*?;base64,.*?)\)`)
		content = reImage.ReplaceAllString(content, `<img src="$1" />`)
		
		// Simple code block formatting
		content = strings.ReplaceAll(content, "\n", "<br>")
		
		divClass := "message " + strings.ToLower(msg.Role)
		if msg.Role == "system" { continue }

		html.WriteString(fmt.Sprintf(`<div class="%s">
<div class="role">%s</div>
<div class="content">%s</div>
</div>`, divClass, strings.ToUpper(msg.Role), content))
	}

	html.WriteString(`</body></html>`)

	// Save File Dialog
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Export Analysis to HTML",
		DefaultFilename: fmt.Sprintf("analysis_%s.html", threadID),
		Filters: []runtime.FileFilter{{DisplayName: "HTML Files", Pattern: "*.html"}},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}

	return os.WriteFile(savePath, []byte(html.String()), 0644)
}