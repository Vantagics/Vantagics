//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"rapidbi/agent"
	"rapidbi/config"
)

// loadConfig 加载配置文件
func loadConfig() (config.Config, error) {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to get home directory: %v", err)
	}

	// 配置文件路径
	configPath := filepath.Join(homeDir, "rapidbi", "config.json")

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 如果配置文件不存在，返回默认配置
		if os.IsNotExist(err) {
			return getDefaultConfig(), nil
		}
		return config.Config{}, fmt.Errorf("failed to read config: %v", err)
	}

	// 解析配置
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, fmt.Errorf("failed to parse config: %v", err)
	}

	return cfg, nil
}

// getDefaultConfig 返回默认配置
func getDefaultConfig() config.Config {
	return config.Config{
		SearchEngines: []config.SearchEngine{
			{
				ID:      "google",
				Name:    "Google",
				URL:     "www.google.com",
				Enabled: true,
			},
		},
		ActiveSearchEngine: "google",
	}
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║        Web搜索工具高级测试程序                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		fmt.Println("   使用默认配置继续...")
		cfg = getDefaultConfig()
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		fmt.Println("请选择测试选项:")
		fmt.Println("  1. 快速测试 (使用默认查询)")
		fmt.Println("  2. 自定义查询测试")
		fmt.Println("  3. 测试不同搜索引擎")
		fmt.Println("  4. 测试中文查询")
		fmt.Println("  5. 压力测试 (多次查询)")
		fmt.Println("  6. 测试Web抓取")
		fmt.Println("  0. 退出")
		fmt.Println()
		fmt.Print("请输入选项 (0-6): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			quickTest(cfg)
		case "2":
			customQueryTest(cfg, reader)
		case "3":
			testDifferentEngines(cfg)
		case "4":
			testChineseQuery(cfg)
		case "5":
			stressTest(cfg, reader)
		case "6":
			testWebFetch(cfg, reader)
		case "0":
			fmt.Println("再见！")
			return
		default:
			fmt.Println("❌ 无效选项，请重新选择")
		}
	}
}

// quickTest 快速测试
func quickTest(cfg config.Config) {
	fmt.Println()
	fmt.Println("=== 快速测试 ===")
	fmt.Println()

	query := "OpenAI GPT-4"
	maxResults := 5

	result, duration, err := performSearch(cfg, query, maxResults)
	if err != nil {
		fmt.Printf("❌ 搜索失败: %v\n", err)
		return
	}

	displayResults(result, duration)
}

// customQueryTest 自定义查询测试
func customQueryTest(cfg config.Config, reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("=== 自定义查询测试 ===")
	fmt.Println()

	fmt.Print("请输入搜索查询: ")
	query, _ := reader.ReadString('\n')
	query = strings.TrimSpace(query)

	if query == "" {
		fmt.Println("❌ 查询不能为空")
		return
	}

	fmt.Print("请输入最大结果数 (1-10, 默认5): ")
	maxResultsStr, _ := reader.ReadString('\n')
	maxResultsStr = strings.TrimSpace(maxResultsStr)

	maxResults := 5
	if maxResultsStr != "" {
		if n, err := strconv.Atoi(maxResultsStr); err == nil && n >= 1 && n <= 10 {
			maxResults = n
		}
	}

	result, duration, err := performSearch(cfg, query, maxResults)
	if err != nil {
		fmt.Printf("❌ 搜索失败: %v\n", err)
		return
	}

	displayResults(result, duration)
}

// testDifferentEngines 测试不同搜索引擎
func testDifferentEngines(cfg config.Config) {
	fmt.Println()
	fmt.Println("=== 测试不同搜索引擎 ===")
	fmt.Println()

	engines := []struct {
		name string
		url  string
	}{
		{"Google", "www.google.com"},
		{"Bing", "www.bing.com"},
		{"Baidu", "www.baidu.com"},
	}

	query := "artificial intelligence"

	for _, engine := range engines {
		fmt.Printf("测试 %s...\n", engine.name)

		// 临时修改配置
		tempEngine := &config.SearchEngine{
			Name:    engine.name,
			URL:     engine.url,
			Enabled: true,
		}

		webSearchTool := agent.NewWebSearchTool(nil, tempEngine, cfg.ProxyConfig)

		searchInput := map[string]interface{}{
			"query":       query,
			"max_results": 3,
		}
		searchInputJSON, _ := json.Marshal(searchInput)

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		startTime := time.Now()
		result, err := webSearchTool.InvokableRun(ctx, string(searchInputJSON))
		duration := time.Since(startTime)
		cancel()

		if err != nil {
			fmt.Printf("  ❌ 失败 (%.2fs): %v\n", duration.Seconds(), err)
		} else {
			var results []map[string]interface{}
			json.Unmarshal([]byte(result), &results)
			fmt.Printf("  ✅ 成功 (%.2fs): 找到 %d 个结果\n", duration.Seconds(), len(results))
		}
		fmt.Println()
	}
}

// testChineseQuery 测试中文查询
func testChineseQuery(cfg config.Config) {
	fmt.Println()
	fmt.Println("=== 测试中文查询 ===")
	fmt.Println()

	queries := []string{
		"人工智能最新进展",
		"中国科技公司",
		"深度学习应用",
	}

	for i, query := range queries {
		fmt.Printf("[%d/%d] 查询: %s\n", i+1, len(queries), query)

		result, duration, err := performSearch(cfg, query, 3)
		if err != nil {
			fmt.Printf("  ❌ 失败: %v\n", err)
		} else {
			var results []map[string]interface{}
			json.Unmarshal([]byte(result), &results)
			fmt.Printf("  ✅ 成功 (%.2fs): 找到 %d 个结果\n", duration.Seconds(), len(results))

			if len(results) > 0 {
				title, _ := results[0]["title"].(string)
				fmt.Printf("  第一个结果: %s\n", truncate(title, 60))
			}
		}
		fmt.Println()
	}
}

