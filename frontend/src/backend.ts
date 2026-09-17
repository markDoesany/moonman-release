import {
  CancelBuild as generatedCancelBuild,
  DeleteProject as generatedDeleteProject,
  GetPackagePlan as generatedGetPackagePlan,
  GetProject as generatedGetProject,
  GetProjects as generatedGetProjects,
  SaveProject as generatedSaveProject,
  StartBuildAndPackage as generatedStartBuildAndPackage,
  StartPackage as generatedStartPackage,
  StartBuild as generatedStartBuild,
  ValidateProject as generatedValidateProject,
} from '../wailsjs/go/app/App';
import type { BuildRequest, BuildRun, PackagePlan, PackageRequest, PackageRun, Project, ReleaseRun, ValidationIssue } from './types';

export const CancelBuild = generatedCancelBuild;
export const DeleteProject = generatedDeleteProject;
export const GetPackagePlan = (request: PackageRequest): Promise<PackagePlan> => generatedGetPackagePlan(request as unknown as Parameters<typeof generatedGetPackagePlan>[0]) as Promise<PackagePlan>;
export const GetProject = (id: string): Promise<Project> => generatedGetProject(id) as Promise<Project>;
export const GetProjects = (): Promise<Project[]> => generatedGetProjects() as Promise<Project[]>;
export const SaveProject = (project: Project): Promise<Project> => generatedSaveProject(project as unknown as Parameters<typeof generatedSaveProject>[0]) as Promise<Project>;
export const StartBuild = (request: BuildRequest): Promise<BuildRun> => generatedStartBuild(request as unknown as Parameters<typeof generatedStartBuild>[0]) as Promise<BuildRun>;
export const StartPackage = (request: PackageRequest): Promise<PackageRun> => generatedStartPackage(request as unknown as Parameters<typeof generatedStartPackage>[0]) as Promise<PackageRun>;
export const StartBuildAndPackage = (request: PackageRequest): Promise<ReleaseRun> => generatedStartBuildAndPackage(request as unknown as Parameters<typeof generatedStartBuildAndPackage>[0]) as Promise<ReleaseRun>;
export const ValidateProject = (project: Project): Promise<ValidationIssue[]> => generatedValidateProject(project as unknown as Parameters<typeof generatedValidateProject>[0]) as Promise<ValidationIssue[]>;
