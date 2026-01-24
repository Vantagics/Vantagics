package agent

import (
"strings"
"sync"
)

type FewShotExample struct {
Domain      string           `json:"domain"`
UserMessage string           `json:"user_message"`
Intent      IntentSuggestion `json:"intent"`
Language    string           `json:"language"`
}

type ExampleProviderImpl struct {
examples    map[string][]FewShotExample
mu          sync.RWMutex
initialized bool
}

const (
DomainSales        = "sales"
DomainFinance      = "finance"
DomainUserBehavior = "user_behavior"
DomainGeneral      = "general"
)

const (
LanguageEnglish = "en"
LanguageChinese = "zh"
)

var domainKeywords = map[string][]string{
DomainSales: {
"sales", "sale", "revenue", "order", "orders", "customer", "customers",
"product", "products", "purchase", "purchases", "transaction", "transactions",
"invoice", "invoices", "discount", "discounts", "price", "prices",
"quantity", "qty", "unit", "units", "sku", "item", "items",
"cart", "checkout", "payment", "payments", "refund", "refunds",
"channel", "channels", "store", "stores", "shop", "shops",
"销售", "销量", "营收", "收入", "订单", "客户", "产品", "商品", "购买", "交易",
"价格", "数量", "金额", "折扣", "支付", "退款", "渠道", "门店",
},
DomainFinance: {
"finance", "financial", "budget", "budgets", "expense", "expenses",
"cost", "costs", "profit", "profits", "loss", "losses", "margin", "margins",
"asset", "assets", "liability", "liabilities", "equity", "equities",
"balance", "balances", "account", "accounts", "ledger", "ledgers",
"tax", "taxes", "interest", "interests", "loan", "loans", "debt", "debts",
"investment", "investments", "dividend", "dividends", "cash", "cashflow",
"receivable", "receivables", "payable", "payables", "depreciation",
"amortization", "capital", "roi", "ebitda", "gross", "net",
"财务", "预算", "费用", "成本", "利润", "亏损", "资产", "负债",
"账户", "税", "利息", "贷款", "投资", "股息", "现金",
},
DomainUserBehavior: {
"user", "users", "visitor", "visitors", "session", "sessions",
"click", "clicks", "view", "views", "pageview", "pageviews",
"bounce", "bounces", "conversion", "conversions", "engagement",
"retention", "churn", "active", "inactive", "login", "logins",
"signup", "signups", "register", "registration", "registrations",
"behavior", "behaviour", "action", "actions", "event", "events",
"track", "tracking", "analytics", "metric", "metrics",
"duration", "frequency", "recency", "funnel", "funnels",
"cohort", "cohorts", "segment", "segments", "journey", "journeys",
"用户", "访客", "会话", "点击", "浏览", "页面", "转化", "留存",
"活跃", "登录", "注册", "行为", "事件", "追踪", "分析",
},
}
func NewExampleProviderImpl() *ExampleProviderImpl {
p := &ExampleProviderImpl{examples: make(map[string][]FewShotExample)}
p.initializeBuiltInExamples()
p.initialized = true
return p
}

