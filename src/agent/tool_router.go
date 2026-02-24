package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// ToolRouterResult 路由结果
type ToolRouterResult struct {
	NeedsTools     bool     // 是否需要使用工�?
	SuggestedTools []string // 建议使用的工具列�?
	Confidence     float64  // 置信�?0-1
	Reason         string   // 路由原因
}

// ToolRouter 智能工具路由�?
// 使用多种非LLM方法判断用户请求是否需要使用工�?
type ToolRouter struct {
	logFunc func(string)

	// 预编译的正则表达�?
	timePatterns     []*regexp.Regexp
	locationPatterns []*regexp.Regexp
	searchPatterns   []*regexp.Regexp
	questionPatterns []*regexp.Regexp
	analysisPatterns []*regexp.Regexp
}

// NewToolRouter 创建新的工具路由�?
func NewToolRouter(logFunc func(string)) *ToolRouter {
	router := &ToolRouter{
		logFunc: logFunc,
	}
	router.compilePatterns()
	return router
}

// compilePatterns 预编译所有正则表达式
func (r *ToolRouter) compilePatterns() {
	// 时间相关模式
	r.timePatterns = compilePatterns([]string{
		`(?i)几点`,
		`(?i)什么时间`,
		`(?i)现在.*时间`,
		`(?i)当前.*时间`,
		`(?i)今天.*日期`,
		`(?i)星期几`,
		`(?i)周几`,
		`(?i)几号`,
		`(?i)几月`,
		`(?i)what\s+time`,
		`(?i)current\s+time`,
		`(?i)what\s+day`,
		`(?i)what\s+date`,
		`(?i)today'?s?\s+date`,
	})

	// 位置相关模式
	r.locationPatterns = compilePatterns([]string{
		`(?i)我在哪`,
		`(?i)我的位置`,
		`(?i)我现在在`,
		`(?i)我所在`,
		`(?i)当前位置`,
		`(?i)我在什么`,
		`(?i)这里是哪`,
		`(?i)这是哪`,
		`(?i)本地`,
		`(?i)附近`,
		`(?i)周边`,
		`(?i)where\s+am\s+i`,
		`(?i)my\s+location`,
		`(?i)current\s+location`,
		`(?i)nearby`,
		`(?i)around\s+here`,
		`(?i)local\s+`,
	})

	// 需要网络搜索的模式
	r.searchPatterns = compilePatterns([]string{
		// 天气相关 - 更宽泛的匹配
		`(?i)天气`,           // 任何包含"天气"的查�?
		`(?i)�?*下雨`,
		`(?i)�?*下雪`,
		`(?i)气温`,           // 任何包含"气温"的查�?
		`(?i)温度`,           // 任何包含"温度"的查�?
		`(?i)几度`,           // "今天几度"
		`(?i)多少度`,         // "现在多少�?
		`(?i)weather`,
		`(?i)forecast`,
		// 新闻/实时信息
		`(?i)新闻`,           // 任何包含"新闻"的查�?
		`(?i)头条`,
		`(?i)latest\s+news`,
		`(?i)recent\s+news`,
		`(?i)news`,
		// 价格/股票/汇率
		`(?i)股票`,
		`(?i)股价`,
		`(?i)汇率`,
		`(?i)多少钱`,
		`(?i)stock`,
		`(?i)exchange\s+rate`,
		`(?i)price`,
		// 航班/交�?
		`(?i)航班`,
		`(?i)机票`,
		`(?i)�?*到`,
		`(?i)�?*的航班`,
		`(?i)�?*的机票`,     // "去成都的机票"
		`(?i)�?*机票`,       // "去成都机�?
		`(?i)�?*机票`,       // "到成都机�?
		`(?i).*�?*`,         // "飞成�?
		`(?i)flight`,
		`(?i)flights?\s+to`,
		// 酒店/住宿
		`(?i)酒店`,
		`(?i)住宿`,
		`(?i)hotel`,
		// 比赛/赛事
		`(?i)比赛`,
		`(?i)比分`,
		`(?i)赛事`,
		`(?i)世界杯`,
		`(?i)结果`,           // 比赛结果、搜索结果等
		`(?i)score`,
		`(?i)match`,
		`(?i)game\s+result`,
		// 搜索意图
		`(?i)帮我查`,
		`(?i)帮我搜`,
		`(?i)搜索一下`,
		`(?i)查一下`,
		`(?i)search\s+for`,
		`(?i)look\s+up`,
		`(?i)find\s+out`,
	})

	// 数据分析相关模式
	r.analysisPatterns = compilePatterns([]string{
		`(?i)分析`,            // 任何包含"分析"的消�?
		`(?i)看看.*数据`,
		`(?i)数据源`,
		`(?i)有哪些数据`,
		`(?i)analyze`,
		`(?i)analysis`,
		`(?i)data\s*source`,
		`(?i)dataset`,
	})

	// 疑问句模式（用于辅助判断�?
	r.questionPatterns = compilePatterns([]string{
		`(?i)^(什么|哪|谁|怎么|为什么|多少|几|是否|能否|可以|有没�?`,
		`(?i)(什么|哪|谁|怎么|为什么|多少|�?\??\s*$`,
		`(?i)^(what|where|who|when|why|how|which|is|are|can|could|do|does|did)`,
		`(?i)\?$`,
	})
}

