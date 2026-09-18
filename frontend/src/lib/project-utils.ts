import type { Component, EnvironmentProfile, Project } from '../types';

export function normalizeProject(project: Project): Project {
  return {
    ...project,
    components: project.components ?? [],
    environments: project.environments ?? [],
  };
}

export function legacyEnvironments(project: Project | null): EnvironmentProfile[] {
  const ids = new Set(['dev', 'staging', 'prod']);

  (project?.components ?? []).forEach((component) => {
    Object.keys(component.buildCommands ?? {}).forEach((id) => ids.add(id));
  });

  return [...ids].map((id) => ({
    id,
    name:
      id === 'dev'
        ? 'Development'
        : id === 'staging'
          ? 'Staging'
          : id === 'prod'
            ? 'Production'
            : id,
    commands: {},
  }));
}

export function defaultEnvironments(): EnvironmentProfile[] {
  return [
    { id: 'dev', name: 'Development', commands: {} },
    { id: 'staging', name: 'Staging', commands: {} },
    { id: 'prod', name: 'Production', commands: {} },
  ];
}

export function createEmptyComponent(
  existing: Component[],
  currentSequence: number,
): { component: Component; nextSequence: number } {
  const used = new Set(existing.map((component) => component.id));
  let nextSequence = currentSequence;
  let id = '';

  do {
    nextSequence += 1;
    id = `component-${nextSequence}`;
  } while (used.has(id));

  return {
    nextSequence,
    component: {
      id,
      name: '',
      path: '',
      buildCommand: 'npm run build',
      buildCommands: {},
      outputDirectory: 'build',
      package: {
        enabled: true,
        filename: '{project}-{component}-v{version}-{date}.zip',
      },
    },
  };
}

export function uniqueEnvironmentId(items: EnvironmentProfile[]): string {
  let id = 'qa';
  let index = 2;

  while (items.some((item) => item.id === id)) {
    id = `env-${index++}`;
  }

  return id;
}
