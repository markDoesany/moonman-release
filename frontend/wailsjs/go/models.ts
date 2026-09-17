export namespace models {
	
	export class BuildResult {
	    projectId: string;
	    projectName: string;
	    componentId: string;
	    componentName: string;
	    success: boolean;
	    status: string;
	    exitCode: number;
	    outputDirectory: string;
	    outputPath: string;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime: any;
	    durationMs: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.success = source["success"];
	        this.status = source["status"];
	        this.exitCode = source["exitCode"];
	        this.outputDirectory = source["outputDirectory"];
	        this.outputPath = source["outputPath"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.durationMs = source["durationMs"];
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
	export class BuildComponentState {
	    componentId: string;
	    componentName: string;
	    selected: boolean;
	    status: string;
	    message: string;
	    result?: BuildResult;
	
	    static createFrom(source: any = {}) {
	        return new BuildComponentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.selected = source["selected"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.result = this.convertValues(source["result"], BuildResult);
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
	export class BuildRequest {
	    projectId: string;
	    componentIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new BuildRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.componentIds = source["componentIds"];
	    }
	}
	
	export class BuildRun {
	    id: string;
	    projectId: string;
	    projectName: string;
	    status: string;
	    components: BuildComponentState[];
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.status = source["status"];
	        this.components = this.convertValues(source["components"], BuildComponentState);
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
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
	export class Component {
	    id: string;
	    name: string;
	    path: string;
	    buildCommand: string;
	    outputDirectory: string;
	    packageEnabled: boolean;
	    packageFilename: string;
	
	    static createFrom(source: any = {}) {
	        return new Component(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.buildCommand = source["buildCommand"];
	        this.outputDirectory = source["outputDirectory"];
	        this.packageEnabled = source["packageEnabled"];
	        this.packageFilename = source["packageFilename"];
	    }
	}
	export class Project {
	    id: string;
	    name: string;
	    components: Component[];
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.components = this.convertValues(source["components"], Component);
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
	export class ValidationIssue {
	    field: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.message = source["message"];
	    }
	}

}

