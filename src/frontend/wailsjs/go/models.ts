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

}

