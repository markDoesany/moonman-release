<script lang="ts">
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import { CancelBuild, DeleteProject, GetDaliConfig, GetPackagePlan, GetProjects, GetTransferPlan, PickDirectory, PickFile, SaveDaliConfig, SaveProject, StartBuild, StartBuildAndPackage, StartBuildPackageAndSend, StartPackage, StartTransfer } from './backend';
  import type { BuildEvent, BuildOutputLine, BuildRun, Component, DaliConfig, PackagePlan, PackageRun, PackageRequest, Project, ReleaseRun, TransferRequest, TransferRun, ValidationIssue } from './types';

  type View = 'launcher' | 'settings';
  type Operation = 'build' | 'package' | 'transfer' | 'release' | 'release-transfer';

  let projects: Project[] = [];
  let selectedProjectId = '';
  let selectedComponentIds: string[] = [];
  let selectedProject: Project | null = null;
  let view: View = 'launcher';
  let settingsProject: Project | null = null;
  let settingsIsNew = false;
  let loading = true;
  let saving = false;
  let daliSaving = false;
  let operationStarting = false;
  let operation: Operation | null = null;
  let buildRun: BuildRun | null = null;
  let packageRun: PackageRun | null = null;
  let transferRun: TransferRun | null = null;
  let releaseRun: ReleaseRun | null = null;
  let daliConfig: DaliConfig = { executable: 'dali', peerName: '', peerAddress: '', auto: true, wait: false };
  let packageVersion = '1.0.0';
  let packageTemplate = '';
  let namingPlan: PackagePlan | null = null;
  let namingNames: Record<string, string> = {};
  let namingError = '';
  let namingRefresh = 0;
  let buildOutput: BuildOutputLine[] = [];
  let pendingBuildEvents: BuildEvent[] = [];
  let errorMessage = '';
  let successMessage = '';
  let issues: ValidationIssue[] = [];

  $: selectedProject = projects.find((project) => project.id === selectedProjectId) ?? null;
  $: operationRunning = (operationStarting && namingPlan === null) || buildRun?.status === 'running' || packageRun?.status === 'running' || transferRun?.status === 'running' || releaseRun?.status === 'running';
  $: operationActive = operationRunning || namingPlan !== null;

  onMount(() => {
    const stopListening = EventsOn('release-launcher:build-event', (event: BuildEvent) => handleBuildEvent(event));
    void loadProjects();
    return stopListening;
  });

  async function loadProjects() {
    loading = true;
    errorMessage = '';
    try {
      const [loadedProjects, loadedDaliConfig] = await Promise.all([GetProjects(), GetDaliConfig()]);
      projects = loadedProjects;
      daliConfig = loadedDaliConfig;
      selectedProjectId = projects[0]?.id ?? '';
      selectedComponentIds = projects[0]?.components.map((component) => component.id) ?? [];
    } catch (error) {
      errorMessage = readableError(error);
    } finally {
      loading = false;
    }
  }

  function selectProject(id: string) {
    if (operationActive) return;
    selectedProjectId = id;
    selectedComponentIds = projects.find((project) => project.id === id)?.components.map((component) => component.id) ?? [];
  }

  function toggleComponent(component: Component) {
    if (operationActive) return;
    selectedComponentIds = selectedComponentIds.includes(component.id)
      ? selectedComponentIds.filter((id) => id !== component.id)
      : [...selectedComponentIds, component.id];
  }

  function prepareOperation(kind: Operation) {
    errorMessage = '';
    successMessage = '';
    buildOutput = [];
    pendingBuildEvents = [];
    operation = kind;
    operationStarting = true;
    buildRun = null;
    packageRun = null;
    transferRun = null;
    releaseRun = null;
    namingPlan = null;
    namingNames = {};
    namingError = '';
  }

  function finishStarting(queuedEvents: BuildEvent[]) {
    operationStarting = false;
    pendingBuildEvents = [];
    queuedEvents.forEach(applyBuildEvent);
  }

  async function startBuild() {
    if (!selectedProject || operationActive) return;
    if (selectedComponentIds.length === 0) {
      errorMessage = 'Select at least one component to build.';
      return;
    }
    prepareOperation('build');
    try {
      buildRun = await StartBuild({ projectId: selectedProject.id, componentIds: selectedComponentIds });
      finishStarting(pendingBuildEvents);
    } catch (error) {
      operationStarting = false;
      errorMessage = readableError(error);
    }
  }

  function packageRequest(overwrite = false): PackageRequest {
    return { projectId: selectedProject?.id ?? '', componentIds: selectedComponentIds, version: packageVersion, overwrite, filenameTemplate: packageTemplate.trim(), packageNames: { ...namingNames } };
  }

  async function beginNaming(kind: Operation) {
    if (!selectedProject || operationActive) return;
    if (selectedComponentIds.length === 0) {
      errorMessage = kind === 'transfer' ? 'Select at least one component to send.' : 'Select at least one component for the release workflow.';
      return;
    }
    errorMessage = '';
    successMessage = '';
    buildOutput = [];
    pendingBuildEvents = [];
    operation = kind;
    operationStarting = true;
    buildRun = null;
    packageRun = null;
    transferRun = null;
    releaseRun = null;
    try {
      const plan = await GetPackagePlan({ ...packageRequest(), packageNames: {} });
      if (kind === 'transfer') {
        const transferPlan = await GetTransferPlan({ ...packageRequest(), packageNames: Object.fromEntries(plan.components.filter((item) => item.selected && item.enabled).map((item) => [item.componentId, item.resolvedFilename ?? ''])) } as TransferRequest);
        if (transferPlan.hasMissing) throw new Error('One or more selected packages do not exist. Package them before sending.');
      }
      namingPlan = plan;
      namingNames = Object.fromEntries(plan.components.filter((item) => item.selected && item.enabled).map((item) => [item.componentId, item.resolvedFilename ?? '']));
    } catch (error) {
      operationStarting = false;
      operation = null;
      errorMessage = readableError(error);
    }
  }

  async function editPackageName(componentId: string, value: string) {
    namingNames = { ...namingNames, [componentId]: value };
    namingError = '';
    const refresh = ++namingRefresh;
    try {
      const refreshed = await GetPackagePlan({ ...packageRequest(false), packageNames: { ...namingNames } });
      if (refresh === namingRefresh) namingPlan = refreshed;
    } catch (error) {
      if (refresh === namingRefresh) namingError = readableError(error);
    }
  }

  async function confirmNaming() {
    if (!selectedProject || !namingPlan || !operation) return;
    try {
      namingError = '';
      const names = { ...namingNames };
      const request = { ...packageRequest(false), packageNames: names };
      const refreshed = await GetPackagePlan(request);
      if (operation === 'transfer') {
        const transferPlan = await GetTransferPlan(request as TransferRequest);
        if (transferPlan.hasMissing) throw new Error('One or more selected packages do not exist. Package them before sending.');
      }
      namingPlan = refreshed;
      namingNames = Object.fromEntries(refreshed.components.filter((item) => item.selected && item.enabled).map((item) => [item.componentId, item.resolvedFilename ?? '']));
      if (refreshed.hasConflicts && operation !== 'transfer') {
        const overwrite = await confirmOverwrite(true, refreshed.releaseDirectory);
        if (overwrite === null) return;
        await startApproved(request, overwrite);
      } else {
        await startApproved(request, false);
      }
    } catch (error) {
      namingError = readableError(error);
    }
  }

  async function startApproved(request: PackageRequest, overwrite: boolean) {
    const approved = { ...request, overwrite };
    namingPlan = null;
    if (operation === 'package') packageRun = await StartPackage(approved);
    else if (operation === 'release') releaseRun = await StartBuildAndPackage(approved);
    else if (operation === 'release-transfer') releaseRun = await StartBuildPackageAndSend(approved);
    else if (operation === 'transfer') transferRun = await StartTransfer(approved as TransferRequest);
    finishStarting(pendingBuildEvents);
  }

  function cancelNaming() {
    namingPlan = null;
    namingNames = {};
    operationStarting = false;
    operation = null;
    namingError = '';
  }

  async function startPackageExisting() {
    await beginNaming('package');
  }

  async function startTransferExisting() {
    await beginNaming('transfer');
  }

  async function startBuildAndPackage() {
    await beginNaming('release');
  }

  async function startBuildPackageAndSend() {
    await beginNaming('release-transfer');
  }

  async function confirmOverwrite(hasConflicts: boolean, releaseDirectory: string): Promise<boolean | null> {
    if (!hasConflicts) return false;
    return window.confirm(`One or more packages already exist in ${releaseDirectory}. Replace them?`) ? true : null;
  }

  async function cancelOperation() {
    const runID = buildRun?.id ?? packageRun?.id ?? transferRun?.id ?? releaseRun?.id;
    if (!runID) return;
    try {
      await CancelBuild(runID);
    } catch (error) {
      errorMessage = readableError(error);
    }
  }

  function clearOutput() {
    if (!operationActive) buildOutput = [];
  }

  function handleBuildEvent(event: BuildEvent) {
    if (operationStarting && !buildRun && !packageRun && !releaseRun) {
      pendingBuildEvents = [...pendingBuildEvents, event];
      return;
    }
    applyBuildEvent(event);
  }

  function applyBuildEvent(event: BuildEvent) {
    if (event.phase === 'package') {
      applyPackageEvent(event);
    } else if (event.phase === 'transfer') {
      applyTransferEvent(event);
    } else if (event.phase === 'release') {
      applyReleaseEvent(event);
    } else {
      applyBuildOnlyEvent(event);
    }
  }

  function appendOutput(event: BuildEvent) {
    if (event.type === 'output' && event.text !== undefined) {
      buildOutput = [...buildOutput, { componentName: event.componentName ?? 'Build', stream: event.stream ?? 'system', text: event.text }];
    }
  }

  function applyBuildOnlyEvent(event: BuildEvent) {
    if (!buildRun || buildRun.id !== event.runId) return;
    appendOutput(event);
    if (event.componentId && event.status) {
      buildRun = { ...buildRun, components: buildRun.components.map((component) => component.componentId === event.componentId
        ? { ...component, status: event.status ?? component.status, message: event.error || statusLabel(event.status ?? component.status), result: event.result ?? component.result }
        : component) };
    }
    if (event.type === 'run_finished' && event.runStatus) {
      buildRun = { ...buildRun, status: event.runStatus, endTime: event.timestamp, error: event.error };
      showRunMessage(event.runStatus, event.error);
    }
  }

  function applyPackageEvent(event: BuildEvent) {
    if (!packageRun || packageRun.id !== event.runId) return;
    appendOutput(event);
    if (event.componentId && event.packageStatus) {
      packageRun = { ...packageRun, components: packageRun.components.map((component) => component.componentId === event.componentId
        ? { ...component, status: event.packageStatus ?? component.status, message: event.error || statusLabel(event.packageStatus ?? component.status), result: event.packageResult ?? component.result }
        : component) };
    }
    if (event.type === 'package_run_finished' && event.runStatus) {
      packageRun = { ...packageRun, status: event.runStatus as PackageRun['status'], endTime: event.timestamp, error: event.error };
      showRunMessage(event.runStatus, event.error);
    }
  }

  function applyTransferEvent(event: BuildEvent) {
    if (!transferRun || transferRun.id !== event.runId) return;
    appendOutput(event);
    if (event.componentId && event.transferStatus) {
      transferRun = { ...transferRun, components: transferRun.components.map((component) => component.componentId === event.componentId
        ? { ...component, status: event.transferStatus ?? component.status, message: event.error || statusLabel(event.transferStatus ?? component.status), result: event.transferResult ?? component.result }
        : component) };
    }
    if (event.type === 'transfer_run_finished' && event.runStatus) {
      transferRun = { ...transferRun, status: event.runStatus as TransferRun['status'], endTime: event.timestamp, error: event.error };
      showRunMessage(event.runStatus, event.error, 'transfer');
    }
  }

  function applyReleaseEvent(event: BuildEvent) {
    if (!releaseRun || releaseRun.id !== event.runId) return;
    appendOutput(event);
    if (event.componentId && event.status) {
      const messages = (event.error ?? '').split(' | ');
      releaseRun = { ...releaseRun, components: releaseRun.components.map((component) => component.componentId === event.componentId
        ? {
            ...component,
            buildStatus: event.status ?? component.buildStatus,
            buildMessage: messages[0] || component.buildMessage,
            buildResult: event.result ?? component.buildResult,
            packageStatus: event.packageStatus ?? component.packageStatus,
            packageMessage: messages[1] || component.packageMessage,
            packageResult: event.packageResult ?? component.packageResult,
            transferStatus: event.transferStatus ?? component.transferStatus,
            transferMessage: messages[2] || component.transferMessage,
            transferResult: event.transferResult ?? component.transferResult,
          }
        : component) };
    }
    if (event.type === 'release_run_finished' && event.runStatus) {
      releaseRun = { ...releaseRun, status: event.runStatus as ReleaseRun['status'], endTime: event.timestamp, error: event.error };
      showRunMessage(event.runStatus, event.error, operation === 'release-transfer' ? 'release-transfer' : 'release');
    }
  }

  function showRunMessage(status: string, error?: string, kind: Operation | 'transfer' | 'release' = operation ?? 'build') {
    if (status === 'completed') {
      successMessage = kind === 'transfer' ? 'Dali transfer completed successfully.' : kind === 'release-transfer' ? 'Build, packaging, and Dali transfer completed successfully.' : kind === 'release' ? 'Build and packaging completed successfully.' : kind === 'package' ? 'Packaging completed successfully.' : 'Build completed successfully.';
    }
    if (status === 'failed') errorMessage = error || (kind === 'transfer' ? 'Dali transfer failed.' : kind === 'release-transfer' ? 'Build, packaging, or Dali transfer failed.' : kind === 'release' ? 'Build or packaging failed.' : kind === 'package' ? 'Packaging failed.' : 'Build failed.');
    if (status === 'cancelled') errorMessage = error || 'Operation cancelled.';
  }

  function openNewProject() {
    if (operationActive) return;
    settingsProject = { id: '', name: '', components: [] };
    settingsIsNew = true;
    issues = [];
    successMessage = '';
    view = 'settings';
  }

  function openProjectSettings(project: Project) {
    if (operationActive) return;
    settingsProject = structuredClone(project);
    settingsIsNew = false;
    issues = [];
    successMessage = '';
    view = 'settings';
  }

  function closeSettings() {
    if (operationActive) return;
    settingsProject = null;
    issues = [];
    view = 'launcher';
  }

  async function saveSettings() {
    if (!settingsProject || operationActive) return;
    saving = true;
    issues = [];
    errorMessage = '';
    successMessage = '';
    try {
      const saved = await SaveProject(settingsProject);
      const index = projects.findIndex((project) => project.id === saved.id);
      projects = index === -1 ? [...projects, saved] : projects.map((project) => (project.id === saved.id ? saved : project));
      selectedProjectId = saved.id;
      selectedComponentIds = saved.components.map((component) => component.id);
      settingsProject = structuredClone(saved);
      settingsIsNew = false;
      successMessage = 'Project saved.';
    } catch (error) {
      errorMessage = readableError(error);
      issues = [{ field: 'project', message: errorMessage }];
    } finally {
      saving = false;
    }
  }

  async function saveDaliSettings() {
    if (operationActive) return;
    daliSaving = true;
    errorMessage = '';
    successMessage = '';
    try {
      daliConfig = await SaveDaliConfig(daliConfig);
      successMessage = 'Dali settings saved.';
    } catch (error) {
      errorMessage = readableError(error);
    } finally {
      daliSaving = false;
    }
  }

  async function deleteSettingsProject() {
    if (!settingsProject || settingsIsNew || operationActive) return;
    if (!window.confirm(`Delete ${settingsProject.name}?`)) return;
    try {
      const deletedId = settingsProject.id;
      await DeleteProject(deletedId);
      projects = projects.filter((project) => project.id !== deletedId);
      selectedProjectId = projects[0]?.id ?? '';
      selectedComponentIds = projects[0]?.components.map((component) => component.id) ?? [];
      closeSettings();
      successMessage = 'Project deleted.';
    } catch (error) {
      errorMessage = readableError(error);
    }
  }

  function updateProjectName(name: string) {
    if (settingsProject && !operationActive) settingsProject = { ...settingsProject, name };
  }

  function addComponent() {
    if (!settingsProject || operationActive) return;
    settingsProject = { ...settingsProject, components: [...settingsProject.components, emptyComponent()] };
  }

  function updateComponent(index: number, changes: Partial<Component>) {
    if (!settingsProject || operationActive) return;
    settingsProject = { ...settingsProject, components: settingsProject.components.map((component, componentIndex) => componentIndex === index ? { ...component, ...changes } : component) };
  }

  async function browseProjectPath(index: number) {
    if (!settingsProject || operationActive) return;
    try {
      const selected = await PickDirectory(settingsProject.components[index].path);
      if (selected) updateComponent(index, { path: selected });
    } catch (error) { errorMessage = readableError(error); }
  }

  async function browseOutputDirectory(index: number) {
    if (!settingsProject || operationActive) return;
    const component = settingsProject.components[index];
    try {
      const selected = await PickDirectory(joinPath(component.path, component.outputDirectory));
      if (selected) updateComponent(index, { outputDirectory: relativeIfInside(component.path, selected) });
    } catch (error) { errorMessage = readableError(error); }
  }

  async function browseDaliExecutable() {
    if (operationActive) return;
    try {
      const selected = await PickFile(daliConfig.executable);
      if (selected) daliConfig = { ...daliConfig, executable: selected };
    } catch (error) { errorMessage = readableError(error); }
  }

  function normalizePath(value: string): string { return value.trim().replaceAll('/', '\\').replace(/[\\]+$/, ''); }
  function joinPath(base: string, child: string): string { if (!child.trim()) return base; return /^[A-Za-z]:[\\/]|^[\\\\]/.test(child) ? child : `${base.replace(/[\\/]+$/, '')}\\${child}`; }
  function relativeIfInside(base: string, selected: string): string {
    const normalizedBase = normalizePath(base);
    const normalizedSelected = normalizePath(selected);
    const lowerBase = normalizedBase.toLowerCase();
    const lowerSelected = normalizedSelected.toLowerCase();
    if (lowerSelected === lowerBase) return '.';
    if (lowerSelected.startsWith(`${lowerBase}\\`)) return normalizedSelected.slice(normalizedBase.length + 1);
    return selected;
  }

  function removeComponent(index: number) {
    if (!settingsProject || operationActive) return;
    settingsProject = { ...settingsProject, components: settingsProject.components.filter((_, componentIndex) => componentIndex !== index) };
  }

  function emptyComponent(): Component {
    return { id: '', name: '', path: '', buildCommand: '', outputDirectory: '', package: { enabled: true, filename: '{project}-{component}-v{version}-{date}.zip' } };
  }

  function statusLabel(status: string): string { return status.charAt(0).toUpperCase() + status.slice(1); }
  function statusIcon(status: string): string { return { ready: '○', building: '⟳', success: '✓', failed: '✕', skipped: '—', cancelled: '!' }[status] ?? '○'; }
  function inputValue(event: Event): string { return (event.currentTarget as HTMLInputElement).value; }
  function checkedValue(event: Event): boolean { return (event.currentTarget as HTMLInputElement).checked; }
  function selectValue(event: Event): string { return (event.currentTarget as HTMLSelectElement).value; }
  function readableError(error: unknown): string { return error instanceof Error ? error.message : String(error); }
