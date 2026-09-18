<script lang="ts">
  import { FolderOpen, Trash2 } from 'lucide-svelte';
  import type { Component } from '../types';

  export let components: Component[] = [];
  export let editingIndex: number | null = null;
  export let onToggleDetails: (index: number) => void = () => undefined;
  export let onCloseDetails: () => void = () => undefined;
  export let onRemove: (index: number) => void = () => undefined;
  export let onUpdate: (index: number, changes: Partial<Component>) => void = () => undefined;
  export let onBrowsePath: (index: number) => void = () => undefined;
  export let onBrowseOutput: (index: number) => void = () => undefined;

  function inputValue(event: Event): string {
    return (event.currentTarget as HTMLInputElement).value;
  }

  function checkedValue(event: Event): boolean {
    return (event.currentTarget as HTMLInputElement).checked;
  }
</script>

<div class="component-list">
  {#each components as component, index}
    <article class:active={editingIndex === index} class="component-row">
      <div class="component-row-main">
        <span class="component-number">{index + 1}</span>
        <div>
          <strong>{component.name || 'New component'}</strong>
          <small>{component.path || 'Project path not configured'}</small>
          <div class="component-row-meta">
            <span class="component-row-id">{component.id}</span>
            <span class:enabled={component.package.enabled} class="tag">
              {component.package.enabled ? 'Packaging enabled' : 'Packaging disabled'}
            </span>
          </div>
        </div>
      </div>
      <div class="component-row-actions">
        <button class="secondary-button" on:click={() => onToggleDetails(index)}>
          {editingIndex === index ? 'Hide details' : 'View details'}
        </button>
        <button class="remove-button" on:click={() => onRemove(index)}>
          <Trash2 size={14} strokeWidth={2} aria-hidden="true" />Remove
        </button>
      </div>
    </article>
  {:else}
    <div class="empty-state">Add a component above.</div>
  {/each}
</div>

{#if editingIndex !== null}
  {#each components as component, index}
    {#if editingIndex === index}
      <article class="component-details">
        <div class="section-heading compact">
          <div>
            <p class="eyebrow">Component details</p>
            <h3>{component.name || 'New component'}</h3>
          </div>
          <button class="text-button" on:click={onCloseDetails}>Close details</button>
        </div>

        <div class="form-grid">
          <label class="field">
            <span>Name</span>
            <input
              value={component.name}
              on:input={(event) => onUpdate(index, { name: inputValue(event) })}
            />
          </label>

          <div class="field">
            <span>Project path</span>
            <div class="input-action">
              <input
                value={component.path}
                on:input={(event) => onUpdate(index, { path: inputValue(event) })}
              />
              <button class="secondary-button" on:click={() => onBrowsePath(index)}>
                <FolderOpen size={15} strokeWidth={2} aria-hidden="true" />Browse
              </button>
            </div>
          </div>

          <label class="field">
            <span>Legacy default build command</span>
            <input
              value={component.buildCommand}
              on:input={(event) => onUpdate(index, { buildCommand: inputValue(event) })}
            />
            <small>Used when the selected profile has no component command.</small>
          </label>

          <div class="field">
            <span>Output directory</span>
            <div class="input-action">
              <input
                value={component.outputDirectory}
                on:input={(event) => onUpdate(index, { outputDirectory: inputValue(event) })}
              />
              <button class="secondary-button" on:click={() => onBrowseOutput(index)}>
                <FolderOpen size={15} strokeWidth={2} aria-hidden="true" />Browse
              </button>
            </div>
          </div>

          <label class="field checkbox-field">
            <input
              type="checkbox"
              checked={component.package.enabled}
              on:change={(event) =>
                onUpdate(index, {
                  package: { ...component.package, enabled: checkedValue(event) },
                })}
            />
            <span>Packaging enabled</span>
          </label>

          <label class="field">
            <span>Package filename template</span>
            <input
              value={component.package.filename}
              on:input={(event) =>
                onUpdate(index, {
                  package: { ...component.package, filename: inputValue(event) },
                })}
            />
          </label>
        </div>
      </article>
    {/if}
  {/each}
{/if}