// compilePatterns 编译正则表达式列�?
func compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// Route 路由用户请求，判断是否需要使用工�?
func (r *ToolRouter) Route(message string) ToolRouterResult {
	message = strings.TrimSpace(message)
	if message == "" {
		return ToolRouterResult{NeedsTools: false, Confidence: 1.0, Reason: "empty message"}
	}

	var suggestedTools []string
	var reasons []string
	totalScore := 0.0

	// 1. 检查时间相关模�?
	if r.matchesAny(message, r.timePatterns) {
		suggestedTools = append(suggestedTools, "get_local_time")
		reasons = append(reasons, "time_pattern")
		totalScore += 0.9
		r.log("[TOOL-ROUTER] Matched time pattern")
	}

	// 2. 检查位置相关模�?
	if r.matchesAny(message, r.locationPatterns) {
		suggestedTools = append(suggestedTools, "get_device_location")
		reasons = append(reasons, "location_pattern")
		totalScore += 0.9
		r.log("[TOOL-ROUTER] Matched location pattern")
	}

	// 3. 检查搜索相关模�?
	if r.matchesAny(message, r.searchPatterns) {
		suggestedTools = append(suggestedTools, "web_search")
		reasons = append(reasons, "search_pattern")
		totalScore += 0.85
		r.log("[TOOL-ROUTER] Matched search pattern")
	}

	// 3.5 检查数据分析相关模�?
	if r.matchesAny(message, r.analysisPatterns) {
		suggestedTools = append(suggestedTools, "start_datasource_analysis")
		reasons = append(reasons, "analysis_pattern")
		totalScore += 0.9
		r.log("[TOOL-ROUTER] Matched analysis pattern")
	}

	// 4. 语义特征分析
	semanticScore, semanticTools, semanticReason := r.analyzeSemanticFeatures(message)
	if semanticScore > 0 {
		for _, tool := range semanticTools {
			if !containsStr(suggestedTools, tool) {
				suggestedTools = append(suggestedTools, tool)
			}
		}
		reasons = append(reasons, semanticReason)
		totalScore += semanticScore
		r.log("[TOOL-ROUTER] Semantic analysis: score=%.2f, reason=%s", semanticScore, semanticReason)
	}

	// 5. 疑问句检测（辅助判断�?
	if r.matchesAny(message, r.questionPatterns) {
		// 疑问句增加一点分数，但不单独触发工具使用
		if totalScore > 0 {
			totalScore += 0.1
		}
	}

	// 计算最终置信度
	confidence := totalScore
	if confidence > 1.0 {
		confidence = 1.0
	}
	needsTools := confidence >= 0.5 && len(suggestedTools) > 0

	result := ToolRouterResult{
		NeedsTools:     needsTools,
		SuggestedTools: suggestedTools,
		Confidence:     confidence,
		Reason:         strings.Join(reasons, ", "),
	}

	r.log("[TOOL-ROUTER] Result: needsTools=%v, confidence=%.2f, tools=%v, reason=%s",
		result.NeedsTools, result.Confidence, result.SuggestedTools, result.Reason)

	return result
}

