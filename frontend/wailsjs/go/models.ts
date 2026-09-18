export namespace models {
	
	export class RunSummary {
	    runId: string;
	    projectId: string;
	    projectName: string;
	    environment?: string;
	    operation: string;
	    version?: string;
	    componentIds: string[];
	    componentNames?: string[];
	    packagePaths?: string[];
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    status: string;
	    errorSummary?: string;
	    filenameTemplate?: string;
	    approvedPackageNames?: Record<string, string>;
	    releaseDirectory?: string;
	    retryOfRunId?: string;
	    attempt?: number;
	    retryStage?: string;
	    command?: string;
	    daliCommands?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RunSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.environment = source["environment"];
	        this.operation = source["operation"];
	        this.version = source["version"];
	        this.componentIds = source["componentIds"];
	        this.componentNames = source["componentNames"];
	        this.packagePaths = source["packagePaths"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.endTime = this.convertValues(source["endTime"], null);
	        this.status = source["status"];
	        this.errorSummary = source["errorSummary"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.approvedPackageNames = source["approvedPackageNames"];
	        this.releaseDirectory = source["releaseDirectory"];
	        this.retryOfRunId = source["retryOfRunId"];
	        this.attempt = source["attempt"];
	        this.retryStage = source["retryStage"];
	        this.command = source["command"];
	        this.daliCommands = source["daliCommands"];
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
	export class ActivityPage {
	    runs: RunSummary[];
	    page: number;
	    pageSize: number;
	    total: number;
	    totalPages: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runs = this.convertValues(source["runs"], RunSummary);
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.total = source["total"];
	        this.totalPages = source["totalPages"];
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
	export class ActivityQuery {
	    projectId?: string;
	    environment?: string;
	    status?: string;
	    search?: string;
	    page?: number;
	    pageSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.environment = source["environment"];
	        this.status = source["status"];
	        this.search = source["search"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class BuildResult {
	    projectId: string;
	    projectName: string;
	    componentId: string;
	    componentName: string;
	    environment?: string;
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
	        this.environment = source["environment"];
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
	    environment?: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.componentIds = source["componentIds"];
	        this.environment = source["environment"];
	    }
	}
	
	export class BuildRun {
	    id: string;
	    projectId: string;
	    projectName: string;
	    environment?: string;
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
	        this.environment = source["environment"];
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
	    buildCommands?: Record<string, string>;
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
	        this.buildCommands = source["buildCommands"];
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
	export class DaliAvailability {
	    available: boolean;
	    configuredExecutable: string;
	    resolvedExecutable: string;
	    error?: string;
	    installCommand: string;
	    releaseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new DaliAvailability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.configuredExecutable = source["configuredExecutable"];
	        this.resolvedExecutable = source["resolvedExecutable"];
	        this.error = source["error"];
	        this.installCommand = source["installCommand"];
	        this.releaseUrl = source["releaseUrl"];
	    }
	}
	export class DaliConfig {
	    executable: string;
	    peerName: string;
	    peerAddress: string;
	    auto: boolean;
	    wait: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DaliConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.executable = source["executable"];
	        this.peerName = source["peerName"];
	        this.peerAddress = source["peerAddress"];
	        this.auto = source["auto"];
	        this.wait = source["wait"];
	    }
	}
	export class EnvironmentProfile {
	    id: string;
	    name: string;
	    commands: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.commands = source["commands"];
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
	    version: string;
	    filenameTemplate?: string;
	    resolvedFilename?: string;
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
	        this.version = source["version"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.resolvedFilename = source["resolvedFilename"];
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
	    filenameTemplate?: string;
	    resolvedFilename?: string;
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
	        this.filenameTemplate = source["filenameTemplate"];
	        this.resolvedFilename = source["resolvedFilename"];
	        this.existing = source["existing"];
	        this.error = source["error"];
	    }
	}
	export class PackagePlan {
	    projectId: string;
	    projectName: string;
	    version: string;
	    releaseDirectory: string;
	    filenameTemplate: string;
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
	        this.filenameTemplate = source["filenameTemplate"];
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
	    filenameTemplate?: string;
	    packageNames?: Record<string, string>;
	    environment?: string;
	    releaseDirectory?: string;
	
	    static createFrom(source: any = {}) {
	        return new PackageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.componentIds = source["componentIds"];
	        this.version = source["version"];
	        this.overwrite = source["overwrite"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.packageNames = source["packageNames"];
	        this.environment = source["environment"];
	        this.releaseDirectory = source["releaseDirectory"];
	    }
	}
	
	export class PackageRun {
	    id: string;
	    projectId: string;
	    projectName: string;
	    environment?: string;
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
	        this.environment = source["environment"];
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
	    defaultEnvironment?: string;
	    environments?: EnvironmentProfile[];
	    components: Component[];
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.defaultEnvironment = source["defaultEnvironment"];
	        this.environments = this.convertValues(source["environments"], EnvironmentProfile);
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
	export class TransferResult {
	    projectId: string;
	    projectName: string;
	    componentId: string;
	    componentName: string;
	    success: boolean;
	    status: string;
	    packagePath: string;
	    version: string;
	    filenameTemplate?: string;
	    resolvedFilename?: string;
	    executable: string;
	    arguments: string[];
	    command: string;
	    peerName?: string;
	    peerAddress?: string;
	    exitCode: number;
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime: any;
	    durationMs: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.success = source["success"];
	        this.status = source["status"];
	        this.packagePath = source["packagePath"];
	        this.version = source["version"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.resolvedFilename = source["resolvedFilename"];
	        this.executable = source["executable"];
	        this.arguments = source["arguments"];
	        this.command = source["command"];
	        this.peerName = source["peerName"];
	        this.peerAddress = source["peerAddress"];
	        this.exitCode = source["exitCode"];
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
	    transferStatus: string;
	    transferMessage: string;
	    transferResult?: TransferResult;
	
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
	        this.transferStatus = source["transferStatus"];
	        this.transferMessage = source["transferMessage"];
	        this.transferResult = this.convertValues(source["transferResult"], TransferResult);
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
	    environment?: string;
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
	        this.environment = source["environment"];
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
	export class RetryRequest {
	    runId: string;
	    stage: string;
	    componentIds?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RetryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.stage = source["stage"];
	        this.componentIds = source["componentIds"];
	    }
	}
	export class TransferComponentState {
	    componentId: string;
	    componentName: string;
	    selected: boolean;
	    status: string;
	    message: string;
	    result?: TransferResult;
	
	    static createFrom(source: any = {}) {
	        return new TransferComponentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.selected = source["selected"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.result = this.convertValues(source["result"], TransferResult);
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
	export class TransferRun {
	    id: string;
	    projectId: string;
	    projectName: string;
	    environment?: string;
	    version: string;
	    status: string;
	    components: TransferComponentState[];
	    // Go type: time
	    startTime: any;
	    // Go type: time
	    endTime?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.environment = source["environment"];
	        this.version = source["version"];
	        this.status = source["status"];
	        this.components = this.convertValues(source["components"], TransferComponentState);
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
	export class RetryRun {
	    operation: string;
	    stage: string;
	    attempt: number;
	    retryOfRunId: string;
	    build?: BuildRun;
	    package?: PackageRun;
	    release?: ReleaseRun;
	    transfer?: TransferRun;
	
	    static createFrom(source: any = {}) {
	        return new RetryRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.stage = source["stage"];
	        this.attempt = source["attempt"];
	        this.retryOfRunId = source["retryOfRunId"];
	        this.build = this.convertValues(source["build"], BuildRun);
	        this.package = this.convertValues(source["package"], PackageRun);
	        this.release = this.convertValues(source["release"], ReleaseRun);
	        this.transfer = this.convertValues(source["transfer"], TransferRun);
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
	
	
	export class TransferPlanItem {
	    componentId: string;
	    componentName: string;
	    selected: boolean;
	    enabled: boolean;
	    packagePath: string;
	    filenameTemplate?: string;
	    resolvedFilename?: string;
	    exists: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferPlanItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.componentId = source["componentId"];
	        this.componentName = source["componentName"];
	        this.selected = source["selected"];
	        this.enabled = source["enabled"];
	        this.packagePath = source["packagePath"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.resolvedFilename = source["resolvedFilename"];
	        this.exists = source["exists"];
	        this.error = source["error"];
	    }
	}
	export class TransferPlan {
	    projectId: string;
	    projectName: string;
	    version: string;
	    releaseDirectory: string;
	    filenameTemplate: string;
	    components: TransferPlanItem[];
	    hasMissing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TransferPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.projectName = source["projectName"];
	        this.version = source["version"];
	        this.releaseDirectory = source["releaseDirectory"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.components = this.convertValues(source["components"], TransferPlanItem);
	        this.hasMissing = source["hasMissing"];
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
	
	export class TransferRequest {
	    projectId: string;
	    componentIds: string[];
	    version: string;
	    filenameTemplate?: string;
	    packageNames?: Record<string, string>;
	    environment?: string;
	    releaseDirectory?: string;
	    packagePaths?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TransferRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.componentIds = source["componentIds"];
	        this.version = source["version"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.packageNames = source["packageNames"];
	        this.environment = source["environment"];
	        this.releaseDirectory = source["releaseDirectory"];
	        this.packagePaths = source["packagePaths"];
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