func (e *ExampleProviderImpl) initializeBuiltInExamples() {
e.examples[DomainSales] = []FewShotExample{
{Domain: DomainSales, Language: LanguageChinese, UserMessage: "分析销售情况", Intent: IntentSuggestion{Title: "月度销售趋势", Description: "按月份分析销售额变化趋势", Icon: "chart_line", Query: "请按月份汇总销售额，绘制趋势图"}},
{Domain: DomainSales, Language: LanguageChinese, UserMessage: "看看产品表现", Intent: IntentSuggestion{Title: "产品销量排行", Description: "分析各产品的销售数量和金额", Icon: "trophy", Query: "请统计各产品的销售数量和金额，按销售额降序排列"}},
{Domain: DomainSales, Language: LanguageChinese, UserMessage: "客户分析", Intent: IntentSuggestion{Title: "客户价值分析", Description: "分析客户购买行为，识别高价值客户", Icon: "users", Query: "请按客户统计购买次数和总金额"}},
{Domain: DomainSales, Language: LanguageEnglish, UserMessage: "analyze sales", Intent: IntentSuggestion{Title: "Monthly Trend", Description: "Analyze monthly sales trends", Icon: "chart_line", Query: "Summarize sales by month, create trend chart"}},
{Domain: DomainSales, Language: LanguageEnglish, UserMessage: "product performance", Intent: IntentSuggestion{Title: "Product Ranking", Description: "Analyze product sales volume", Icon: "trophy", Query: "Calculate sales by product, rank by revenue"}},
{Domain: DomainSales, Language: LanguageEnglish, UserMessage: "customer analysis", Intent: IntentSuggestion{Title: "Customer Value", Description: "Analyze customer purchase behavior", Icon: "users", Query: "Calculate purchase frequency by customer"}},
}
e.examples[DomainFinance] = []FewShotExample{
{Domain: DomainFinance, Language: LanguageChinese, UserMessage: "分析财务状况", Intent: IntentSuggestion{Title: "收支分析", Description: "分析收入和支出的构成", Icon: "dollar", Query: "请按类别汇总收入和支出"}},
{Domain: DomainFinance, Language: LanguageChinese, UserMessage: "看看成本情况", Intent: IntentSuggestion{Title: "成本结构分析", Description: "分析各项成本的构成", Icon: "chart_bar", Query: "请按成本类型汇总金额"}},
{Domain: DomainFinance, Language: LanguageChinese, UserMessage: "预算执行分析", Intent: IntentSuggestion{Title: "预算对比分析", Description: "对比实际支出与预算", Icon: "clipboard", Query: "请对比各部门实际支出与预算金额"}},
{Domain: DomainFinance, Language: LanguageEnglish, UserMessage: "financial status", Intent: IntentSuggestion{Title: "Income Analysis", Description: "Analyze income and expense", Icon: "dollar", Query: "Summarize income and expenses by category"}},
{Domain: DomainFinance, Language: LanguageEnglish, UserMessage: "cost breakdown", Intent: IntentSuggestion{Title: "Cost Structure", Description: "Analyze cost composition", Icon: "chart_bar", Query: "Summarize costs by type"}},
{Domain: DomainFinance, Language: LanguageEnglish, UserMessage: "budget analysis", Intent: IntentSuggestion{Title: "Budget Comparison", Description: "Compare actual vs budget", Icon: "clipboard", Query: "Compare actual vs budget by department"}},
}
e.examples[DomainUserBehavior] = []FewShotExample{
{Domain: DomainUserBehavior, Language: LanguageChinese, UserMessage: "分析用户行为", Intent: IntentSuggestion{Title: "用户活跃度分析", Description: "分析用户的活跃程度", Icon: "mobile", Query: "请统计日活跃用户数(DAU)"}},
{Domain: DomainUserBehavior, Language: LanguageChinese, UserMessage: "看看转化情况", Intent: IntentSuggestion{Title: "转化漏斗分析", Description: "分析用户转化各环节", Icon: "funnel", Query: "请构建用户转化漏斗"}},
{Domain: DomainUserBehavior, Language: LanguageChinese, UserMessage: "用户留存分析", Intent: IntentSuggestion{Title: "留存率分析", Description: "分析用户的留存情况", Icon: "chart_bar", Query: "请计算次日留存、7日留存率"}},
{Domain: DomainUserBehavior, Language: LanguageEnglish, UserMessage: "user behavior", Intent: IntentSuggestion{Title: "User Activity", Description: "Analyze user activity levels", Icon: "mobile", Query: "Calculate DAU and WAU"}},
{Domain: DomainUserBehavior, Language: LanguageEnglish, UserMessage: "conversion rates", Intent: IntentSuggestion{Title: "Conversion Funnel", Description: "Analyze user conversion", Icon: "funnel", Query: "Build conversion funnel"}},
{Domain: DomainUserBehavior, Language: LanguageEnglish, UserMessage: "retention analysis", Intent: IntentSuggestion{Title: "Retention Rate", Description: "Analyze user retention", Icon: "chart_bar", Query: "Calculate D1, D7 retention rates"}},
}
e.examples[DomainGeneral] = []FewShotExample{
{Domain: DomainGeneral, Language: LanguageChinese, UserMessage: "分析数据", Intent: IntentSuggestion{Title: "数据概览", Description: "对数据进行整体概览", Icon: "chart_bar", Query: "请展示数据的基本统计信息"}},
{Domain: DomainGeneral, Language: LanguageChinese, UserMessage: "看看趋势", Intent: IntentSuggestion{Title: "时间趋势分析", Description: "分析数据随时间的变化", Icon: "chart_line", Query: "请按时间维度汇总数据"}},
{Domain: DomainGeneral, Language: LanguageChinese, UserMessage: "对比分析", Intent: IntentSuggestion{Title: "分组对比分析", Description: "按不同维度分组对比", Icon: "balance", Query: "请按主要分类维度分组"}},
{Domain: DomainGeneral, Language: LanguageEnglish, UserMessage: "analyze data", Intent: IntentSuggestion{Title: "Data Overview", Description: "Get an overview of the data", Icon: "chart_bar", Query: "Show basic statistics"}},
{Domain: DomainGeneral, Language: LanguageEnglish, UserMessage: "show trends", Intent: IntentSuggestion{Title: "Time Trend", Description: "Analyze how data changes", Icon: "chart_line", Query: "Aggregate data by time dimension"}},
{Domain: DomainGeneral, Language: LanguageEnglish, UserMessage: "compare groups", Intent: IntentSuggestion{Title: "Group Comparison", Description: "Compare data across groups", Icon: "balance", Query: "Group by main category dimension"}},
}
}

func (e *ExampleProviderImpl) GetExamplesForDomain(domain string) []FewShotExample {
e.mu.RLock()
defer e.mu.RUnlock()
if examples, ok := e.examples[domain]; ok {
return examples
}
return []FewShotExample{}
}