// stressTest 压力测试
func stressTest(cfg config.Config, reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("=== 压力测试 ===")
	fmt.Println()

	fmt.Print("请输入测试次数 (1-10): ")
	countStr, _ := reader.ReadString('\n')
	countStr = strings.TrimSpace(countStr)

	count := 3
	if n, err := strconv.Atoi(countStr); err == nil && n >= 1 && n <= 10 {
		count = n
	}

	queries := []string{
		"machine learning",
		"data science",
		"cloud computing",
		"blockchain technology",
		"quantum computing",
	}

	successCount := 0
	totalDuration := time.Duration(0)

	for i := 0; i < count; i++ {
		query := queries[i%len(queries)]
		fmt.Printf("[%d/%d] 查询: %s\n", i+1, count, query)

		result, duration, err := performSearch(cfg, query, 3)
		totalDuration += duration

		if err != nil {
			fmt.Printf("  ❌ 失败 (%.2fs): %v\n", duration.Seconds(), err)
		} else {
			successCount++
			var results []map[string]interface{}
			json.Unmarshal([]byte(result), &results)
			fmt.Printf("  ✅ 成功 (%.2fs): %d 个结果\n", duration.Seconds(), len(results))
		}

		// 避免请求过快
		if i < count-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println()
	fmt.Println("=== 压力测试结果 ===")
	fmt.Printf("总测试次数: %d\n", count)
	fmt.Printf("成功次数: %d\n", successCount)
	fmt.Printf("失败次数: %d\n", count-successCount)
	fmt.Printf("成功率: %.1f%%\n", float64(successCount)/float64(count)*100)
	fmt.Printf("平均耗时: %.2fs\n", totalDuration.Seconds()/float64(count))
}

// testWebFetch 测试Web抓取
func testWebFetch(cfg config.Config, reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("=== 测试Web抓取 ===")
	fmt.Println()

	fmt.Print("请输入URL: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	if url == "" {
		fmt.Println("❌ URL不能为空")
		return
	}

	fmt.Println()
	fmt.Println("选择抓取模式:")
	fmt.Println("  1. truncated (前8KB)")
	fmt.Println("  2. full (完整内容)")
	fmt.Println("  3. selective (搜索特定内容)")
	fmt.Print("请选择 (1-3, 默认1): ")

	modeInput, _ := reader.ReadString('\n')
	modeInput = strings.TrimSpace(modeInput)

	mode := "truncated"
	var searchPhrase string

	switch modeInput {
	case "2":
		mode = "full"
	case "3":
		mode = "selective"
		fmt.Print("请输入搜索关键词: ")
		searchPhrase, _ = reader.ReadString('\n')
		searchPhrase = strings.TrimSpace(searchPhrase)
	}

	webFetchTool := agent.NewWebFetchTool(
		func(msg string) { fmt.Printf("[LOG] %s\n", msg) },
		cfg.ProxyConfig,
	)

	fetchInput := map[string]interface{}{
		"url":  url,
		"mode": mode,
	}
	if searchPhrase != "" {
		fetchInput["search_phrase"] = searchPhrase
	}
	fetchInputJSON, _ := json.Marshal(fetchInput)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println("开始抓取...")
	startTime := time.Now()
	result, err := webFetchTool.InvokableRun(ctx, string(fetchInputJSON))
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ 抓取失败 (%.2fs): %v\n", duration.Seconds(), err)
		return
	}

	fmt.Printf("✅ 抓取成功 (%.2fs)\n", duration.Seconds())
	fmt.Printf("内容长度: %d 字符\n", len(result))
	fmt.Println()
	fmt.Println("内容预览 (前1000字符):")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println(truncate(result, 1000))
	fmt.Println("═══════════════════════════════════════════════════════════")
}

// performSearch 执行搜索
func performSearch(cfg config.Config, query string, maxResults int) (string, time.Duration, error) {
	activeEngine := cfg.GetActiveSearchEngine()
	webSearchTool := agent.NewWebSearchTool(nil, activeEngine, cfg.ProxyConfig)

	searchInput := map[string]interface{}{
		"query":       query,
		"max_results": maxResults,
	}
	searchInputJSON, _ := json.Marshal(searchInput)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	startTime := time.Now()
	result, err := webSearchTool.InvokableRun(ctx, string(searchInputJSON))
	duration := time.Since(startTime)

	return result, duration, err
}

// displayResults 显示搜索结果
func displayResults(resultJSON string, duration time.Duration) {
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(resultJSON), &results); err != nil {
		fmt.Printf("❌ 解析结果失败: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Printf("✅ 搜索成功 (耗时: %.2f秒)\n", duration.Seconds())
	fmt.Printf("📊 找到 %d 个结果:\n", len(results))
	fmt.Println()

	for i, result := range results {
		title, _ := result["title"].(string)
		url, _ := result["url"].(string)
		snippet, _ := result["snippet"].(string)

		fmt.Printf("【结果 #%d】\n", i+1)
		fmt.Printf("标题: %s\n", title)
		fmt.Printf("URL:  %s\n", url)
		fmt.Printf("摘要: %s\n", truncate(snippet, 200))
		fmt.Println()
	}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