</script>

<svelte:head><title>Release Launcher</title></svelte:head>

<div class="shell">
  <header class="topbar">
    <div><p class="eyebrow">Developer release workspace</p><h1>Release Launcher</h1></div>
    <button class="ghost-button" disabled={operationActive} on:click={() => (view = view === 'launcher' ? 'settings' : 'launcher')}>
      {view === 'launcher' ? 'Project Settings' : 'Back to Launcher'}
    </button>
  </header>

  {#if errorMessage}<div class="alert error" role="alert">{errorMessage}</div>{/if}
  {#if successMessage}<div class="alert success" role="status">{successMessage}</div>{/if}

  {#if loading}
    <main class="card loading">Loading project configuration...</main>
  {:else if view === 'launcher'}
    <main class="card launcher-view">
      <div class="section-heading">
        <div><p class="eyebrow">Build, package, and send</p><h2>Prepare a release</h2></div>
        <span class:active={operationActive} class="status-pill">{operationActive ? 'Operation in progress' : 'Ready'}</span>
      </div>

      <div class="release-inputs">
        <label class="field project-field"><span>Project</span>
          <select value={selectedProjectId} disabled={operationActive} on:change={(event) => selectProject(selectValue(event))}>
            {#if projects.length === 0}<option value="">No projects configured</option>{/if}
            {#each projects as project}<option value={project.id}>{project.name}</option>{/each}
          </select>
        </label>
        <label class="field version-field"><span>Release Version</span><input value={packageVersion} disabled={operationActive} on:input={(event) => (packageVersion = inputValue(event))} placeholder="1.0.0" /></label>
        <label class="field template-field"><span>Package naming template</span><input value={packageTemplate} disabled={operationActive} on:input={(event) => (packageTemplate = inputValue(event))} placeholder="Use component templates" /><small>&#123;project&#125; &#123;component&#125; &#123;version&#125; &#123;date&#125; &#123;time&#125; &#123;datetime&#125;</small></label>
      </div>

      {#if selectedProject}
        <div class="component-section">
          <div class="section-heading compact"><div><h3>Components</h3><p>Select components for the release workflow.</p></div><button class="text-button" disabled={operationActive} on:click={() => openProjectSettings(selectedProject)}>Edit project</button></div>
          <div class="component-grid">
            {#each selectedProject.components as component}
              <label class="component-option"><input type="checkbox" disabled={operationActive} checked={selectedComponentIds.includes(component.id)} on:change={() => toggleComponent(component)} /><span>{component.name}</span></label>
            {:else}<p class="muted">No components configured.</p>{/each}
          </div>
        </div>
      {:else}<div class="empty-state">Add a project in Project Settings to begin.</div>{/if}

      <div class="action-row">
        <button class="primary-button" disabled={operationActive || !selectedProject || selectedComponentIds.length === 0} on:click={startBuild}>Build Only</button>
        <button class="secondary-button" disabled={operationActive || !selectedProject || selectedComponentIds.length === 0} on:click={startBuildAndPackage}>Build &amp; Package</button>
        <button class="secondary-button" disabled={operationActive || !selectedProject || selectedComponentIds.length === 0} on:click={startPackageExisting}>Package Existing Build</button>
        <button class="secondary-button" disabled={operationActive || !selectedProject || selectedComponentIds.length === 0} on:click={startTransferExisting}>Send Packages</button>
        <button class="primary-button" disabled={operationActive || !selectedProject || selectedComponentIds.length === 0} on:click={startBuildPackageAndSend}>Build, Package &amp; Send</button>
        {#if operationActive && (buildRun || packageRun || transferRun || releaseRun)}<button class="danger-button" on:click={cancelOperation}>Cancel</button>{/if}
        <button class="text-button clear-button" disabled={operationActive || buildOutput.length === 0} on:click={clearOutput}>Clear Output</button>
      </div>
      <p class="phase-note">Packages are written to releases/{selectedProject?.name ?? 'Project'}/{packageVersion || 'timestamp'} and sent through Dali to {daliConfig.peerName || daliConfig.peerAddress || 'an automatically selected peer'}.</p>

      {#if namingPlan}
        <section class="naming-panel">
          <div class="section-heading compact"><div><p class="eyebrow">Naming preview</p><h3>Review package names before {operation === 'transfer' ? 'sending' : 'starting'}</h3><p>Tokens: &#123;project&#125; &#123;component&#125; &#123;version&#125; &#123;date&#125; &#123;time&#125; &#123;datetime&#125;</p></div><span class="run-status">{namingPlan.version}</span></div>
          <div class="naming-table"><div class="naming-row naming-header"><strong>Component</strong><strong>Resolved filename</strong><strong>Release path / conflict</strong></div>
            {#each namingPlan.components.filter((item) => item.selected && item.enabled) as item}
              <div class="naming-row"><strong>{item.componentName}</strong><label class="inline-field"><span class="sr-only">Filename for {item.componentName}</span><input value={namingNames[item.componentId] ?? item.resolvedFilename ?? ''} on:input={(event) => editPackageName(item.componentId, inputValue(event))} /></label><div><code>{namingPlan.releaseDirectory}\{namingNames[item.componentId] ?? item.resolvedFilename}</code>{#if item.existing}<small class="conflict">Existing file will be replaced after confirmation</small>{:else}<small class="available">Available</small>{/if}</div></div>
            {/each}
          </div>
          {#if namingError}<div class="validation-box naming-error">{namingError}</div>{/if}
          <div class="action-row naming-actions"><button class="secondary-button" on:click={cancelNaming}>Cancel</button><button class="primary-button" on:click={confirmNaming}>Confirm naming &amp; continue</button></div>
        </section>
      {/if}

      {#if buildRun}
        <section class="build-panel"><div class="section-heading compact"><div><p class="eyebrow">Build Progress</p><h3>{buildRun.projectName}</h3></div><span class="run-status">{statusLabel(buildRun.status)}</span></div>
          <div class="build-progress">{#each buildRun.components as state}<div class="build-row"><div><strong>{state.componentName}</strong><small>{state.message}</small></div><span class={`build-status status-${state.status}`}><b>{statusIcon(state.status)}</b>{statusLabel(state.status)}</span></div>{/each}</div>
        </section>
      {:else if packageRun}
        <section class="build-panel"><div class="section-heading compact"><div><p class="eyebrow">Packaging Progress</p><h3>{packageRun.projectName} / {packageRun.version}</h3></div><span class="run-status">{statusLabel(packageRun.status)}</span></div>
          <div class="build-progress">{#each packageRun.components as state}<div class="build-row"><div><strong>{state.componentName}</strong><small>{state.message}{state.result?.packagePath ? ` — ${state.result.packagePath}` : ''}</small></div><span class={`build-status status-${state.status}`}><b>{statusIcon(state.status)}</b>{statusLabel(state.status)}</span></div>{/each}</div>
        </section>
      {:else if transferRun}
        <section class="build-panel"><div class="section-heading compact"><div><p class="eyebrow">Dali Transfer Progress</p><h3>{transferRun.projectName} / {transferRun.version}</h3></div><span class="run-status">{statusLabel(transferRun.status)}</span></div>
          <div class="build-progress">{#each transferRun.components as state}<div class="build-row"><div><strong>{state.componentName}</strong><small>{state.message}{state.result?.packagePath ? ` — ${state.result.packagePath}` : ''}</small></div><span class={`build-status status-${state.status}`}><b>{statusIcon(state.status)}</b>{statusLabel(state.status)}</span></div>{/each}</div>
        </section>
      {:else if releaseRun}
        <section class="build-panel"><div class="section-heading compact"><div><p class="eyebrow">Build &amp; Packaging Progress</p><h3>{releaseRun.projectName} / {releaseRun.version}</h3></div><span class="run-status">{statusLabel(releaseRun.status)}</span></div>
          <div class="build-progress">{#each releaseRun.components as state}<div class="build-row release-row"><div><strong>{state.componentName}</strong><small>Build: {state.buildMessage} · Package: {state.packageMessage} · Send: {state.transferMessage}</small></div><span class="release-status"><i class={`build-status status-${state.buildStatus}`}>{statusIcon(state.buildStatus)} {statusLabel(state.buildStatus)}</i><i class={`build-status status-${state.packageStatus}`}>{statusIcon(state.packageStatus)} {statusLabel(state.packageStatus)}</i><i class={`build-status status-${state.transferStatus}`}>{statusIcon(state.transferStatus)} {statusLabel(state.transferStatus)}</i></span></div>{/each}</div>
        </section>
      {/if}

      {#if buildOutput.length > 0}<section class="console-panel"><div class="section-heading compact"><div><p class="eyebrow">Build Output</p><h3>Live process console</h3></div></div><div class="build-console" aria-live="polite">{#each buildOutput as line}<div class:stderr={line.stream === 'stderr'} class="console-line"><span>[{line.componentName}]</span> {line.text}</div>{/each}</div></section>{/if}
    </main>
  {:else}
    <main class="settings-layout">
      <aside class="card project-list"><div class="section-heading compact"><h2>Projects</h2><button class="icon-button" disabled={operationActive} title="Add project" on:click={openNewProject}>+</button></div>{#each projects as project}<button class:active={settingsProject?.id === project.id} class="project-list-item" disabled={operationActive} on:click={() => openProjectSettings(project)}><span>{project.name}</span><small>{project.components.length} components</small></button>{:else}<p class="muted">No projects yet.</p>{/each}</aside>
      <section class="card settings-editor">{#if settingsProject}<div class="section-heading"><div><p class="eyebrow">Project configuration</p><h2>{settingsIsNew ? 'Add project' : `Edit ${settingsProject.name}`}</h2></div>{#if !settingsIsNew}<button class="danger-button" disabled={operationActive} on:click={deleteSettingsProject}>Delete project</button>{/if}</div>
        <label class="field"><span>Project Name</span><input disabled={operationActive} value={settingsProject.name} on:input={(event) => updateProjectName(inputValue(event))} placeholder="LokalStore" /></label>
        {#if issues.length > 0}<div class="validation-box">{#each issues as issue}<p>{issue.field}: {issue.message}</p>{/each}</div>{/if}
        <div class="section-heading compact components-heading"><div><h3>Components</h3><p>Configure build and package inputs.</p></div><button class="text-button" disabled={operationActive} on:click={addComponent}>+ Add component</button></div>
        {#each settingsProject.components as component, index}<article class="component-editor"><div class="component-editor-title"><h3>{component.name || 'New component'}</h3><button class="remove-button" disabled={operationActive} on:click={() => removeComponent(index)}>Remove</button></div><div class="form-grid">
          <label class="field"><span>Component Name</span><input disabled={operationActive} value={component.name} on:input={(event) => updateComponent(index, { name: inputValue(event) })} /></label>
          <div class="field"><span>Component Project Path</span><div class="input-action"><input disabled={operationActive} value={component.path} on:input={(event) => updateComponent(index, { path: inputValue(event) })} placeholder="C:\Projects\LokalStore\Admin" /><button class="secondary-button" disabled={operationActive} on:click={() => browseProjectPath(index)}>Browse</button></div></div>
          <label class="field"><span>Build Command</span><input disabled={operationActive} value={component.buildCommand} on:input={(event) => updateComponent(index, { buildCommand: inputValue(event) })} placeholder="npm run build" /></label>
          <div class="field"><span>Component Output Directory</span><div class="input-action"><input disabled={operationActive} value={component.outputDirectory} on:input={(event) => updateComponent(index, { outputDirectory: inputValue(event) })} placeholder="build or an absolute path" /><button class="secondary-button" disabled={operationActive} on:click={() => browseOutputDirectory(index)}>Browse</button></div><small class="field-help">Folders inside the project are saved as relative paths.</small></div>
          <label class="field checkbox-field"><input type="checkbox" disabled={operationActive} checked={component.package.enabled} on:change={(event) => updateComponent(index, { package: { ...component.package, enabled: checkedValue(event) } })} /><span>Package Enabled</span></label>
          <label class="field"><span>Package Filename Template</span><input disabled={operationActive} value={component.package.filename} on:input={(event) => updateComponent(index, { package: { ...component.package, filename: inputValue(event) } })} placeholder="&#123;project&#125;-&#123;component&#125;-v&#123;version&#125;-&#123;date&#125;.zip" /><small class="field-help">Tokens: &#123;project&#125; &#123;component&#125; &#123;version&#125; &#123;date&#125; &#123;time&#125; &#123;datetime&#125;</small></label>
        </div></article>{:else}<div class="empty-state">No components. Add the first component above.</div>{/each}
        <div class="action-row settings-actions"><button class="secondary-button" disabled={operationActive} on:click={closeSettings}>Cancel</button><button class="primary-button" disabled={saving || operationActive} on:click={saveSettings}>{saving ? 'Saving...' : 'Save Project'}</button></div>
      {:else}<div class="empty-state large">Select a project or add a new one.</div>{/if}
        <section class="dali-settings"><div class="section-heading compact"><div><p class="eyebrow">Dali transfer</p><h3>DevOps destination</h3><p>Configure the installed Dali CLI and the peer that receives release ZIP files.</p></div></div>
          <div class="form-grid">
            <div class="field"><span>Dali Executable</span><div class="input-action"><input disabled={operationActive} value={daliConfig.executable} on:input={(event) => (daliConfig = { ...daliConfig, executable: inputValue(event) })} placeholder="dali" /><button class="secondary-button" disabled={operationActive} on:click={browseDaliExecutable}>Browse</button></div></div>
            <label class="field"><span>Peer Name</span><input disabled={operationActive} value={daliConfig.peerName} on:input={(event) => (daliConfig = { ...daliConfig, peerName: inputValue(event), peerAddress: '' })} placeholder="DevOps" /></label>
            <label class="field"><span>Peer Address</span><input disabled={operationActive} value={daliConfig.peerAddress} on:input={(event) => (daliConfig = { ...daliConfig, peerAddress: inputValue(event), peerName: '' })} placeholder="192.168.1.20:45679" /></label>
            <label class="field checkbox-field"><input type="checkbox" disabled={operationActive} checked={daliConfig.auto} on:change={(event) => (daliConfig = { ...daliConfig, auto: checkedValue(event) })} /><span>Auto-select a single peer</span></label>
            <label class="field checkbox-field"><input type="checkbox" disabled={operationActive} checked={daliConfig.wait} on:change={(event) => (daliConfig = { ...daliConfig, wait: checkedValue(event) })} /><span>Wait for discovery timeout</span></label>
          </div>
          <div class="action-row settings-actions"><button class="primary-button" disabled={daliSaving || operationActive} on:click={saveDaliSettings}>{daliSaving ? 'Saving...' : 'Save Dali Settings'}</button></div>
        </section>
      </section>
    </main>
  {/if}
</div>
