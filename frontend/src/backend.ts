import {
  CancelBuild as generatedCancelBuild,
  DeleteProject as generatedDeleteProject,
  GetProject as generatedGetProject,
  GetProjects as generatedGetProjects,
  SaveProject as generatedSaveProject,
  StartBuild as generatedStartBuild,
  ValidateProject as generatedValidateProject,
} from '../wailsjs/go/app/App';
import type { BuildRequest, BuildRun, Project, ValidationIssue } from './types';

export const CancelBuild = generatedCancelBuild;
export const DeleteProject = generatedDeleteProject;
export const GetProject = (id: string): Promise<Project> => generatedGetProject(id) as Promise<Project>;
export const GetProjects = (): Promise<Project[]> => generatedGetProjects() as Promise<Project[]>;
export const SaveProject = (project: Project): Promise<Project> => generatedSaveProject(project as unknown as Parameters<typeof generatedSaveProject>[0]) as Promise<Project>;
export const StartBuild = (request: BuildRequest): Promise<BuildRun> => generatedStartBuild(request as unknown as Parameters<typeof generatedStartBuild>[0]) as Promise<BuildRun>;
export const ValidateProject = (project: Project): Promise<ValidationIssue[]> => generatedValidateProject(project as unknown as Parameters<typeof generatedValidateProject>[0]) as Promise<ValidationIssue[]>;