func (e *ExampleProviderImpl) AddExample(domain string, example FewShotExample) {
e.mu.Lock()
defer e.mu.Unlock()
if _, ok := e.examples[domain]; !ok {
e.examples[domain] = []FewShotExample{}
}
e.examples[domain] = append(e.examples[domain], example)
}

func (e *ExampleProviderImpl) IsInitialized() bool {
e.mu.RLock()
defer e.mu.RUnlock()
return e.initialized
}

func (e *ExampleProviderImpl) DetectDomain(columns []string, tableName string) string {
allTerms := make([]string, 0, len(columns)+1)
allTerms = append(allTerms, strings.ToLower(tableName))
for _, col := range columns {
allTerms = append(allTerms, strings.ToLower(col))
}
domainScores := make(map[string]int)
for domain, keywords := range domainKeywords {
score := 0
for _, term := range allTerms {
for _, keyword := range keywords {
if strings.Contains(term, keyword) {
score++
break
}
}
}
domainScores[domain] = score
}
maxScore := 0
detectedDomain := DomainGeneral
for domain, score := range domainScores {
if score > maxScore {
maxScore = score
detectedDomain = domain
}
}
if maxScore == 0 {
return DomainGeneral
}
return detectedDomain
}

func (e *ExampleProviderImpl) GetExamples(domain string, language string, count int) []FewShotExample {
e.mu.RLock()
defer e.mu.RUnlock()
normalizedLang := normalizeLanguage(language)
domainExamples, ok := e.examples[domain]
if !ok || len(domainExamples) == 0 {
domainExamples = e.examples[DomainGeneral]
}
var filteredExamples []FewShotExample
for _, example := range domainExamples {
if example.Language == normalizedLang {
filteredExamples = append(filteredExamples, example)
}
}
if len(filteredExamples) == 0 {
for _, example := range domainExamples {
filteredExamples = append(filteredExamples, example)
}
}
if len(filteredExamples) == 0 {
generalExamples := e.examples[DomainGeneral]
for _, example := range generalExamples {
if example.Language == normalizedLang {
filteredExamples = append(filteredExamples, example)
}
}
if len(filteredExamples) == 0 {
filteredExamples = generalExamples
}
}
if count > 0 && len(filteredExamples) > count {
filteredExamples = filteredExamples[:count]
}
return filteredExamples
}

func (e *ExampleProviderImpl) BuildExampleSection(examples []FewShotExample, language string) string {
if len(examples) == 0 {
return ""
}
normalizedLang := normalizeLanguage(language)
var builder strings.Builder
if normalizedLang == LanguageChinese {
builder.WriteString("\n## 参考示例\n以下是一些意图理解的示例，请参考这些示例的格式和风格：\n\n")
} else {
builder.WriteString("\n## Reference Examples\nHere are some examples of intent understanding. Please follow the format and style:\n\n")
}
for i, example := range examples {
if normalizedLang == LanguageChinese {
builder.WriteString("### 示例 ")
builder.WriteString(intToString(i + 1))
builder.WriteString("\n**用户输入**: ")
builder.WriteString(example.UserMessage)
builder.WriteString("\n**意图建议**:\n- 标题: ")
builder.WriteString(example.Intent.Title)
builder.WriteString("\n- 描述: ")
builder.WriteString(example.Intent.Description)
builder.WriteString("\n- 图标: ")
builder.WriteString(example.Intent.Icon)
builder.WriteString("\n- 查询: ")
builder.WriteString(example.Intent.Query)
builder.WriteString("\n\n")
} else {
builder.WriteString("### Example ")
builder.WriteString(intToString(i + 1))
builder.WriteString("\n**User Input**: ")
builder.WriteString(example.UserMessage)
builder.WriteString("\n**Intent Suggestion**:\n- Title: ")
builder.WriteString(example.Intent.Title)
builder.WriteString("\n- Description: ")
builder.WriteString(example.Intent.Description)
builder.WriteString("\n- Icon: ")
builder.WriteString(example.Intent.Icon)
builder.WriteString("\n- Query: ")
builder.WriteString(example.Intent.Query)
builder.WriteString("\n\n")
}
}
return builder.String()
}

func normalizeLanguage(language string) string {
lang := strings.ToLower(strings.TrimSpace(language))
if lang == "zh" || lang == "zh-cn" || lang == "zh-hans" || lang == "chinese" {
return LanguageChinese
}
for _, r := range language {
if r >= 0x4E00 && r <= 0x9FFF {
return LanguageChinese
}
}
return LanguageEnglish
}

func intToString(n int) string {
if n == 0 {
return "0"
}
var digits []byte
negative := n < 0
if negative {
n = -n
}
for n > 0 {
digits = append([]byte{byte('0' + n%10)}, digits...)
n /= 10
}
if negative {
digits = append([]byte{'-'}, digits...)
}
return string(digits)
}
