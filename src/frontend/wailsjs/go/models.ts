export namespace agent {
	
	export class ConfirmedFinding {
	    finding_id: string;
	    content: string;
	    confirmed_by: string;
	    importance: number;
	    timestamp: number;
	    related_steps: string[];
	    tags?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ConfirmedFinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.finding_id = source["finding_id"];
	        this.content = source["content"];
	        this.confirmed_by = source["confirmed_by"];
	        this.importance = source["importance"];
	        this.timestamp = source["timestamp"];
	        this.related_steps = source["related_steps"];
	        this.tags = source["tags"];
	    }
	}
	export class Evidence {
	    type: string;
	    description: string;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new Evidence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.description = source["description"];
	        this.data = source["data"];
	    }
	}
	export class PathStep {
	    step_id: string;
	    timestamp: number;
	    phenomenon: string;
	    action: string;
	    conclusion: string;
	    evidence: Evidence[];
	    user_query: string;
	    ai_response: string;
	
	    static createFrom(source: any = {}) {
	        return new PathStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step_id = source["step_id"];
	        this.timestamp = source["timestamp"];
	        this.phenomenon = source["phenomenon"];
	        this.action = source["action"];
	        this.conclusion = source["conclusion"];
	        this.evidence = this.convertValues(source["evidence"], Evidence);
	        this.user_query = source["user_query"];
	        this.ai_response = source["ai_response"];
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
	export class AnalysisPath {
	    session_id: string;
	    created_at: number;
	    updated_at: number;
	    steps: PathStep[];
	    findings: ConfirmedFinding[];
	
	    static createFrom(source: any = {}) {
	        return new AnalysisPath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.steps = this.convertValues(source["steps"], PathStep);
	        this.findings = this.convertValues(source["findings"], ConfirmedFinding);
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
	export class AnalysisRecord {
	    id: string;
	    data_source_id: string;
	    analysis_type: string;
	    target_columns: string[];
	    key_findings: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.data_source_id = source["data_source_id"];
	        this.analysis_type = source["analysis_type"];
	        this.target_columns = source["target_columns"];
	        this.key_findings = source["key_findings"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	    optimized: boolean;
	    mysql_export_config?: MySQLExportConfig;
	    shopify_store?: string;
	    shopify_access_token?: string;
	    shopify_api_version?: string;
	    bigcommerce_store_hash?: string;
	    bigcommerce_access_token?: string;
	    ebay_access_token?: string;
	    ebay_environment?: string;
	    ebay_api_fulfillment?: boolean;
	    ebay_api_finances?: boolean;
	    ebay_api_analytics?: boolean;
	    etsy_shop_id?: string;
	    etsy_access_token?: string;
	
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
	        this.optimized = source["optimized"];
	        this.mysql_export_config = this.convertValues(source["mysql_export_config"], MySQLExportConfig);
	        this.shopify_store = source["shopify_store"];
	        this.shopify_access_token = source["shopify_access_token"];
	        this.shopify_api_version = source["shopify_api_version"];
	        this.bigcommerce_store_hash = source["bigcommerce_store_hash"];
	        this.bigcommerce_access_token = source["bigcommerce_access_token"];
	        this.ebay_access_token = source["ebay_access_token"];
	        this.ebay_environment = source["ebay_environment"];
	        this.ebay_api_fulfillment = source["ebay_api_fulfillment"];
	        this.ebay_api_finances = source["ebay_api_finances"];
	        this.ebay_api_analytics = source["ebay_api_analytics"];
	        this.etsy_shop_id = source["etsy_shop_id"];
	        this.etsy_access_token = source["etsy_access_token"];
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
	
	
	export class DataSourceSummary {
	    id: string;
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DataSourceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class DataSourceStatistics {
	    total_count: number;
	    breakdown_by_type: Record<string, number>;
	    data_sources: DataSourceSummary[];
	
	    static createFrom(source: any = {}) {
	        return new DataSourceStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_count = source["total_count"];
	        this.breakdown_by_type = source["breakdown_by_type"];
	        this.data_sources = this.convertValues(source["data_sources"], DataSourceSummary);
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
	
	export class ShopifyOAuthConfig {
	    client_id: string;
	    client_secret: string;
	    scopes: string;
	
	    static createFrom(source: any = {}) {
	        return new ShopifyOAuthConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.client_id = source["client_id"];
	        this.client_secret = source["client_secret"];
	        this.scopes = source["scopes"];
	    }
	}
	export class Skill {
	    name: string;
	    description: string;
	    content: string;
	    path: string;
	    // Go type: time
	    installed_at: any;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.content = source["content"];
	        this.path = source["path"];
	        this.installed_at = this.convertValues(source["installed_at"], null);
	        this.enabled = source["enabled"];
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
	
	export class StoreCredentials {
	    platform: string;
	    client_id: string;
	    client_secret: string;
	    api_key?: string;
	    api_secret?: string;
	    scopes?: string;
	    redirect_uri?: string;
	    enabled: boolean;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new StoreCredentials(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.client_id = source["client_id"];
	        this.client_secret = source["client_secret"];
	        this.api_key = source["api_key"];
	        this.api_secret = source["api_secret"];
	        this.scopes = source["scopes"];
	        this.redirect_uri = source["redirect_uri"];
	        this.enabled = source["enabled"];
	        this.description = source["description"];
	    }
	}
	

}

export namespace config {
	
	export class LocationConfig {
	    country: string;
	    city: string;
	    latitude?: number;
	    longitude?: number;
	
	    static createFrom(source: any = {}) {
	        return new LocationConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.country = source["country"];
	        this.city = source["city"];
	        this.latitude = source["latitude"];
	        this.longitude = source["longitude"];
	    }
	}
	export class IntentEnhancementConfig {
	    enable_context_enhancement: boolean;
	    enable_preference_learning: boolean;
	    enable_dynamic_dimensions: boolean;
	    enable_few_shot_examples: boolean;
	    enable_caching: boolean;
	    cache_similarity_threshold: number;
	    cache_expiration_hours: number;
	    max_cache_entries: number;
	    max_history_records: number;
	
	    static createFrom(source: any = {}) {
	        return new IntentEnhancementConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enable_context_enhancement = source["enable_context_enhancement"];
	        this.enable_preference_learning = source["enable_preference_learning"];
	        this.enable_dynamic_dimensions = source["enable_dynamic_dimensions"];
	        this.enable_few_shot_examples = source["enable_few_shot_examples"];
	        this.enable_caching = source["enable_caching"];
	        this.cache_similarity_threshold = source["cache_similarity_threshold"];
	        this.cache_expiration_hours = source["cache_expiration_hours"];
	        this.max_cache_entries = source["max_cache_entries"];
	        this.max_history_records = source["max_history_records"];
	    }
	}
	export class UAPIConfig {
	    enabled: boolean;
	    apiToken: string;
	    baseUrl?: string;
	    tested: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UAPIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.apiToken = source["apiToken"];
	        this.baseUrl = source["baseUrl"];
	        this.tested = source["tested"];
	    }
	}
	export class ProxyConfig {
	    enabled: boolean;
	    protocol: string;
	    host: string;
	    port: number;
	    username?: string;
	    password?: string;
	    tested: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.tested = source["tested"];
	    }
	}
	export class SearchAPIConfig {
	    id: string;
	    name: string;
	    description: string;
	    apiKey?: string;
	    customId?: string;
	    enabled: boolean;
	    tested: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchAPIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.apiKey = source["apiKey"];
	        this.customId = source["customId"];
	        this.enabled = source["enabled"];
	        this.tested = source["tested"];
	    }
	}
	export class SearchEngine {
	    id: string;
	    name: string;
	    url: string;
	    enabled: boolean;
	    tested: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchEngine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.enabled = source["enabled"];
	        this.tested = source["tested"];
	    }
	}
	export class MCPService {
	    id: string;
	    name: string;
	    description: string;
	    url: string;
	    enabled: boolean;
	    tested: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.url = source["url"];
	        this.enabled = source["enabled"];
	        this.tested = source["tested"];
	    }
	}
	export class Config {
	    llmProvider: string;
	    apiKey: string;
	    baseUrl: string;
	    modelName: string;
	    maxTokens: number;
	    darkMode: boolean;
	    enableMemory: boolean;
	    autoAnalysisSuggestions: boolean;
	    localCache: boolean;
	    language: string;
	    claudeHeaderStyle: string;
	    dataCacheDir: string;
	    pythonPath: string;
	    maxPreviewRows: number;
	    maxConcurrentAnalysis: number;
	    detailedLog: boolean;
	    logMaxSizeMB: number;
	    autoIntentUnderstanding: boolean;
	    mcpServices: MCPService[];
	    searchEngines?: SearchEngine[];
	    searchAPIs: SearchAPIConfig[];
	    activeSearchEngine?: string;
	    activeSearchAPI?: string;
	    proxyConfig?: ProxyConfig;
	    uapiConfig?: UAPIConfig;
	    webSearchProvider?: string;
	    webSearchAPIKey?: string;
	    webSearchMCPURL?: string;
	    intentEnhancement?: IntentEnhancementConfig;
	    location?: LocationConfig;
	    shopifyClientId?: string;
	    shopifyClientSecret?: string;
	
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
	        this.enableMemory = source["enableMemory"];
	        this.autoAnalysisSuggestions = source["autoAnalysisSuggestions"];
	        this.localCache = source["localCache"];
	        this.language = source["language"];
	        this.claudeHeaderStyle = source["claudeHeaderStyle"];
	        this.dataCacheDir = source["dataCacheDir"];
	        this.pythonPath = source["pythonPath"];
	        this.maxPreviewRows = source["maxPreviewRows"];
	        this.maxConcurrentAnalysis = source["maxConcurrentAnalysis"];
	        this.detailedLog = source["detailedLog"];
	        this.logMaxSizeMB = source["logMaxSizeMB"];
	        this.autoIntentUnderstanding = source["autoIntentUnderstanding"];
	        this.mcpServices = this.convertValues(source["mcpServices"], MCPService);
	        this.searchEngines = this.convertValues(source["searchEngines"], SearchEngine);
	        this.searchAPIs = this.convertValues(source["searchAPIs"], SearchAPIConfig);
	        this.activeSearchEngine = source["activeSearchEngine"];
	        this.activeSearchAPI = source["activeSearchAPI"];
	        this.proxyConfig = this.convertValues(source["proxyConfig"], ProxyConfig);
	        this.uapiConfig = this.convertValues(source["uapiConfig"], UAPIConfig);
	        this.webSearchProvider = source["webSearchProvider"];
	        this.webSearchAPIKey = source["webSearchAPIKey"];
	        this.webSearchMCPURL = source["webSearchMCPURL"];
	        this.intentEnhancement = this.convertValues(source["intentEnhancement"], IntentEnhancementConfig);
	        this.location = this.convertValues(source["location"], LocationConfig);
	        this.shopifyClientId = source["shopifyClientId"];
	        this.shopifyClientSecret = source["shopifyClientSecret"];
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

export namespace database {
	
	export class LayoutItem {
	    i: string;
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	    minW?: number;
	    minH?: number;
	    maxW?: number;
	    maxH?: number;
	    static: boolean;
	    type: string;
	    instanceIdx: number;
	
	    static createFrom(source: any = {}) {
	        return new LayoutItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.i = source["i"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	        this.minW = source["minW"];
	        this.minH = source["minH"];
	        this.maxW = source["maxW"];
	        this.maxH = source["maxH"];
	        this.static = source["static"];
	        this.type = source["type"];
	        this.instanceIdx = source["instanceIdx"];
	    }
	}
	export class LayoutConfiguration {
	    id: string;
	    userId: string;
	    isLocked: boolean;
	    items: LayoutItem[];
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new LayoutConfiguration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userId = source["userId"];
	        this.isLocked = source["isLocked"];
	        this.items = this.convertValues(source["items"], LayoutItem);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class ExportRequest {
	    layoutConfig: LayoutConfiguration;
	    format: string;
	    userId: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.layoutConfig = this.convertValues(source["layoutConfig"], LayoutConfiguration);
	        this.format = source["format"];
	        this.userId = source["userId"];
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
	export class ExportResult {
	    filePath: string;
	    includedComponents: string[];
	    excludedComponents: string[];
	    totalComponents: number;
	    exportedAt: string;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.includedComponents = source["includedComponents"];
	        this.excludedComponents = source["excludedComponents"];
	        this.totalComponents = source["totalComponents"];
	        this.exportedAt = source["exportedAt"];
	        this.format = source["format"];
	    }
	}
	export class FileInfo {
	    id: string;
	    name: string;
	    size: number;
	    createdAt: number;
	    category: string;
	    downloadUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.createdAt = source["createdAt"];
	        this.category = source["category"];
	        this.downloadUrl = source["downloadUrl"];
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
	export class OriginalResults {
	    summary: string;
	    visualizations?: any[];
	
	    static createFrom(source: any = {}) {
	        return new OriginalResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.visualizations = source["visualizations"];
	    }
	}
	export class ProducesInfo {
	    type: string;
	    filename?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProducesInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.filename = source["filename"];
	    }
	}
	export class ResultSchema {
	    columns: string[];
	    types: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResultSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.types = source["types"];
	    }
	}
	export class ExecutableStep {
	    step_id: number;
	    step_type: string;
	    description: string;
	    sql?: string;
	    code?: string;
	    depends_on?: number[];
	    expected_result_schema?: ResultSchema;
	    produces?: ProducesInfo;
	
	    static createFrom(source: any = {}) {
	        return new ExecutableStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step_id = source["step_id"];
	        this.step_type = source["step_type"];
	        this.description = source["description"];
	        this.sql = source["sql"];
	        this.code = source["code"];
	        this.depends_on = source["depends_on"];
	        this.expected_result_schema = this.convertValues(source["expected_result_schema"], ResultSchema);
	        this.produces = this.convertValues(source["produces"], ProducesInfo);
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
	export class TableRequirement {
	    name: string;
	    columns: string[];
	    types?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new TableRequirement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.columns = source["columns"];
	        this.types = source["types"];
	    }
	}
	export class SchemaRequirements {
	    tables: TableRequirement[];
	
	    static createFrom(source: any = {}) {
	        return new SchemaRequirements(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tables = this.convertValues(source["tables"], TableRequirement);
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
	export class DataSourceInfo {
	    id: string;
	    name: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DataSourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class AnalysisExport {
	    file_type: string;
	    format_version: string;
	    description: string;
	    exported_at: string;
	    data_source: DataSourceInfo;
	    schema_requirements: SchemaRequirements;
	    executable_steps: ExecutableStep[];
	    original_results?: OriginalResults;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisExport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_type = source["file_type"];
	        this.format_version = source["format_version"];
	        this.description = source["description"];
	        this.exported_at = source["exported_at"];
	        this.data_source = this.convertValues(source["data_source"], DataSourceInfo);
	        this.schema_requirements = this.convertValues(source["schema_requirements"], SchemaRequirements);
	        this.executable_steps = this.convertValues(source["executable_steps"], ExecutableStep);
	        this.original_results = this.convertValues(source["original_results"], OriginalResults);
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
	export class AnalysisResultItem {
	    id: string;
	    type: string;
	    data: any;
	    metadata: Record<string, any>;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisResultItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.data = source["data"];
	        this.metadata = source["metadata"];
	        this.source = source["source"];
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
	    timing_data?: Record<string, any>;
	    analysis_results?: AnalysisResultItem[];
	
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
	        this.timing_data = source["timing_data"];
	        this.analysis_results = this.convertValues(source["analysis_results"], AnalysisResultItem);
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
	    message_id?: string;
	
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
	        this.message_id = source["message_id"];
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
	export class TableColumn {
	    title: string;
	    dataType: string;
	
	    static createFrom(source: any = {}) {
	        return new TableColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.dataType = source["dataType"];
	    }
	}
	export class TableData {
	    columns: TableColumn[];
	    data: any[][];
	
	    static createFrom(source: any = {}) {
	        return new TableData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], TableColumn);
	        this.data = source["data"];
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
	export class DashboardMetric {
	    title: string;
	    value: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardMetric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.value = source["value"];
	        this.change = source["change"];
	    }
	}
	export class DashboardExportData {
	    userRequest: string;
	    metrics: DashboardMetric[];
	    insights: string[];
	    chartImage: string;
	    chartImages: string[];
	    tableData?: TableData;
	
	    static createFrom(source: any = {}) {
	        return new DashboardExportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userRequest = source["userRequest"];
	        this.metrics = this.convertValues(source["metrics"], DashboardMetric);
	        this.insights = source["insights"];
	        this.chartImage = source["chartImage"];
	        this.chartImages = source["chartImages"];
	        this.tableData = this.convertValues(source["tableData"], TableData);
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
	
	
	export class IndexSuggestion {
	    table_name: string;
	    index_name: string;
	    columns: string[];
	    reason: string;
	    sql_command: string;
	    applied: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexSuggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table_name = source["table_name"];
	        this.index_name = source["index_name"];
	        this.columns = source["columns"];
	        this.reason = source["reason"];
	        this.sql_command = source["sql_command"];
	        this.applied = source["applied"];
	        this.error = source["error"];
	    }
	}
	
	export class IntentSuggestion {
	    id: string;
	    title: string;
	    description: string;
	    icon: string;
	    query: string;
	
	    static createFrom(source: any = {}) {
	        return new IntentSuggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	        this.query = source["query"];
	    }
	}
	export class LogStats {
	    totalSizeMB: number;
	    logCount: number;
	    archiveCount: number;
	    logDir: string;
	
	    static createFrom(source: any = {}) {
	        return new LogStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalSizeMB = source["totalSizeMB"];
	        this.logCount = source["logCount"];
	        this.archiveCount = source["archiveCount"];
	        this.logDir = source["logDir"];
	    }
	}
	
	export class OptimizeDataSourceResult {
	    data_source_id: string;
	    data_source_name: string;
	    suggestions: IndexSuggestion[];
	    summary: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new OptimizeDataSourceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data_source_id = source["data_source_id"];
	        this.data_source_name = source["data_source_name"];
	        this.suggestions = this.convertValues(source["suggestions"], IndexSuggestion);
	        this.summary = source["summary"];
	        this.success = source["success"];
	        this.error = source["error"];
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
	export class OptimizeSuggestionsResult {
	    data_source_id: string;
	    data_source_name: string;
	    suggestions: IndexSuggestion[];
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new OptimizeSuggestionsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data_source_id = source["data_source_id"];
	        this.data_source_name = source["data_source_name"];
	        this.suggestions = this.convertValues(source["suggestions"], IndexSuggestion);
	        this.success = source["success"];
	        this.error = source["error"];
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
	
	
	
	
	
	export class ShopifyOAuthConfig {
	    client_id: string;
	    client_secret: string;
	
	    static createFrom(source: any = {}) {
	        return new ShopifyOAuthConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.client_id = source["client_id"];
	        this.client_secret = source["client_secret"];
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
	
	
	
	export class ValidationIssue {
	    type: string;
	    table?: string;
	    column?: string;
	    message: string;
	    severity: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.table = source["table"];
	        this.column = source["column"];
	        this.message = source["message"];
	        this.severity = source["severity"];
	    }
	}
	export class ValidationResult {
	    compatible: boolean;
	    issues: ValidationIssue[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.compatible = source["compatible"];
	        this.issues = this.convertValues(source["issues"], ValidationIssue);
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

