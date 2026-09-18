export type Component = {
  id: string;
  name: string;
  path: string;
  buildCommand: string;
  outputDirectory: string;
  package: PackageConfig;
};

export type Project = {
  id: string;
  name: string;
  components: Component[];
};

export type ValidationIssue = {
  field: string;
  message: string;
};

export type BuildStatus = 'ready' | 'building' | 'success' | 'failed' | 'skipped' | 'cancelled';
export type BuildRunStatus = 'running' | 'completed' | 'failed' | 'cancelled';

export type BuildRequest = {
  projectId: string;
  componentIds: string[];
};

export type BuildResult = {
  projectId: string;
  projectName: string;
  componentId: string;
  componentName: string;
  success: boolean;
  status: BuildStatus;
  exitCode: number;
  outputDirectory: string;
  outputPath: string;
  startTime: string;
  endTime: string;
  durationMs: number;
  error?: string;
};

export type BuildComponentState = {
  componentId: string;
  componentName: string;
  selected: boolean;
  status: BuildStatus;
  message: string;
  result?: BuildResult;
};

export type BuildRun = {
  id: string;
  projectId: string;
  projectName: string;
  status: BuildRunStatus;
  components: BuildComponentState[];
  startTime: string;
  endTime?: string;
  error?: string;
};

export type BuildEvent = {
  type: string;
  phase?: 'build' | 'package' | 'transfer' | 'release';
  runId: string;
  projectId: string;
  projectName: string;
  componentId?: string;
  componentName?: string;
  status?: BuildStatus;
  runStatus?: BuildRunStatus;
  stream?: 'system' | 'stdout' | 'stderr';
  text?: string;
  timestamp: string;
  version?: string;
  result?: BuildResult;
  results?: BuildResult[];
  packageStatus?: PackageStatus;
  packageResult?: PackageResult;
  packageResults?: PackageResult[];
  transferStatus?: TransferStatus;
  transferResult?: TransferResult;
  transferResults?: TransferResult[];
  error?: string;
};

export type BuildOutputLine = {
  componentName: string;
  stream: 'system' | 'stdout' | 'stderr';
  text: string;
};

export type PackageConfig = {
  enabled: boolean;
  filename: string;
};

export type DaliConfig = {
  executable: string;
  peerName: string;
  peerAddress: string;
  auto: boolean;
  wait: boolean;
};

export type TransferStatus = 'ready' | 'sending' | 'success' | 'failed' | 'skipped' | 'cancelled';
export type TransferRunStatus = 'running' | 'completed' | 'failed' | 'cancelled';

export type TransferRequest = {
  projectId: string;
  componentIds: string[];
  version: string;
  filenameTemplate?: string;
  packageNames?: Record<string, string>;
};

export type TransferPlanItem = {
  componentId: string;
  componentName: string;
  selected: boolean;
  enabled: boolean;
  packagePath: string;
  filenameTemplate?: string;
  resolvedFilename?: string;
  exists: boolean;
  error?: string;
};

export type TransferPlan = {
  projectId: string;
  projectName: string;
  version: string;
  components: TransferPlanItem[];
  hasMissing: boolean;
  filenameTemplate: string;
};

export type TransferResult = {
  projectId: string;
  projectName: string;
  componentId: string;
  componentName: string;
  success: boolean;
  status: TransferStatus;
  packagePath: string;
  executable: string;
  arguments: string[];
  peerName?: string;
  peerAddress?: string;
  exitCode: number;
  startTime: string;
  endTime: string;
  durationMs: number;
  error?: string;
};

export type TransferComponentState = {
  componentId: string;
  componentName: string;
  selected: boolean;
  status: TransferStatus;
  message: string;
  result?: TransferResult;
};

export type TransferRun = {
  id: string;
  projectId: string;
  projectName: string;
  version: string;
  status: TransferRunStatus;
  components: TransferComponentState[];
  startTime: string;
  endTime?: string;
  error?: string;
};

export type PackageStatus = 'ready' | 'packaging' | 'success' | 'failed' | 'skipped' | 'cancelled';
export type PackageRunStatus = 'running' | 'completed' | 'failed' | 'cancelled';

export type PackageRequest = {
  projectId: string;
  componentIds: string[];
  version: string;
  overwrite: boolean;
  filenameTemplate?: string;
  packageNames?: Record<string, string>;
};

export type PackagePlanItem = {
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
};

export type PackagePlan = {
  projectId: string;
  projectName: string;
  version: string;
  releaseDirectory: string;
  components: PackagePlanItem[];
  hasConflicts: boolean;
  filenameTemplate: string;
};

export type PackageResult = {
  projectId: string;
  projectName: string;
  componentId: string;
  componentName: string;
  success: boolean;
  status: PackageStatus;
  sourcePath: string;
  packagePath: string;
  sizeBytes: number;
  startTime: string;
  endTime: string;
  durationMs: number;
  error?: string;
};

export type PackageComponentState = {
  componentId: string;
  componentName: string;
  selected: boolean;
  status: PackageStatus;
  message: string;
  result?: PackageResult;
};

export type PackageRun = {
  id: string;
  projectId: string;
  projectName: string;
  version: string;
  status: PackageRunStatus;
  components: PackageComponentState[];
  startTime: string;
  endTime?: string;
  error?: string;
};

export type ReleaseComponentState = {
  componentId: string;
  componentName: string;
  selected: boolean;
  buildStatus: BuildStatus;
  buildMessage: string;
  buildResult?: BuildResult;
  packageStatus: PackageStatus;
  packageMessage: string;
  packageResult?: PackageResult;
  transferStatus: TransferStatus;
  transferMessage: string;
  transferResult?: TransferResult;
};

export type ReleaseRun = {
  id: string;
  projectId: string;
  projectName: string;
  version: string;
  status: 'running' | 'completed' | 'failed' | 'cancelled';
  components: ReleaseComponentState[];
  startTime: string;
  endTime?: string;
  error?: string;
};
