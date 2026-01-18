package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportMessageToPDF exports a single message content to PDF using chromedp (headless Chrome)
// This provides perfect Chinese support and preserves all HTML/CSS styling
func (a *App) ExportMessageToPDF(content string, messageID string) error {
	// Generate HTML
	html := generateMessageHTML(content, messageID)
	
	// Create temp HTML file
	tmpDir := os.TempDir()
	htmlPath := filepath.Join(tmpDir, "temp_export.html")
	err := os.WriteFile(htmlPath, []byte(html), 0644)
	if err != nil {
		return fmt.Errorf("创建临时HTML文件失败: %v", err)
	}
	defer os.Remove(htmlPath)
	
	// Save dialog
	timestamp := time.Now().Format("20060102_150405")
	shortID := messageID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	defaultFilename := fmt.Sprintf("analysis_%s.pdf", shortID)
	if messageID == "" {
		defaultFilename = fmt.Sprintf("analysis_%s.pdf", timestamp)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出为PDF",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF文件", Pattern: "*.pdf"},
		},
	})

	if err != nil || savePath == "" {
		return nil // User cancelled
	}
	
	// Use chromedp to render HTML to PDF
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	
	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var pdfBuf []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate("file://"+htmlPath),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().Do(ctx)
			return err
		}),
	)
	
	if err != nil {
		return fmt.Errorf("PDF生成失败: %v\n提示：需要安装Chrome浏览器", err)
	}
	
	// Write PDF file
	return os.WriteFile(savePath, pdfBuf, 0644)
}

// exportAsHTML exports content as HTML (backup fallback if chromedp fails)
func (a *App) exportAsHTML(content string, messageID string) error {
	html := generateMessageHTML(content, messageID)

	timestamp := time.Now().Format("20060102_150405")
	shortID := messageID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	defaultFilename := fmt.Sprintf("analysis_%s.html", shortID)
	if messageID == "" {
		defaultFilename = fmt.Sprintf("analysis_%s.html", timestamp)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出为HTML (浏览器中Ctrl+P打印为PDF，完美支持中文)",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "HTML文件 (可打印为PDF)", Pattern: "*.html"},
		},
	})

	if err != nil || savePath == "" {
		return nil
	}

	return os.WriteFile(savePath, []byte(html), 0644)
}
