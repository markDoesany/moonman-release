<script lang="ts">
  import { onMount } from 'svelte';
  import { DeleteProject, GetProjects, SaveProject } from './backend';
  import type { Component, Project, ValidationIssue } from './types';

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
  let errorMessage = '';
  let successMessage = '';
  let issues: ValidationIssue[] = [];

  $: selectedProject = projects.find((project) => project.id === selectedProjectId) ?? null;

  onMount(loadProjects);

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
    selectedProjectId = id;
    selectedComponentIds = projects.find((project) => project.id === id)?.components.map((component) => component.id) ?? [];
  }

  function toggleComponent(component: Component) {
    selectedComponentIds = selectedComponentIds.includes(component.id)
      ? selectedComponentIds.filter((id) => id !== component.id)
      : [...selectedComponentIds, component.id];
  }

  function openNewProject() {
    settingsProject = { id: '', name: '', components: [] };
    settingsIsNew = true;
    issues = [];
    successMessage = '';
    view = 'settings';
  }

  function openProjectSettings(project: Project) {
    settingsProject = structuredClone(project);
    settingsIsNew = false;
    issues = [];
    successMessage = '';
    view = 'settings';
  }

  function closeSettings() {
    settingsProject = null;
    issues = [];
    view = 'launcher';
  }

  async function saveSettings() {
    if (!settingsProject) return;
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
    if (!settingsProject || settingsIsNew) return;
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
    if (settingsProject) settingsProject = { ...settingsProject, name };
  }

  function addComponent() {
    if (!settingsProject) return;
    settingsProject = { ...settingsProject, components: [...settingsProject.components, emptyComponent()] };
  }

  function updateComponent(index: number, changes: Partial<Component>) {
    if (!settingsProject) return;
    settingsProject = {
      ...settingsProject,
      components: settingsProject.components.map((component, componentIndex) =>
        componentIndex === index ? { ...component, ...changes } : component,
      ),
    };
  }

  function removeComponent(index: number) {
    if (!settingsProject) return;
    settingsProject = {
      ...settingsProject,
      components: settingsProject.components.filter((_, componentIndex) => componentIndex !== index),
    };
  }

  function emptyComponent(): Component {
    return { id: '', name: '', path: '', buildCommand: '', outputDirectory: '', packageEnabled: true, packageFilename: '' };
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
    <button class="ghost-button" on:click={() => (view = view === 'launcher' ? 'settings' : 'launcher')}>
      {view === 'launcher' ? 'Project Settings' : 'Back to Launcher'}
    </button>
  </header>

  {#if errorMessage}<div class="alert error" role="alert">{errorMessage}</div>{/if}
  {#if successMessage}<div class="alert success" role="status">{successMessage}</div>{/if}

  {#if loading}
    <main class="card loading">Loading project configuration…</main>
  {:else if view === 'launcher'}
    <main class="card launcher-view">
      <div class="section-heading">
        <div><p class="eyebrow">Phase 1 foundation</p><h2>Prepare a release</h2></div>
        <span class="status-pill">Build pipeline coming in Phase 2</span>
      </div>

      <label class="field project-field"><span>Project</span>
        <select value={selectedProjectId} on:change={(event) => selectProject(selectValue(event))}>
          {#if projects.length === 0}<option value="">No projects configured</option>{/if}
          {#each projects as project}<option value={project.id}>{project.name}</option>{/each}
        </select>
      </label>

      {#if selectedProject}
        <div class="component-section">
          <div class="section-heading compact"><div><h3>Components</h3><p>Select components for a future release.</p></div><button class="text-button" on:click={() => openProjectSettings(selectedProject)}>Edit project</button></div>
          <div class="component-grid">
            {#each selectedProject.components as component}
              <label class="component-option"><input type="checkbox" checked={selectedComponentIds.includes(component.id)} on:change={() => toggleComponent(component)} /><span>{component.name}</span></label>
            {:else}<p class="muted">No components configured.</p>{/each}
          </div>
        </div>
      {:else}<div class="empty-state">Add a project in Project Settings to begin.</div>{/if}

      <div class="action-row"><button class="primary-button" disabled>Build Only</button><button class="secondary-button" disabled>Build &amp; Send</button></div>
      <p class="phase-note">Release execution will be available in Phase 2.</p>
    </main>
  {:else}
    <main class="settings-layout">
      <aside class="card project-list">
        <div class="section-heading compact"><h2>Projects</h2><button class="icon-button" title="Add project" on:click={openNewProject}>+</button></div>
        {#each projects as project}
          <button class:active={settingsProject?.id === project.id} class="project-list-item" on:click={() => openProjectSettings(project)}><span>{project.name}</span><small>{project.components.length} components</small></button>
        {:else}<p class="muted">No projects yet.</p>{/each}
      </aside>

      <section class="card settings-editor">
        {#if settingsProject}
          <div class="section-heading"><div><p class="eyebrow">Project configuration</p><h2>{settingsIsNew ? 'Add project' : `Edit ${settingsProject.name}`}</h2></div>{#if !settingsIsNew}<button class="danger-button" on:click={deleteSettingsProject}>Delete project</button>{/if}</div>
          <label class="field"><span>Project Name</span><input value={settingsProject.name} on:input={(event) => updateProjectName(inputValue(event))} placeholder="LokalStore" /></label>
          {#if issues.length > 0}<div class="validation-box">{#each issues as issue}<p>{issue.field}: {issue.message}</p>{/each}</div>{/if}

          <div class="section-heading compact components-heading"><div><h3>Components</h3><p>Configure future build and package inputs.</p></div><button class="text-button" on:click={addComponent}>+ Add component</button></div>
          {#each settingsProject.components as component, index}
            <article class="component-editor">
              <div class="component-editor-title"><h3>{component.name || 'New component'}</h3><button class="remove-button" on:click={() => removeComponent(index)}>Remove</button></div>
              <div class="form-grid">
                <label class="field"><span>Component Name</span><input value={component.name} on:input={(event) => updateComponent(index, { name: inputValue(event) })} /></label>
                <label class="field"><span>Project Path</span><input value={component.path} on:input={(event) => updateComponent(index, { path: inputValue(event) })} placeholder="C:\Projects\LokalStore\Admin" /></label>
                <label class="field"><span>Build Command</span><input value={component.buildCommand} on:input={(event) => updateComponent(index, { buildCommand: inputValue(event) })} placeholder="npm run build" /></label>
                <label class="field"><span>Output Directory</span><input value={component.outputDirectory} on:input={(event) => updateComponent(index, { outputDirectory: inputValue(event) })} placeholder="build" /></label>
                <label class="field checkbox-field"><input type="checkbox" checked={component.packageEnabled} on:change={(event) => updateComponent(index, { packageEnabled: checkedValue(event) })} /><span>Package Enabled</span></label>
                <label class="field"><span>Package Filename</span><input value={component.packageFilename} on:input={(event) => updateComponent(index, { packageFilename: inputValue(event) })} placeholder="component.zip" /></label>
              </div>
            </article>
          {:else}<div class="empty-state">No components. Add the first component above.</div>{/each}

          <div class="action-row settings-actions"><button class="secondary-button" on:click={closeSettings}>Cancel</button><button class="primary-button" disabled={saving} on:click={saveSettings}>{saving ? 'Saving…' : 'Save Project'}</button></div>
        {:else}<div class="empty-state large">Select a project or add a new one.</div>{/if}
      </section>
    </main>
  {/if}
</div>
