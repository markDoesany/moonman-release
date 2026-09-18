export type Component = {
  id: string;
  name: string;
  path: string;
  buildCommand: string;
  buildCommands?: Record<string, string>;
  outputDirectory: string;
  package: PackageConfig;
};

export type EnvironmentProfile = {
  id: string;
  name: string;
  commands: Record<string, string>;
};

export type Project = {
  id: string;
  name: string;
  defaultEnvironment?: string;
  environments?: EnvironmentProfile[];
  components: Component[];
};

export type RunSummary = {
  runId: string;
  projectId: string;
  projectName: string;
  environment?: string;
  operation: string;
  version?: string;
  componentIds: string[];
  componentNames?: string[];
  packagePaths?: string[];
  startTime: string;
  endTime?: string;
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
};

export type ActivityQuery = { projectId?: string; environment?: string; status?: string; search?: string; page: number; pageSize: number };
export type ActivityPage = { runs: RunSummary[]; page: number; pageSize: number; total: number; totalPages: number };

export type RetryStage = 'build' | 'package' | 'transfer';
export type RetryRequest = { runId: string; stage: RetryStage; componentIds?: string[] };
export type RetryRunResult = {
  operation: OperationName;
  stage: RetryStage;
  attempt: number;
  retryOfRunId: string;
  build?: BuildRun;
  package?: PackageRun;
  release?: ReleaseRun;
  transfer?: TransferRun;
};
type OperationName = 'build' | 'package' | 'release' | 'release-transfer' | 'transfer';

export type ValidationIssue = {
  field: string;
  message: string;
};

export type BuildStatus = 'ready' | 'building' | 'success' | 'failed' | 'skipped' | 'cancelled';
export type BuildRunStatus = 'running' | 'completed' | 'failed' | 'cancelled';

export type BuildRequest = {
  projectId: string;
  componentIds: string[];
  environment?: string;
};

export type BuildResult = {
  projectId: string;
  projectName: string;
  componentId: string;
  componentName: string;
  environment?: string;
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
  environment?: string;
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
  environment?: string;
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
export type DaliAvailability = { available: boolean; configuredExecutable: string; resolvedExecutable: string; error?: string; installCommand: string; releaseUrl: string };

export type TransferStatus = 'ready' | 'sending' | 'success' | 'failed' | 'skipped' | 'cancelled';
export type TransferRunStatus = 'running' | 'completed' | 'failed' | 'cancelled';

export type TransferRequest = {
  projectId: string;
  componentIds: string[];
  version: string;
  filenameTemplate?: string;
  packageNames?: Record<string, string>;
  environment?: string;
  releaseDirectory?: string;
  packagePaths?: string[];
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
  releaseDirectory: string;
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
  command: string;
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
  environment?: string;
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
  environment?: string;
  releaseDirectory?: string;
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
  environment?: string;
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
  environment?: string;
  version: string;
  status: 'running' | 'completed' | 'failed' | 'cancelled';
  components: ReleaseComponentState[];
  startTime: string;
  endTime?: string;
  error?: string;
};
