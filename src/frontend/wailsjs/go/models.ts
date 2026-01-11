export namespace agent {
	
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
	    // Go type: time
	    created_at: any;
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
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class ChatMessage {
	    id: string;
	    role: string;
	    content: string;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class ChatThread {
	    id: string;
	    title: string;
	    data_source_id: string;
	    created_at: number;
	    messages: ChatMessage[];
	
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
	

}

