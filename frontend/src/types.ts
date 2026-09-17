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
