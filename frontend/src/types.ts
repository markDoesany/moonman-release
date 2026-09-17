export type Component = {
  id: string;
  name: string;
  path: string;
  buildCommand: string;
  outputDirectory: string;
  packageEnabled: boolean;
  packageFilename: string;
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
  result?: BuildResult;
  results?: BuildResult[];
  error?: string;
};

export type BuildOutputLine = {
  componentName: string;
  stream: 'system' | 'stdout' | 'stderr';
  text: string;
};
