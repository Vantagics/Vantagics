export namespace main {
	
	export class Config {
	    llmProvider: string;
	    apiKey: string;
	    baseUrl: string;
	    modelName: string;
	    maxTokens: number;
	    darkMode: boolean;
	    localCache: boolean;
	    language: string;
	
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
	    }
	}
	export class Insight {
	    text: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new Insight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.icon = source["icon"];
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

