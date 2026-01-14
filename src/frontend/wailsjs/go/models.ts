export namespace agent {
	
	export class ConversationTurn {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversationTurn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class AnalysisStep {
	    step_id: number;
	    timestamp: number;
	    tool_name: string;
	    description: string;
	    input: string;
	    output: string;
	    chart_type: string;
	    chart_data: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step_id = source["step_id"];
	        this.timestamp = source["timestamp"];
	        this.tool_name = source["tool_name"];
	        this.description = source["description"];
	        this.input = source["input"];
	        this.output = source["output"];
	        this.chart_type = source["chart_type"];
	        this.chart_data = source["chart_data"];
	    }
	}
	export class ReplayTableSchema {
	    table_name: string;
	    columns: string[];
	
	    static createFrom(source: any = {}) {
	        return new ReplayTableSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table_name = source["table_name"];
	        this.columns = source["columns"];
	    }
	}
	export class AnalysisRecording {
	    recording_id: string;
	    title: string;
	    description: string;
	    created_at: number;
	    source_id: string;
	    source_name: string;
	    source_schema: ReplayTableSchema[];
	    steps: AnalysisStep[];
	    llm_conversation: ConversationTurn[];
	
	    static createFrom(source: any = {}) {
	        return new AnalysisRecording(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recording_id = source["recording_id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.created_at = source["created_at"];
	        this.source_id = source["source_id"];
	        this.source_name = source["source_name"];
	        this.source_schema = this.convertValues(source["source_schema"], ReplayTableSchema);
	        this.steps = this.convertValues(source["steps"], AnalysisStep);
	        this.llm_conversation = this.convertValues(source["llm_conversation"], ConversationTurn);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class TableSchema {
	    table_name: string;
	    columns: string[];
	
	    static createFrom(source: any = {}) {
	        return new TableSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table_name = source["table_name"];
	        this.columns = source["columns"];
	    }
	}
	export class DataSourceAnalysis {
	    summary: string;
	    schema: TableSchema[];
	
	    static createFrom(source: any = {}) {
	        return new DataSourceAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.schema = this.convertValues(source["schema"], TableSchema);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MySQLExportConfig {
	    host?: string;
	    port?: string;
	    user?: string;
	    password?: string;
	    database?: string;
	
	    static createFrom(source: any = {}) {
	        return new MySQLExportConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.database = source["database"];
	    }
	}
	export class DataSourceConfig {
	    original_file?: string;
	    db_path: string;
	    table_name: string;
	    host?: string;
	    port?: string;
	    user?: string;
	    password?: string;
	    database?: string;
	    store_locally: boolean;
	    mysql_export_config?: MySQLExportConfig;
	
	    static createFrom(source: any = {}) {
	        return new DataSourceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original_file = source["original_file"];
	        this.db_path = source["db_path"];
	        this.table_name = source["table_name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.database = source["database"];
	        this.store_locally = source["store_locally"];
	        this.mysql_export_config = this.convertValues(source["mysql_export_config"], MySQLExportConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DataSource {
	    id: string;
	    name: string;
	    type: string;
	    created_at: number;
	    config: DataSourceConfig;
	    analysis?: DataSourceAnalysis;
	
	    static createFrom(source: any = {}) {
	        return new DataSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.created_at = source["created_at"];
	        this.config = this.convertValues(source["config"], DataSourceConfig);
	        this.analysis = this.convertValues(source["analysis"], DataSourceAnalysis);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class FieldMapping {
	    old_field: string;
	    new_field: string;
	
	    static createFrom(source: any = {}) {
	        return new FieldMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.old_field = source["old_field"];
	        this.new_field = source["new_field"];
	    }
	}
	
	export class PythonEnvironment {
	    path: string;
	    version: string;
	    type: string;
	    isRecommended: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PythonEnvironment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.version = source["version"];
	        this.type = source["type"];
	        this.isRecommended = source["isRecommended"];
	    }
	}
	export class PythonValidationResult {
	    valid: boolean;
	    version: string;
	    missingPackages: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new PythonValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.version = source["version"];
	        this.missingPackages = source["missingPackages"];
	        this.error = source["error"];
	    }
	}
	export class TableMapping {
	    source_table: string;
	    target_table: string;
	    mappings: FieldMapping[];
	
	    static createFrom(source: any = {}) {
	        return new TableMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_table = source["source_table"];
	        this.target_table = source["target_table"];
	        this.mappings = this.convertValues(source["mappings"], FieldMapping);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StepResult {
	    step_id: number;
	    success: boolean;
	    output: string;
	    error_message: string;
	    chart_data: string;
	    modified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StepResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step_id = source["step_id"];
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error_message = source["error_message"];
	        this.chart_data = source["chart_data"];
	        this.modified = source["modified"];
	    }
	}
	export class ReplayResult {
	    success: boolean;
	    steps_executed: number;
	    steps_failed: number;
	    step_results: StepResult[];
	    field_mappings: TableMapping[];
	    generated_files: string[];
	    error_message: string;
	    charts: any[];
	
	    static createFrom(source: any = {}) {
	        return new ReplayResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.steps_executed = source["steps_executed"];
	        this.steps_failed = source["steps_failed"];
	        this.step_results = this.convertValues(source["step_results"], StepResult);
	        this.field_mappings = this.convertValues(source["field_mappings"], TableMapping);
	        this.generated_files = source["generated_files"];
	        this.error_message = source["error_message"];
	        this.charts = source["charts"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	

}

export namespace config {
	
	export class Config {
	    llmProvider: string;
	    apiKey: string;
	    baseUrl: string;
	    modelName: string;
	    maxTokens: number;
	    darkMode: boolean;
	    localCache: boolean;
	    language: string;
	    claudeHeaderStyle: string;
	    dataCacheDir: string;
	    pythonPath: string;
	    maxPreviewRows: number;
	    detailedLog: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llmProvider = source["llmProvider"];
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.modelName = source["modelName"];
	        this.maxTokens = source["maxTokens"];
	        this.darkMode = source["darkMode"];
	        this.localCache = source["localCache"];
	        this.language = source["language"];
	        this.claudeHeaderStyle = source["claudeHeaderStyle"];
	        this.dataCacheDir = source["dataCacheDir"];
	        this.pythonPath = source["pythonPath"];
	        this.maxPreviewRows = source["maxPreviewRows"];
	        this.detailedLog = source["detailedLog"];
	    }
	}

}

export namespace main {
	
	export class AgentMemoryView {
	    long_term: string[];
	    medium_term: string[];
	    short_term: string[];
	
	    static createFrom(source: any = {}) {
	        return new AgentMemoryView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.long_term = source["long_term"];
	        this.medium_term = source["medium_term"];
	        this.short_term = source["short_term"];
	    }
	}
	export class ChartItem {
	    type: string;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new ChartItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.data = source["data"];
	    }
	}
	export class ChartData {
	    charts: ChartItem[];
	
	    static createFrom(source: any = {}) {
	        return new ChartData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.charts = this.convertValues(source["charts"], ChartItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ChatMessage {
	    id: string;
	    role: string;
	    content: string;
	    timestamp: number;
	    chart_data?: ChartData;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	        this.chart_data = this.convertValues(source["chart_data"], ChartData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionFile {
	    name: string;
	    path: string;
	    type: string;
	    size: number;
	    created_at: number;
	
	    static createFrom(source: any = {}) {
	        return new SessionFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.created_at = source["created_at"];
	    }
	}
	export class ChatThread {
	    id: string;
	    title: string;
	    data_source_id: string;
	    created_at: number;
	    messages: ChatMessage[];
	    files?: SessionFile[];
	
	    static createFrom(source: any = {}) {
	        return new ChatThread(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.data_source_id = source["data_source_id"];
	        this.created_at = source["created_at"];
	        this.messages = this.convertValues(source["messages"], ChatMessage);
	        this.files = this.convertValues(source["files"], SessionFile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConnectionResult {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class Insight {
	    text: string;
	    icon: string;
	    data_source_id?: string;
	    source_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new Insight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.icon = source["icon"];
	        this.data_source_id = source["data_source_id"];
	        this.source_name = source["source_name"];
	    }
	}
	export class Metric {
	    title: string;
	    value: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new Metric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.value = source["value"];
	        this.change = source["change"];
	    }
	}
	export class DashboardData {
	    metrics: Metric[];
	    insights: Insight[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.metrics = this.convertValues(source["metrics"], Metric);
	        this.insights = this.convertValues(source["insights"], Insight);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ErrorRecord {
	    timestamp: string;
	    error_type: string;
	    error_message: string;
	    context: string;
	    solution: string;
	    successful: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ErrorRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.error_type = source["error_type"];
	        this.error_message = source["error_message"];
	        this.context = source["context"];
	        this.solution = source["solution"];
	        this.successful = source["successful"];
	    }
	}
	export class ErrorKnowledgeSummary {
	    total_records: number;
	    successful_count: number;
	    success_rate: number;
	    by_type: Record<string, number>;
	    recent_errors: ErrorRecord[];
	
	    static createFrom(source: any = {}) {
	        return new ErrorKnowledgeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_records = source["total_records"];
	        this.successful_count = source["successful_count"];
	        this.success_rate = source["success_rate"];
	        this.by_type = source["by_type"];
	        this.recent_errors = this.convertValues(source["recent_errors"], ErrorRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	export class SkillInfo {
	    id: string;
	    name: string;
	    description: string;
	    version: string;
	    author: string;
	    category: string;
	    keywords: string[];
	    required_columns: string[];
	    tools: string[];
	    enabled: boolean;
	    icon: string;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new SkillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.category = source["category"];
	        this.keywords = source["keywords"];
	        this.required_columns = source["required_columns"];
	        this.tools = source["tools"];
	        this.enabled = source["enabled"];
	        this.icon = source["icon"];
	        this.tags = source["tags"];
	    }
	}

}

