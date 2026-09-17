<script lang="ts">
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import { CancelBuild, DeleteProject, GetProjects, SaveProject, StartBuild } from './backend';
  import type { BuildEvent, BuildOutputLine, BuildRun, Component, Project, ValidationIssue } from './types';

  type View = 'launcher' | 'settings';

  let projects: Project[] = [];
  let selectedProjectId = '';
  let selectedComponentIds: string[] = [];
  let selectedProject: Project | null = null;
  let view: View = 'launcher';
  let settingsProject: Project | null = null;
  let settingsIsNew = false;
  let loading = true;
  let saving = false;
  let buildStarting = false;
  let buildRun: BuildRun | null = null;
  let buildOutput: BuildOutputLine[] = [];
  let pendingBuildEvents: BuildEvent[] = [];
  let errorMessage = '';
  let successMessage = '';
  let issues: ValidationIssue[] = [];

  $: selectedProject = projects.find((project) => project.id === selectedProjectId) ?? null;
  $: buildActive = buildStarting || buildRun?.status === 'running';

  onMount(() => {
    const stopListening = EventsOn('release-launcher:build-event', (event: BuildEvent) => handleBuildEvent(event));
    void loadProjects();
    return stopListening;
  });

  async function loadProjects() {
    loading = true;
    errorMessage = '';
    try {
      projects = await GetProjects();
      selectedProjectId = projects[0]?.id ?? '';
      selectedComponentIds = projects[0]?.components.map((component) => component.id) ?? [];
    } catch (error) {
      errorMessage = readableError(error);
    } finally {
      loading = false;
    }
  }

  function selectProject(id: string) {
    if (buildActive) return;
    selectedProjectId = id;
    selectedComponentIds = projects.find((project) => project.id === id)?.components.map((component) => component.id) ?? [];
  }

  function toggleComponent(component: Component) {
    if (buildActive) return;
    selectedComponentIds = selectedComponentIds.includes(component.id)
      ? selectedComponentIds.filter((id) => id !== component.id)
      : [...selectedComponentIds, component.id];
  }

  async function startBuild() {
    if (!selectedProject || buildActive) return;
    if (selectedComponentIds.length === 0) {
      errorMessage = 'Select at least one component to build.';
      return;
    }
    errorMessage = '';
    successMessage = '';
    buildOutput = [];
    pendingBuildEvents = [];
    buildRun = null;
    buildStarting = true;
    try {
      buildRun = await StartBuild({ projectId: selectedProject.id, componentIds: selectedComponentIds });
      buildStarting = false;
      const queuedEvents = pendingBuildEvents;
      pendingBuildEvents = [];
      queuedEvents.forEach(applyBuildEvent);
    } catch (error) {
      buildStarting = false;
      pendingBuildEvents = [];
      errorMessage = readableError(error);
    }
  }

  async function cancelBuild() {
    if (!buildRun) return;
    try {
      await CancelBuild(buildRun.id);
    } catch (error) {
      errorMessage = readableError(error);
    }
  }

  function clearOutput() {
    if (!buildActive) buildOutput = [];
  }

  function handleBuildEvent(event: BuildEvent) {
    if (buildStarting && !buildRun) {
      pendingBuildEvents = [...pendingBuildEvents, event];
      return;
    }
    applyBuildEvent(event);
  }

  function applyBuildEvent(event: BuildEvent) {
    if (!buildRun || buildRun.id !== event.runId) return;
    if (event.type === 'output' && event.text !== undefined) {
      buildOutput = [...buildOutput, {
        componentName: event.componentName ?? 'Build',
        stream: event.stream ?? 'system',
        text: event.text,
      }];
    }
    if (event.componentId && event.status) {
      buildRun = {
        ...buildRun,
        components: buildRun.components.map((component) => component.componentId === event.componentId
          ? {
              ...component,
              status: event.status ?? component.status,
              message: event.error || statusLabel(event.status ?? component.status),
              result: event.result ?? component.result,
            }
          : component),
      };
    }
    if (event.type === 'run_finished' && event.runStatus) {
      buildRun = {
        ...buildRun,
        status: event.runStatus,
        endTime: event.timestamp,
        error: event.error,
      };
      if (event.runStatus === 'completed') successMessage = 'Build completed successfully.';
      if (event.runStatus === 'failed') errorMessage = event.error || 'Build failed.';
      if (event.runStatus === 'cancelled') errorMessage = 'Build cancelled.';
    }
  }

  function openNewProject() {
    if (buildActive) return;
    settingsProject = { id: '', name: '', components: [] };
    settingsIsNew = true;
    issues = [];
    successMessage = '';
    view = 'settings';
  }

  function openProjectSettings(project: Project) {
    if (buildActive) return;
    settingsProject = structuredClone(project);
    settingsIsNew = false;
    issues = [];
    successMessage = '';
    view = 'settings';
  }

  function closeSettings() {
    if (buildActive) return;
    settingsProject = null;
    issues = [];
    view = 'launcher';
  }

  async function saveSettings() {
    if (!settingsProject || buildActive) return;
    saving = true;
    issues = [];
    errorMessage = '';
    successMessage = '';
    try {
      const saved = await SaveProject(settingsProject);
      const index = projects.findIndex((project) => project.id === saved.id);
      projects = index === -1
        ? [...projects, saved]
        : projects.map((project) => (project.id === saved.id ? saved : project));
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

  async function deleteSettingsProject() {
    if (!settingsProject || settingsIsNew || buildActive) return;
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
    if (settingsProject && !buildActive) settingsProject = { ...settingsProject, name };
  }

  function addComponent() {
    if (!settingsProject || buildActive) return;
    settingsProject = { ...settingsProject, components: [...settingsProject.components, emptyComponent()] };
  }

  function updateComponent(index: number, changes: Partial<Component>) {
    if (!settingsProject || buildActive) return;
    settingsProject = {
      ...settingsProject,
      components: settingsProject.components.map((component, componentIndex) =>
        componentIndex === index ? { ...component, ...changes } : component,
      ),
    };
  }

  function removeComponent(index: number) {
    if (!settingsProject || buildActive) return;
    settingsProject = {
      ...settingsProject,
      components: settingsProject.components.filter((_, componentIndex) => componentIndex !== index),
    };
  }

  function emptyComponent(): Component {
    return { id: '', name: '', path: '', buildCommand: '', outputDirectory: '', packageEnabled: true, packageFilename: '' };
  }

  function statusLabel(status: string): string {
    return status.charAt(0).toUpperCase() + status.slice(1);
  }

  function statusIcon(status: string): string {
    return { ready: '○', building: '⟳', success: '✓', failed: '✕', skipped: '—', cancelled: '!' }[status] ?? '○';
  }

  function inputValue(event: Event): string {
    return (event.currentTarget as HTMLInputElement).value;
  }

  function checkedValue(event: Event): boolean {
    return (event.currentTarget as HTMLInputElement).checked;
  }

  function selectValue(event: Event): string {
    return (event.currentTarget as HTMLSelectElement).value;
  }

  function readableError(error: unknown): string {
    return error instanceof Error ? error.message : String(error);
  }
</script>

<svelte:head><title>Release Launcher</title></svelte:head>

<div class="shell">
  <header class="topbar">
    <div><p class="eyebrow">Developer release workspace</p><h1>Release Launcher</h1></div>
    <button class="ghost-button" disabled={buildActive} on:click={() => (view = view === 'launcher' ? 'settings' : 'launcher')}>
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
        <div><p class="eyebrow">Phase 2 build engine</p><h2>Prepare a release</h2></div>
        <span class:active={buildActive} class="status-pill">{buildActive ? 'Build in progress' : 'Build ready'}</span>
      </div>

      <label class="field project-field"><span>Project</span>
        <select value={selectedProjectId} disabled={buildActive} on:change={(event) => selectProject(selectValue(event))}>
          {#if projects.length === 0}<option value="">No projects configured</option>{/if}
          {#each projects as project}<option value={project.id}>{project.name}</option>{/each}
        </select>
      </label>

      {#if selectedProject}
        <div class="component-section">
          <div class="section-heading compact"><div><h3>Components</h3><p>Select components to build sequentially.</p></div><button class="text-button" disabled={buildActive} on:click={() => openProjectSettings(selectedProject)}>Edit project</button></div>
          <div class="component-grid">
            {#each selectedProject.components as component}
              <label class="component-option"><input type="checkbox" disabled={buildActive} checked={selectedComponentIds.includes(component.id)} on:change={() => toggleComponent(component)} /><span>{component.name}</span></label>
            {:else}<p class="muted">No components configured.</p>{/each}
          </div>
        </div>
      {:else}<div class="empty-state">Add a project in Project Settings to begin.</div>{/if}

      <div class="action-row">
        <button class="primary-button" disabled={buildActive || !selectedProject || selectedComponentIds.length === 0} on:click={startBuild}>{buildStarting ? 'Starting...' : 'Build Selected'}</button>
        {#if buildActive && buildRun}<button class="danger-button" on:click={cancelBuild}>Cancel Build</button>{/if}
        <button class="secondary-button" disabled>Build &amp; Send</button>
        <button class="text-button clear-button" disabled={buildActive || buildOutput.length === 0} on:click={clearOutput}>Clear Output</button>
      </div>
      <p class="phase-note">Build and output validation are active. Packaging and Dali sending remain future phases.</p>

      {#if buildRun || buildStarting}
        <section class="build-panel">
          <div class="section-heading compact"><div><p class="eyebrow">Build Progress</p><h3>{buildRun?.projectName ?? selectedProject?.name}</h3></div><span class="run-status">{buildStarting ? 'Starting...' : statusLabel(buildRun?.status ?? 'running')}</span></div>
          {#if buildRun}
            <div class="build-progress">
              {#each buildRun.components as state}
                <div class="build-row">
                  <div><strong>{state.componentName}</strong><small>{state.message}</small></div>
                  <span class={`build-status status-${state.status}`}><b>{statusIcon(state.status)}</b>{statusLabel(state.status)}</span>
                </div>
              {/each}
            </div>
          {/if}
        </section>
      {/if}

      {#if buildOutput.length > 0}
        <section class="console-panel">
          <div class="section-heading compact"><div><p class="eyebrow">Build Output</p><h3>Live process console</h3></div></div>
          <div class="build-console" aria-live="polite">
            {#each buildOutput as line}
              <div class:stderr={line.stream === 'stderr'} class="console-line"><span>[{line.componentName}]</span> {line.text}</div>
            {/each}
          </div>
        </section>
      {/if}
    </main>
  {:else}
    <main class="settings-layout">
      <aside class="card project-list">
        <div class="section-heading compact"><h2>Projects</h2><button class="icon-button" disabled={buildActive} title="Add project" on:click={openNewProject}>+</button></div>
        {#each projects as project}
          <button class:active={settingsProject?.id === project.id} class="project-list-item" disabled={buildActive} on:click={() => openProjectSettings(project)}><span>{project.name}</span><small>{project.components.length} components</small></button>
        {:else}<p class="muted">No projects yet.</p>{/each}
      </aside>

      <section class="card settings-editor">
        {#if settingsProject}
          <div class="section-heading"><div><p class="eyebrow">Project configuration</p><h2>{settingsIsNew ? 'Add project' : `Edit ${settingsProject.name}`}</h2></div>{#if !settingsIsNew}<button class="danger-button" disabled={buildActive} on:click={deleteSettingsProject}>Delete project</button>{/if}</div>
          <label class="field"><span>Project Name</span><input disabled={buildActive} value={settingsProject.name} on:input={(event) => updateProjectName(inputValue(event))} placeholder="LokalStore" /></label>
          {#if issues.length > 0}<div class="validation-box">{#each issues as issue}<p>{issue.field}: {issue.message}</p>{/each}</div>{/if}

          <div class="section-heading compact components-heading"><div><h3>Components</h3><p>Configure build and future package inputs.</p></div><button class="text-button" disabled={buildActive} on:click={addComponent}>+ Add component</button></div>
          {#each settingsProject.components as component, index}
            <article class="component-editor">
              <div class="component-editor-title"><h3>{component.name || 'New component'}</h3><button class="remove-button" disabled={buildActive} on:click={() => removeComponent(index)}>Remove</button></div>
              <div class="form-grid">
                <label class="field"><span>Component Name</span><input disabled={buildActive} value={component.name} on:input={(event) => updateComponent(index, { name: inputValue(event) })} /></label>
                <label class="field"><span>Project Path</span><input disabled={buildActive} value={component.path} on:input={(event) => updateComponent(index, { path: inputValue(event) })} placeholder="C:\Projects\LokalStore\Admin" /></label>
                <label class="field"><span>Build Command</span><input disabled={buildActive} value={component.buildCommand} on:input={(event) => updateComponent(index, { buildCommand: inputValue(event) })} placeholder="npm run build" /></label>
                <label class="field"><span>Output Directory</span><input disabled={buildActive} value={component.outputDirectory} on:input={(event) => updateComponent(index, { outputDirectory: inputValue(event) })} placeholder="build" /></label>
                <label class="field checkbox-field"><input type="checkbox" disabled={buildActive} checked={component.packageEnabled} on:change={(event) => updateComponent(index, { packageEnabled: checkedValue(event) })} /><span>Package Enabled</span></label>
                <label class="field"><span>Package Filename</span><input disabled={buildActive} value={component.packageFilename} on:input={(event) => updateComponent(index, { packageFilename: inputValue(event) })} placeholder="component.zip" /></label>
              </div>
            </article>
          {:else}<div class="empty-state">No components. Add the first component above.</div>{/each}

          <div class="action-row settings-actions"><button class="secondary-button" disabled={buildActive} on:click={closeSettings}>Cancel</button><button class="primary-button" disabled={saving || buildActive} on:click={saveSettings}>{saving ? 'Saving...' : 'Save Project'}</button></div>
        {:else}<div class="empty-state large">Select a project or add a new one.</div>{/if}
      </section>
    </main>
  {/if}
</div>
