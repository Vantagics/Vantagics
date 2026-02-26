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
"销售", "订单", "营收", "收入", "金额", "客户", "产品", "商品", "数量", "价格",
"售价", "折扣", "库存", "退款", "支付", "退货", "渠道", "门店",
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
"����", "Ԥ��", "����", "�ɱ�", "����", "����", "�ʲ�", "��ծ",
"�˻�", "˰", "��Ϣ", "����", "Ͷ��", "��Ϣ", "�ֽ�",
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
"�û�", "�ÿ�", "�Ự", "���", "���", "ҳ��", "ת��", "����",
"��Ծ", "��¼", "ע��", "��Ϊ", "�¼�", "׷��", "����",
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
{Domain: DomainSales, Language: LanguageChinese, UserMessage: "�����������", Intent: IntentSuggestion{Title: "�¶���������", Description: "���·ݷ������۶�仯����", Icon: "chart_line", Query: "�밴�·ݻ������۶��������ͼ"}},
{Domain: DomainSales, Language: LanguageChinese, UserMessage: "������Ʒ����", Intent: IntentSuggestion{Title: "��Ʒ��������", Description: "��������Ʒ�����������ͽ��", Icon: "trophy", Query: "��ͳ�Ƹ���Ʒ�����������ͽ������۶������"}},
{Domain: DomainSales, Language: LanguageChinese, UserMessage: "�ͻ�����", Intent: IntentSuggestion{Title: "�ͻ���ֵ����", Description: "�����ͻ�������Ϊ��ʶ��߼�ֵ�ͻ�", Icon: "users", Query: "�밴�ͻ�ͳ�ƹ���������ܽ��"}},
{Domain: DomainSales, Language: LanguageEnglish, UserMessage: "analyze sales", Intent: IntentSuggestion{Title: "Monthly Trend", Description: "Analyze monthly sales trends", Icon: "chart_line", Query: "Summarize sales by month, create trend chart"}},
{Domain: DomainSales, Language: LanguageEnglish, UserMessage: "product performance", Intent: IntentSuggestion{Title: "Product Ranking", Description: "Analyze product sales volume", Icon: "trophy", Query: "Calculate sales by product, rank by revenue"}},
{Domain: DomainSales, Language: LanguageEnglish, UserMessage: "customer analysis", Intent: IntentSuggestion{Title: "Customer Value", Description: "Analyze customer purchase behavior", Icon: "users", Query: "Calculate purchase frequency by customer"}},
}
e.examples[DomainFinance] = []FewShotExample{
{Domain: DomainFinance, Language: LanguageChinese, UserMessage: "��������״��", Intent: IntentSuggestion{Title: "��֧����", Description: "���������֧���Ĺ���", Icon: "dollar", Query: "�밴�����������֧��"}},
{Domain: DomainFinance, Language: LanguageChinese, UserMessage: "�����ɱ����", Intent: IntentSuggestion{Title: "�ɱ��ṹ����", Description: "��������ɱ��Ĺ���", Icon: "chart_bar", Query: "�밴�ɱ����ͻ��ܽ��"}},
{Domain: DomainFinance, Language: LanguageChinese, UserMessage: "Ԥ��ִ�з���", Intent: IntentSuggestion{Title: "Ԥ��Աȷ���", Description: "�Ա�ʵ��֧����Ԥ��", Icon: "clipboard", Query: "��Աȸ�����ʵ��֧����Ԥ����"}},
{Domain: DomainFinance, Language: LanguageEnglish, UserMessage: "financial status", Intent: IntentSuggestion{Title: "Income Analysis", Description: "Analyze income and expense", Icon: "dollar", Query: "Summarize income and expenses by category"}},
{Domain: DomainFinance, Language: LanguageEnglish, UserMessage: "cost breakdown", Intent: IntentSuggestion{Title: "Cost Structure", Description: "Analyze cost composition", Icon: "chart_bar", Query: "Summarize costs by type"}},
{Domain: DomainFinance, Language: LanguageEnglish, UserMessage: "budget analysis", Intent: IntentSuggestion{Title: "Budget Comparison", Description: "Compare actual vs budget", Icon: "clipboard", Query: "Compare actual vs budget by department"}},
}
e.examples[DomainUserBehavior] = []FewShotExample{
{Domain: DomainUserBehavior, Language: LanguageChinese, UserMessage: "�����û���Ϊ", Intent: IntentSuggestion{Title: "�û���Ծ�ȷ���", Description: "�����û��Ļ�Ծ�̶�", Icon: "mobile", Query: "��ͳ���ջ�Ծ�û���(DAU)"}},
{Domain: DomainUserBehavior, Language: LanguageChinese, UserMessage: "����ת�����", Intent: IntentSuggestion{Title: "ת��©������", Description: "�����û�ת��������", Icon: "funnel", Query: "�빹���û�ת��©��"}},
{Domain: DomainUserBehavior, Language: LanguageChinese, UserMessage: "�û��������", Intent: IntentSuggestion{Title: "�����ʷ���", Description: "�����û����������", Icon: "chart_bar", Query: "�����������桢7��������"}},
{Domain: DomainUserBehavior, Language: LanguageEnglish, UserMessage: "user behavior", Intent: IntentSuggestion{Title: "User Activity", Description: "Analyze user activity levels", Icon: "mobile", Query: "Calculate DAU and WAU"}},
{Domain: DomainUserBehavior, Language: LanguageEnglish, UserMessage: "conversion rates", Intent: IntentSuggestion{Title: "Conversion Funnel", Description: "Analyze user conversion", Icon: "funnel", Query: "Build conversion funnel"}},
{Domain: DomainUserBehavior, Language: LanguageEnglish, UserMessage: "retention analysis", Intent: IntentSuggestion{Title: "Retention Rate", Description: "Analyze user retention", Icon: "chart_bar", Query: "Calculate D1, D7 retention rates"}},
}
e.examples[DomainGeneral] = []FewShotExample{
{Domain: DomainGeneral, Language: LanguageChinese, UserMessage: "��������", Intent: IntentSuggestion{Title: "���ݸ���", Description: "�����ݽ����������", Icon: "chart_bar", Query: "��չʾ���ݵĻ���ͳ����Ϣ"}},
{Domain: DomainGeneral, Language: LanguageChinese, UserMessage: "��������", Intent: IntentSuggestion{Title: "ʱ�����Ʒ���", Description: "����������ʱ��ı仯", Icon: "chart_line", Query: "�밴ʱ��ά�Ȼ�������"}},
{Domain: DomainGeneral, Language: LanguageChinese, UserMessage: "�Աȷ���", Intent: IntentSuggestion{Title: "����Աȷ���", Description: "����ͬά�ȷ���Ա�", Icon: "balance", Query: "�밴��Ҫ����ά�ȷ���"}},
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
builder.WriteString("\n## �ο�ʾ��\n������һЩ��ͼ�����ʾ������ο���Щʾ���ĸ�ʽ�ͷ��\n\n")
} else {
builder.WriteString("\n## Reference Examples\nHere are some examples of intent understanding. Please follow the format and style:\n\n")
}
for i, example := range examples {
if normalizedLang == LanguageChinese {
builder.WriteString("### ʾ�� ")
builder.WriteString(intToString(i + 1))
builder.WriteString("\n**�û�����**: ")
builder.WriteString(example.UserMessage)
builder.WriteString("\n**��ͼ����**:\n- ����: ")
builder.WriteString(example.Intent.Title)
builder.WriteString("\n- ����: ")
builder.WriteString(example.Intent.Description)
builder.WriteString("\n- ͼ��: ")
builder.WriteString(example.Intent.Icon)
builder.WriteString("\n- ��ѯ: ")
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