// analyzeSemanticFeatures 分析语义特征
func (r *ToolRouter) analyzeSemanticFeatures(message string) (float64, []string, string) {
	var tools []string
	var reasons []string
	score := 0.0

	// Lowercase once for all Contains checks below
	msgLower := strings.ToLower(message)

	// 检测实时性需�?
	realtimeIndicators := []string{
		"现在", "当前", "今天", "此刻", "目前", "实时", "最�?,
		"now", "current", "today", "right now", "at the moment", "latest", "recent",
	}
	for _, indicator := range realtimeIndicators {
		if strings.Contains(msgLower, indicator) {
			score += 0.3
			reasons = append(reasons, "realtime_indicator")
			break
		}
	}

	// 检测地理位置相关词�?
	geoIndicators := []string{
		"这里", "这儿", "本地", "附近", "周边", "当地",
		"here", "local", "nearby", "around",
	}
	for _, indicator := range geoIndicators {
		if strings.Contains(msgLower, indicator) {
			tools = append(tools, "get_device_location")
			score += 0.4
			reasons = append(reasons, "geo_indicator")
			break
		}
	}

	// 检测外部信息需�?
	externalInfoIndicators := []string{
		"�?, "�?, "�?, "看看", "告诉�?,
		"search", "find", "look", "tell me", "show me",
	}
	externalInfoTopics := []string{
		"天气", "新闻", "价格", "股票", "汇率", "比赛", "航班", "酒店",
		"weather", "news", "price", "stock", "exchange", "match", "flight", "hotel",
	}

	hasIndicator := false
	hasTopic := false
	for _, indicator := range externalInfoIndicators {
		if strings.Contains(msgLower, indicator) {
			hasIndicator = true
			break
		}
	}
	for _, topic := range externalInfoTopics {
		if strings.Contains(msgLower, topic) {
			hasTopic = true
			break
		}
	}
	if hasIndicator || hasTopic {
		tools = append(tools, "web_search")
		score += 0.4
		if hasIndicator && hasTopic {
			score += 0.2
		}
		reasons = append(reasons, "external_info_need")
	}

	// 检�?�?+"位置/地点"组合
	if strings.Contains(msgLower, "�?) {
		locationWords := []string{"在哪", "位置", "地方", "城市", "国家", "地址"}
		for _, word := range locationWords {
			if strings.Contains(msgLower, word) {
				tools = append(tools, "get_device_location")
				score += 0.5
				reasons = append(reasons, "self_location_query")
				break
			}
		}
	}

	// 检测导�?下载需�?
	exportIndicators := []string{
		"导出", "下载", "保存", "生成报告", "生成文件",
		"export", "download", "save as", "generate report",
		"excel", "pdf", "ppt", "csv", "xlsx",
	}
	for _, indicator := range exportIndicators {
		if strings.Contains(msgLower, indicator) {
			tools = append(tools, "export_data")
			score += 0.7
			reasons = append(reasons, "export_need")
			break
		}
	}

	// 去重
	tools = unique(tools)
	reasons = unique(reasons)

	return score, tools, strings.Join(reasons, "+")
}

// matchesAny 检查消息是否匹配任意一个模�?
func (r *ToolRouter) matchesAny(message string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(message) {
			return true
		}
	}
	return false
}

// log 记录日志
func (r *ToolRouter) log(format string, args ...interface{}) {
	if r.logFunc != nil {
		r.logFunc(fmt.Sprintf(format, args...))
	}
}

// 辅助函数
func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func unique(slice []string) []string {
	seen := make(map[string]bool, len(slice))
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

