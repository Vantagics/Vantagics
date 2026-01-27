package agent

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// ConversationContext tracks conversation context for better follow-up understanding
type ConversationContext struct {
	ThreadID         string                 `json:"thread_id"`
	LastEntities     map[string]string      `json:"last_entities"`     // Type -> Value (e.g., "city" -> "北京")
	LastTopics       []string               `json:"last_topics"`       // Recent conversation topics
	LastIntent       string                 `json:"last_intent"`       // Last detected intent
	LastToolUsed     string                 `json:"last_tool_used"`    // Last tool that was called
	LastToolResult   string                 `json:"last_tool_result"`  // Summary of last tool result
	ConversationFlow []ChatTurn             `json:"conversation_flow"` // Recent turns
	LastUpdate       int64                  `json:"last_update"`
}

// ChatTurn represents a single turn in conversation
type ChatTurn struct {
	UserMessage      string            `json:"user_message"`
	AssistantMessage string            `json:"assistant_message"`
	ExtractedEntities map[string]string `json:"extracted_entities"`
	Intent           string            `json:"intent"`
	Timestamp        int64             `json:"timestamp"`
}

// ConversationContextManager manages conversation context per thread
type ConversationContextManager struct {
	contexts map[string]*ConversationContext
	mu       sync.RWMutex
}

// NewConversationContextManager creates a new manager
func NewConversationContextManager() *ConversationContextManager {
	return &ConversationContextManager{
		contexts: make(map[string]*ConversationContext),
	}
}

// GetOrCreateContext gets or creates context for a thread
func (m *ConversationContextManager) GetOrCreateContext(threadID string) *ConversationContext {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx, exists := m.contexts[threadID]; exists {
		return ctx
	}

	ctx := &ConversationContext{
		ThreadID:         threadID,
		LastEntities:     make(map[string]string),
		LastTopics:       []string{},
		ConversationFlow: []ChatTurn{},
		LastUpdate:       time.Now().Unix(),
	}
	m.contexts[threadID] = ctx
	return ctx
}

// UpdateFromUserMessage extracts entities and intent from user message
func (m *ConversationContextManager) UpdateFromUserMessage(threadID, userMessage string) {
	ctx := m.GetOrCreateContext(threadID)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract entities from user message
	entities := extractEntities(userMessage)
	for k, v := range entities {
		ctx.LastEntities[k] = v
	}

	// Detect intent
	intent := detectIntent(userMessage)
	if intent != "" {
		ctx.LastIntent = intent
	}

	// Add to conversation flow
	turn := ChatTurn{
		UserMessage:       userMessage,
		ExtractedEntities: entities,
		Intent:            intent,
		Timestamp:         time.Now().Unix(),
	}
	ctx.ConversationFlow = append(ctx.ConversationFlow, turn)

	// Keep only last 10 turns
	if len(ctx.ConversationFlow) > 10 {
		ctx.ConversationFlow = ctx.ConversationFlow[len(ctx.ConversationFlow)-10:]
	}

	ctx.LastUpdate = time.Now().Unix()
}

// UpdateFromAssistantResponse updates context from assistant response
func (m *ConversationContextManager) UpdateFromAssistantResponse(threadID, response, toolUsed, toolResult string) {
	ctx := m.GetOrCreateContext(threadID)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update last turn with assistant response
	if len(ctx.ConversationFlow) > 0 {
		ctx.ConversationFlow[len(ctx.ConversationFlow)-1].AssistantMessage = response
	}

	// Extract entities from response
	entities := extractEntities(response)
	for k, v := range entities {
		ctx.LastEntities[k] = v
	}

	// Update tool info
	if toolUsed != "" {
		ctx.LastToolUsed = toolUsed
	}
	if toolResult != "" {
		// Keep only summary of tool result
		if len(toolResult) > 500 {
			ctx.LastToolResult = toolResult[:500] + "..."
		} else {
			ctx.LastToolResult = toolResult
		}
	}

	ctx.LastUpdate = time.Now().Unix()
}

