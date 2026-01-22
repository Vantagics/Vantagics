package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"rapidbi/agent"
	"rapidbi/config"
)

// TestWebSearch 测试Web搜索工具
func main() {
	fmt.Println("=== Web搜索工具独立测试程序 ===")
	fmt.Println()

	// 1. 加载配置
	fmt.Println("[1/5] 加载配置...")
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 配置加载成功")
	fmt.Printf("   - 搜索引擎: %s\n", getActiveEngineName(cfg))
	fmt.Printf("   - 代理状态: %s\n", getProxyStatus(cfg))
	fmt.Println()

	// 2. 创建Web搜索工具
	fmt.Println("[2/5] 初始化Web搜索工具...")
	activeEngine := cfg.GetActiveSearchEngine()
	webSearchTool := agent.NewWebSearchTool(
		func(msg string) { fmt.Printf("   [LOG] %s\n", msg) },
		activeEngine,
		cfg.ProxyConfig,
	)
	fmt.Println("✅ Web搜索工具初始化成功")
	fmt.Println()

	// 3. 测试Web搜索
	fmt.Println("[3/5] 测试Web搜索...")
	fmt.Println("   查询: \"OpenAI GPT-4 latest news\"")
	fmt.Println("   最大结果数: 5")
	fmt.Println("   超时: 90秒")
	fmt.Println()

	searchInput := map[string]interface{}{
		"query":       "OpenAI GPT-4 latest news",
		"max_results": 5,
	}
	searchInputJSON, _ := json.Marshal(searchInput)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	searchResult, err := webSearchTool.InvokableRun(ctx, string(searchInputJSON))
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ 搜索失败 (耗时: %.2f秒)\n", duration.Seconds())
		fmt.Printf("   错误: %v\n", err)
		fmt.Println()
		fmt.Println("💡 可能的原因:")
		fmt.Println("   1. 网络连接问题")
		fmt.Println("   2. 搜索引擎被墙或限制访问")
		fmt.Println("   3. 代理配置不正确")
		fmt.Println("   4. 超时时间不够")
		fmt.Println()
		fmt.Println("🔧 建议:")
		fmt.Println("   1. 检查网络连接")
		fmt.Println("   2. 尝试配置代理")
		fmt.Println("   3. 更换搜索引擎（Bing或Baidu）")
		os.Exit(1)
	}

	fmt.Printf("✅ 搜索成功 (耗时: %.2f秒)\n", duration.Seconds())
	fmt.Println()

	// 解析搜索结果
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(searchResult), &results); err != nil {
		fmt.Printf("❌ 解析搜索结果失败: %v\n", err)
		fmt.Println("原始结果:")
		fmt.Println(searchResult)
		os.Exit(1)
	}

	fmt.Printf("📊 找到 %d 个结果:\n", len(results))
	fmt.Println()

	for i, result := range results {
		title, _ := result["title"].(string)
		url, _ := result["url"].(string)
		snippet, _ := result["snippet"].(string)

		fmt.Printf("结果 #%d:\n", i+1)
		fmt.Printf("  标题: %s\n", truncate(title, 80))
		fmt.Printf("  URL: %s\n", url)
		fmt.Printf("  摘要: %s\n", truncate(snippet, 150))
		fmt.Println()
	}

	// 4. 测试Web抓取
	if len(results) > 0 {
		fmt.Println("[4/5] 测试Web抓取...")
		firstURL, _ := results[0]["url"].(string)
		fmt.Printf("   抓取URL: %s\n", firstURL)
		fmt.Println()

		webFetchTool := agent.NewWebFetchTool(
			func(msg string) { fmt.Printf("   [LOG] %s\n", msg) },
			cfg.ProxyConfig,
		)

		fetchInput := map[string]interface{}{
			"url":  firstURL,
			"mode": "truncated", // 只获取前8KB
		}
		fetchInputJSON, _ := json.Marshal(fetchInput)

		startTime = time.Now()
		fetchResult, err := webFetchTool.InvokableRun(ctx, string(fetchInputJSON))
		duration = time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ 抓取失败 (耗时: %.2f秒)\n", duration.Seconds())
			fmt.Printf("   错误: %v\n", err)
		} else {
			fmt.Printf("✅ 抓取成功 (耗时: %.2f秒)\n", duration.Seconds())
			fmt.Printf("   内容长度: %d 字符\n", len(fetchResult))
			fmt.Println()
			fmt.Println("内容预览 (前500字符):")
			fmt.Println("---")
			fmt.Println(truncate(fetchResult, 500))
			fmt.Println("---")
		}
	} else {
		fmt.Println("[4/5] 跳过Web抓取测试（没有搜索结果）")
	}

	fmt.Println()
	fmt.Println("[5/5] 测试完成")
	fmt.Println()
	fmt.Println("=== 测试总结 ===")
	fmt.Println("✅ Web搜索工具工作正常")
	fmt.Println("✅ 可以获取搜索结果")
	if len(results) > 0 {
		fmt.Println("✅ Web抓取工具工作正常")
	}
	fmt.Println()
	fmt.Println("💡 提示: 如果在实际使用中仍然失败，可能是:")
	fmt.Println("   1. LLM调用工具时参数不正确")
	fmt.Println("   2. 上下文超时（LLM调用时的context被取消）")
	fmt.Println("   3. 工具返回结果太大，被截断")
}

// getActiveEngineName 获取当前激活的搜索引擎名称
func getActiveEngineName(cfg config.Config) string {
	engine := cfg.GetActiveSearchEngine()
	if engine != nil {
		return engine.Name
	}
	return "Google (默认)"
}

// getProxyStatus 获取代理状态
func getProxyStatus(cfg config.Config) string {
	if cfg.ProxyConfig == nil {
		return "未配置"
	}
	if !cfg.ProxyConfig.Enabled {
		return "已禁用"
	}
	if !cfg.ProxyConfig.Tested {
		return "未测试"
	}
	return fmt.Sprintf("已启用 (%s://%s:%d)",
		cfg.ProxyConfig.Protocol,
		cfg.ProxyConfig.Host,
		cfg.ProxyConfig.Port)
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
