package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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
		reImage := regexp.MustCompile(`!\[.*?]\]\((data:image\/.*?;base64,.*?)\)`) // Escaped for Go string literal
		content = reImage.ReplaceAllString(content, `<img src="$1" />`)
		
		// Simple code block formatting
		content = strings.ReplaceAll(content, "\n", "<br>")
		
		divClass := "message " + strings.ToLower(msg.Role)
		if msg.Role == "system" { continue } // Skip system prompt in export? Or keep it? Usually skip.

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

// AssetizeSession saves the session as a replayable asset
func (a *App) AssetizeSession(threadID string) error {
	// Replayable asset contains the sequence of User Inputs.
	// We might also want to save the Context (DataSource ID).
	
	threads, _ := a.chatService.LoadThreads()
	var targetThread *ChatThread
	for _, t := range threads {
		if t.ID == threadID {
			targetThread = &t
			break
		}
	}
	if targetThread == nil {
		return fmt.Errorf("thread not found")
	}

	type ReplayStep struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type ReplayAsset struct {
		DataSourceID string       `json:"data_source_id"`
		Steps        []ReplayStep `json:"steps"`
		CreatedAt    time.Time    `json:"created_at"`
	}

	asset := ReplayAsset{
		DataSourceID: targetThread.DataSourceID,
		CreatedAt:    time.Now(),
	}

	for _, msg := range targetThread.Messages {
		if msg.Role == "user" {
			asset.Steps = append(asset.Steps, ReplayStep{
				Role:    "user",
				Content: msg.Content,
			})
		}
	}

	data, err := json.MarshalIndent(asset, "", "  ")
	if err != nil {
		return err
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Save Analysis Asset",
		DefaultFilename: fmt.Sprintf("analysis_asset_%s.rbi", threadID),
		Filters: []runtime.FileFilter{{DisplayName: "RapidBI Replay Asset", Pattern: "*.rbi"}},
	})

	if err != nil || savePath == "" {
		return nil
	}

	return os.WriteFile(savePath, data, 0644)
}

// ReplayAnalysis loads an .rbi file and re-runs the analysis
func (a *App) ReplayAnalysis() error {
	loadPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Analysis Asset",
		Filters: []runtime.FileFilter{{DisplayName: "RapidBI Replay Asset", Pattern: "*.rbi"}},
	})

	if err != nil {
		return fmt.Errorf("failed to open file dialog: %v", err)
	}
	
	if loadPath == "" {
		// User cancelled the dialog, this is not an error
		return nil
	}

	data, err := os.ReadFile(loadPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	type ReplayAsset struct {
		DataSourceID string `json:"data_source_id"`
		Steps        []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"steps"`
	}

	var asset ReplayAsset
	if err := json.Unmarshal(data, &asset); err != nil {
		return fmt.Errorf("invalid asset file format: %v", err)
	}

	// Validate data source exists
	if asset.DataSourceID == "" {
		return fmt.Errorf("asset file missing data source ID")
	}

	// Create new thread
	newThread, err := a.CreateChatThread(asset.DataSourceID, "Replay: "+filepath.Base(loadPath))
	if err != nil {
		return fmt.Errorf("failed to create chat thread: %v", err)
	}

	// Show start message
	a.ShowMessage("info", "Replay Started", "Analysis replay has started. Please check the chat window.")

	// Trigger Replay in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.ShowMessage("error", "Replay Error", fmt.Sprintf("Analysis replay failed: %v", r))
			}
		}()

		// Notify frontend to switch to this thread
		runtime.EventsEmit(a.ctx, "start-new-chat", map[string]string{
			"dataSourceId": asset.DataSourceID,
			"sessionName":  newThread.Title, 
		})
		
		for _, step := range asset.Steps {
			if step.Role == "user" {
				// Add user message locally
				usrMsg := ChatMessage{
					ID: strconv.FormatInt(time.Now().UnixNano(), 10),
					Role: "user",
					Content: step.Content,
					Timestamp: time.Now().Unix(),
				}
				a.chatService.AddMessage(newThread.ID, usrMsg)
				runtime.EventsEmit(a.ctx, "thread-updated", newThread.ID)

				// Generate response
				resp, err := a.SendMessage(newThread.ID, step.Content, "")
				if err != nil {
					a.ShowMessage("warning", "Replay Warning", fmt.Sprintf("Failed to generate response for step: %v", err))
					continue
				}
				
				// Add assistant message
				asstMsg := ChatMessage{
					ID: strconv.FormatInt(time.Now().UnixNano(), 10),
					Role: "assistant",
					Content: resp,
					Timestamp: time.Now().Unix(),
				}
				a.chatService.AddMessage(newThread.ID, asstMsg)
				runtime.EventsEmit(a.ctx, "thread-updated", newThread.ID)
			}
		}
		
		a.ShowMessage("info", "Replay Complete", "Analysis replay has finished successfully.")
	}()

	return nil
}