// GetContextForPrompt returns context formatted for LLM prompt
func (m *ConversationContextManager) GetContextForPrompt(threadID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, exists := m.contexts[threadID]
	if !exists || ctx == nil {
		return ""
	}

	// Check if context is stale (more than 30 minutes old)
	if time.Now().Unix()-ctx.LastUpdate > 1800 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n=== 对话上下文 ===\n")

	// Add relevant entities with strong emphasis
	if len(ctx.LastEntities) > 0 {
		sb.WriteString("**已知信息（必须使用，不要重复询问）**:\n")
		for entityType, value := range ctx.LastEntities {
			sb.WriteString("  - ")
			sb.WriteString(entityType)
			sb.WriteString(": ")
			sb.WriteString(value)
			sb.WriteString("\n")
		}
	}

	// Add last intent if relevant
	if ctx.LastIntent != "" {
		sb.WriteString("上次意图: ")
		sb.WriteString(ctx.LastIntent)
		sb.WriteString("\n")
	}

	// Add recent conversation summary
	if len(ctx.ConversationFlow) > 0 {
		sb.WriteString("最近对话:\n")
		// Show last 3 turns
		start := 0
		if len(ctx.ConversationFlow) > 3 {
			start = len(ctx.ConversationFlow) - 3
		}
		for i := start; i < len(ctx.ConversationFlow); i++ {
			turn := ctx.ConversationFlow[i]
			userMsg := turn.UserMessage
			if len(userMsg) > 100 {
				userMsg = userMsg[:100] + "..."
			}
			sb.WriteString("  用户: ")
			sb.WriteString(userMsg)
			sb.WriteString("\n")
			
			if turn.AssistantMessage != "" {
				assistMsg := turn.AssistantMessage
				if len(assistMsg) > 100 {
					assistMsg = assistMsg[:100] + "..."
				}
				sb.WriteString("  助手: ")
				sb.WriteString(assistMsg)
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("=== 上下文结束 ===\n")
	sb.WriteString("⚠️ 重要: 如果用户的问题涉及上述已知信息（如城市、地点），请直接使用该信息，不要重复询问用户！\n")
	sb.WriteString("例如: 如果已知城市是'三亚'，用户问'天气怎样'，直接查询三亚的天气。\n")

	return sb.String()
}

// ResolveReferences resolves pronouns and references in user message
func (m *ConversationContextManager) ResolveReferences(threadID, userMessage string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, exists := m.contexts[threadID]
	if !exists || ctx == nil {
		return userMessage
	}

	resolved := userMessage

	// Resolve city references for weather queries
	if isWeatherQuery(userMessage) && !hasExplicitCity(userMessage) {
		if city, ok := ctx.LastEntities["city"]; ok {
			// Add city context to the message
			resolved = userMessage + " (城市: " + city + ")"
		}
	}

	// Resolve "这个"/"那个" references
	if strings.Contains(userMessage, "这个") || strings.Contains(userMessage, "那个") {
		// Try to resolve from last entities
		for entityType, value := range ctx.LastEntities {
			if entityType != "city" { // Already handled above
				resolved = strings.Replace(resolved, "这个", value, 1)
				resolved = strings.Replace(resolved, "那个", value, 1)
				break
			}
		}
	}

	return resolved
}

// ClearContext clears context for a thread
func (m *ConversationContextManager) ClearContext(threadID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.contexts, threadID)
}

// Helper functions

// extractEntities extracts named entities from text
func extractEntities(text string) map[string]string {
	entities := make(map[string]string)

	// Extract cities - use a more comprehensive approach
	// First try known cities
	knownCities := map[string]string{
		// 一线城市
		"北京": "北京", "Beijing": "北京",
		"上海": "上海", "Shanghai": "上海",
		"广州": "广州", "Guangzhou": "广州",
		"深圳": "深圳", "Shenzhen": "深圳",
		// 新一线城市
		"杭州": "杭州", "Hangzhou": "杭州",
		"成都": "成都", "Chengdu": "成都",
		"武汉": "武汉", "Wuhan": "武汉",
		"西安": "西安", "Xi'an": "西安", "Xian": "西安",
		"南京": "南京", "Nanjing": "南京",
		"重庆": "重庆", "Chongqing": "重庆",
		"天津": "天津", "Tianjin": "天津",
		"苏州": "苏州", "Suzhou": "苏州",
		"郑州": "郑州", "Zhengzhou": "郑州",
		"长沙": "长沙", "Changsha": "长沙",
		"东莞": "东莞", "Dongguan": "东莞",
		"沈阳": "沈阳", "Shenyang": "沈阳",
		"青岛": "青岛", "Qingdao": "青岛",
		"宁波": "宁波", "Ningbo": "宁波",
		"昆明": "昆明", "Kunming": "昆明",
		// 旅游城市
		"三亚": "三亚", "Sanya": "三亚",
		"海口": "海口", "Haikou": "海口",
		"厦门": "厦门", "Xiamen": "厦门",
		"大连": "大连", "Dalian": "大连",
		"桂林": "桂林", "Guilin": "桂林",
		"丽江": "丽江", "Lijiang": "丽江",
		"西双版纳": "西双版纳", "Xishuangbanna": "西双版纳",
		"张家界": "张家界", "Zhangjiajie": "张家界",
		"黄山": "黄山", "Huangshan": "黄山",
		"九寨沟": "九寨沟", "Jiuzhaigou": "九寨沟",
		// 其他省会城市
		"哈尔滨": "哈尔滨", "Harbin": "哈尔滨",
		"长春": "长春", "Changchun": "长春",
		"石家庄": "石家庄", "Shijiazhuang": "石家庄",
		"太原": "太原", "Taiyuan": "太原",
		"呼和浩特": "呼和浩特", "Hohhot": "呼和浩特",
		"济南": "济南", "Jinan": "济南",
		"合肥": "合肥", "Hefei": "合肥",
		"福州": "福州", "Fuzhou": "福州",
		"南昌": "南昌", "Nanchang": "南昌",
		"贵阳": "贵阳", "Guiyang": "贵阳",
		"南宁": "南宁", "Nanning": "南宁",
		"兰州": "兰州", "Lanzhou": "兰州",
		"银川": "银川", "Yinchuan": "银川",
		"西宁": "西宁", "Xining": "西宁",
		"乌鲁木齐": "乌鲁木齐", "Urumqi": "乌鲁木齐",
		"拉萨": "拉萨", "Lhasa": "拉萨",
		// 国际城市
		"东京": "东京", "Tokyo": "东京",
		"纽约": "纽约", "New York": "纽约",
		"伦敦": "伦敦", "London": "伦敦",
		"巴黎": "巴黎", "Paris": "巴黎",
		"首尔": "首尔", "Seoul": "首尔",
		"新加坡": "新加坡", "Singapore": "新加坡",
		"曼谷": "曼谷", "Bangkok": "曼谷",
		"悉尼": "悉尼", "Sydney": "悉尼",
		"洛杉矶": "洛杉矶", "Los Angeles": "洛杉矶",
		"旧金山": "旧金山", "San Francisco": "旧金山",
	}

	// Check for known cities
	for pattern, cityName := range knownCities {
		if strings.Contains(text, pattern) {
			entities["city"] = cityName
			break
		}
	}

	// If no known city found, try to extract Chinese city names dynamically
	// Pattern: X亚, X州, X京, X海, X山, X口 etc. (common Chinese city name suffixes)
	if _, hasCity := entities["city"]; !hasCity {
		cityPatterns := []string{
			`([一-龥]{1,3}(?:市|县|区|镇))`,  // 市/县/区/镇
			`([一-龥]{2,4}(?:亚|州|京|海|山|口|岛|江|河|湖|港|门|关|城|原|川|谷))`, // Common suffixes
		}
		for _, pattern := range cityPatterns {
			re := regexp.MustCompile(pattern)
			if match := re.FindString(text); match != "" {
				// Filter out common non-city words
				nonCities := []string{"分析", "统计", "查询", "数据", "天气", "温度", "预报"}
				isCity := true
				for _, nc := range nonCities {
					if strings.Contains(match, nc) {
						isCity = false
						break
					}
				}
				if isCity && len(match) >= 2 {
					entities["city"] = match
					break
				}
			}
		}
	}

	// Extract dates
	datePatterns := []string{
		`今天|today`,
		`明天|tomorrow`,
		`后天|day after tomorrow`,
		`\d{4}[-/]\d{1,2}[-/]\d{1,2}`,
		`\d{1,2}月\d{1,2}日`,
	}
	for _, pattern := range datePatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if match := re.FindString(text); match != "" {
			entities["date"] = match
			break
		}
	}

	// Extract numbers/amounts
	amountPattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(元|美元|万|亿|%|度|℃)`)
	if matches := amountPattern.FindStringSubmatch(text); len(matches) > 0 {
		entities["amount"] = matches[0]
	}

	// Extract location/place mentions (for travel context)
	placePatterns := []string{
		`去([一-龥]{2,4})`,      // 去三亚
		`到([一-龥]{2,4})`,      // 到三亚
		`在([一-龥]{2,4})`,      // 在三亚
		`([一-龥]{2,4})好玩`,    // 三亚好玩
		`([一-龥]{2,4})怎么样`,  // 三亚怎么样
		`([一-龥]{2,4})的天气`,  // 三亚的天气
	}
	if _, hasCity := entities["city"]; !hasCity {
		for _, pattern := range placePatterns {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 {
				place := matches[1]
				// Validate it looks like a place name
				if len(place) >= 2 && len(place) <= 4 {
					entities["city"] = place
					entities["place"] = place
					break
				}
			}
		}
	}

	return entities
}

// detectIntent detects the intent of a message
func detectIntent(text string) string {
	textLower := strings.ToLower(text)

	// Weather intent
	if strings.Contains(textLower, "天气") || strings.Contains(textLower, "weather") ||
		strings.Contains(textLower, "温度") || strings.Contains(textLower, "气温") ||
		strings.Contains(textLower, "下雨") || strings.Contains(textLower, "晴") {
		return "weather_query"
	}

	// Time intent
	if strings.Contains(textLower, "时间") || strings.Contains(textLower, "几点") ||
		strings.Contains(textLower, "日期") || strings.Contains(textLower, "time") {
		return "time_query"
	}

	// Search intent
	if strings.Contains(textLower, "搜索") || strings.Contains(textLower, "查询") ||
		strings.Contains(textLower, "search") || strings.Contains(textLower, "查找") {
		return "search_query"
	}

	// Data analysis intent
	if strings.Contains(textLower, "分析") || strings.Contains(textLower, "统计") ||
		strings.Contains(textLower, "图表") || strings.Contains(textLower, "趋势") {
		return "data_analysis"
	}

	return ""
}

// isWeatherQuery checks if the message is about weather
func isWeatherQuery(text string) bool {
	textLower := strings.ToLower(text)
	weatherKeywords := []string{"天气", "weather", "温度", "气温", "下雨", "晴", "阴", "多云", "预报"}
	for _, kw := range weatherKeywords {
		if strings.Contains(textLower, kw) {
			return true
		}
	}
	return false
}

// hasExplicitCity checks if the message contains an explicit city name
func hasExplicitCity(text string) bool {
	// Must match all cities in extractEntities knownCities map
	cityPatterns := []string{
		// 一线城市
		`北京|上海|广州|深圳`,
		`Beijing|Shanghai|Guangzhou|Shenzhen`,
		// 新一线城市
		`杭州|成都|武汉|西安|南京|重庆|天津|苏州|郑州|长沙|东莞|沈阳|青岛|宁波|昆明`,
		`Hangzhou|Chengdu|Wuhan|Xian|Xi'an|Nanjing|Chongqing|Tianjin|Suzhou|Zhengzhou|Changsha|Dongguan|Shenyang|Qingdao|Ningbo|Kunming`,
		// 旅游城市
		`三亚|海口|厦门|大连|桂林|丽江|西双版纳|张家界|黄山|九寨沟`,
		`Sanya|Haikou|Xiamen|Dalian|Guilin|Lijiang|Xishuangbanna|Zhangjiajie|Huangshan|Jiuzhaigou`,
		// 其他省会城市
		`哈尔滨|长春|石家庄|太原|呼和浩特|济南|合肥|福州|南昌|贵阳|南宁|兰州|银川|西宁|乌鲁木齐|拉萨`,
		`Harbin|Changchun|Shijiazhuang|Taiyuan|Hohhot|Jinan|Hefei|Fuzhou|Nanchang|Guiyang|Nanning|Lanzhou|Yinchuan|Xining|Urumqi|Lhasa`,
		// 国际城市
		`东京|纽约|伦敦|巴黎|首尔|新加坡|曼谷|悉尼|洛杉矶|旧金山`,
		`Tokyo|New York|London|Paris|Seoul|Singapore|Bangkok|Sydney|Los Angeles|San Francisco`,
	}
	for _, pattern := range cityPatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
