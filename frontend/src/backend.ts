import {
  CancelBuild as generatedCancelBuild,
  DeleteProject as generatedDeleteProject,
  GetDaliConfig as generatedGetDaliConfig,
  GetPackagePlan as generatedGetPackagePlan,
  GetTransferPlan as generatedGetTransferPlan,
  GetProject as generatedGetProject,
  GetProjects as generatedGetProjects,
  SaveDaliConfig as generatedSaveDaliConfig,
  SaveProject as generatedSaveProject,
  StartBuildAndPackage as generatedStartBuildAndPackage,
  StartBuildPackageAndSend as generatedStartBuildPackageAndSend,
  StartPackage as generatedStartPackage,
  StartBuild as generatedStartBuild,
  StartTransfer as generatedStartTransfer,
  ValidateProject as generatedValidateProject,
} from '../wailsjs/go/app/App';
import type { BuildRequest, BuildRun, DaliConfig, PackagePlan, PackageRequest, PackageRun, Project, ReleaseRun, TransferPlan, TransferRequest, TransferRun, ValidationIssue } from './types';

export const CancelBuild = generatedCancelBuild;
export const DeleteProject = generatedDeleteProject;
export const GetDaliConfig = (): Promise<DaliConfig> => generatedGetDaliConfig() as Promise<DaliConfig>;
export const GetPackagePlan = (request: PackageRequest): Promise<PackagePlan> => generatedGetPackagePlan(request as unknown as Parameters<typeof generatedGetPackagePlan>[0]) as Promise<PackagePlan>;
export const GetTransferPlan = (request: TransferRequest): Promise<TransferPlan> => generatedGetTransferPlan(request as unknown as Parameters<typeof generatedGetTransferPlan>[0]) as Promise<TransferPlan>;
export const GetProject = (id: string): Promise<Project> => generatedGetProject(id) as Promise<Project>;
export const GetProjects = (): Promise<Project[]> => generatedGetProjects() as Promise<Project[]>;
export const SaveProject = (project: Project): Promise<Project> => generatedSaveProject(project as unknown as Parameters<typeof generatedSaveProject>[0]) as Promise<Project>;
export const StartBuild = (request: BuildRequest): Promise<BuildRun> => generatedStartBuild(request as unknown as Parameters<typeof generatedStartBuild>[0]) as Promise<BuildRun>;
export const StartPackage = (request: PackageRequest): Promise<PackageRun> => generatedStartPackage(request as unknown as Parameters<typeof generatedStartPackage>[0]) as Promise<PackageRun>;
export const StartBuildAndPackage = (request: PackageRequest): Promise<ReleaseRun> => generatedStartBuildAndPackage(request as unknown as Parameters<typeof generatedStartBuildAndPackage>[0]) as Promise<ReleaseRun>;
export const StartBuildPackageAndSend = (request: PackageRequest): Promise<ReleaseRun> => generatedStartBuildPackageAndSend(request as unknown as Parameters<typeof generatedStartBuildPackageAndSend>[0]) as Promise<ReleaseRun>;
export const StartTransfer = (request: TransferRequest): Promise<TransferRun> => generatedStartTransfer(request as unknown as Parameters<typeof generatedStartTransfer>[0]) as Promise<TransferRun>;
export const SaveDaliConfig = (settings: DaliConfig): Promise<DaliConfig> => generatedSaveDaliConfig(settings as unknown as Parameters<typeof generatedSaveDaliConfig>[0]) as Promise<DaliConfig>;
export const ValidateProject = (project: Project): Promise<ValidationIssue[]> => generatedValidateProject(project as unknown as Parameters<typeof generatedValidateProject>[0]) as Promise<ValidationIssue[]>;
