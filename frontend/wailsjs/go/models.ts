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
	export class PackageConfig {
	    enabled: boolean;
	    filename: string;
	
	    static createFrom(source: any = {}) {
	        return new PackageConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.filename = source["filename"];
	    }
	}
	export class Component {
	    id: string;
	    name: string;
	    path: string;
	    buildCommand: string;
	    outputDirectory: string;
	    package: PackageConfig;
	
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
	        this.package = this.convertValues(source["package"], PackageConfig);
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
	export class PackageResult {
	    projectId: string;
	    projectName: string;
	    componentId: string;
	    componentName: string;
	    success: boolean;
	    status: string;
	    sourcePath: string;
	    packagePath: string;
	    sizeBytes: number;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime: any;
	    durationMs: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PackageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.success = source["success"];
	        this.status = source["status"];
	        this.sourcePath = source["sourcePath"];
	        this.packagePath = source["packagePath"];
	        this.sizeBytes = source["sizeBytes"];
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
	export class PackageComponentState {
	    componentId: string;
	    componentName: string;
	    selected: boolean;
	    status: string;
	    message: string;
	    result?: PackageResult;
	
	    static createFrom(source: any = {}) {
	        return new PackageComponentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.selected = source["selected"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.result = this.convertValues(source["result"], PackageResult);
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
	
	export class PackagePlanItem {
	    componentId: string;
	    componentName: string;
	    selected: boolean;
	    enabled: boolean;
	    sourcePath: string;
	    packagePath: string;
	    existing: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PackagePlanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.selected = source["selected"];
	        this.enabled = source["enabled"];
	        this.sourcePath = source["sourcePath"];
	        this.packagePath = source["packagePath"];
	        this.existing = source["existing"];
	        this.error = source["error"];
	    }
	}
	export class PackagePlan {
	    projectId: string;
	    projectName: string;
	    version: string;
	    releaseDirectory: string;
	    components: PackagePlanItem[];
	    hasConflicts: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PackagePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.version = source["version"];
	        this.releaseDirectory = source["releaseDirectory"];
	        this.components = this.convertValues(source["components"], PackagePlanItem);
	        this.hasConflicts = source["hasConflicts"];
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
	
	export class PackageRequest {
	    projectId: string;
	    componentIds: string[];
	    version: string;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PackageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.componentIds = source["componentIds"];
	        this.version = source["version"];
	        this.overwrite = source["overwrite"];
	    }
	}
	
	export class PackageRun {
	    id: string;
	    projectId: string;
	    projectName: string;
	    version: string;
	    status: string;
	    components: PackageComponentState[];
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PackageRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.version = source["version"];
	        this.status = source["status"];
	        this.components = this.convertValues(source["components"], PackageComponentState);
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
	export class ReleaseComponentState {
	    componentId: string;
	    componentName: string;
	    selected: boolean;
	    buildStatus: string;
	    buildMessage: string;
	    buildResult?: BuildResult;
	    packageStatus: string;
	    packageMessage: string;
	    packageResult?: PackageResult;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseComponentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.selected = source["selected"];
	        this.buildStatus = source["buildStatus"];
	        this.buildMessage = source["buildMessage"];
	        this.buildResult = this.convertValues(source["buildResult"], BuildResult);
	        this.packageStatus = source["packageStatus"];
	        this.packageMessage = source["packageMessage"];
	        this.packageResult = this.convertValues(source["packageResult"], PackageResult);
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
	export class ReleaseRun {
	    id: string;
	    projectId: string;
	    projectName: string;
	    version: string;
	    status: string;
	    components: ReleaseComponentState[];
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.version = source["version"];
	        this.status = source["status"];
	        this.components = this.convertValues(source["components"], ReleaseComponentState);
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

