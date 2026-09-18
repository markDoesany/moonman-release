package models

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Project describes a releaseable product and its components.
type Project struct {
	ID                 string               `yaml:"id" json:"id"`
	Name               string               `yaml:"name" json:"name"`
	DefaultEnvironment string               `yaml:"default_environment,omitempty" json:"defaultEnvironment,omitempty"`
	Environments       []EnvironmentProfile `yaml:"environments,omitempty" json:"environments,omitempty"`
	Components         []Component          `yaml:"components" json:"components"`
}

// EnvironmentProfile groups build commands for a project-wide release target.
// Commands are keyed by component ID. Empty commands intentionally fall back to
// the legacy component command fields.
type EnvironmentProfile struct {
	ID       string            `yaml:"id" json:"id"`
	Name     string            `yaml:"name" json:"name"`
	Commands map[string]string `yaml:"commands" json:"commands"`
}

var DefaultEnvironmentProfiles = []EnvironmentProfile{
	{ID: "dev", Name: "Development", Commands: map[string]string{}},
	{ID: "staging", Name: "Staging", Commands: map[string]string{}},
	{ID: "prod", Name: "Production", Commands: map[string]string{}},
}

// Environment returns a configured profile by ID.
func (p Project) Environment(id string) (EnvironmentProfile, bool) {
	id = strings.TrimSpace(id)
	for _, environment := range p.Environments {
		if environment.ID == id {
			return environment, true
		}
	}
	return EnvironmentProfile{}, false
}

// EffectiveEnvironment applies the project's default when the caller omitted
// an environment. Legacy projects without profiles retain their old behavior.
func (p Project) EffectiveEnvironment(id string) string {
	if value := strings.TrimSpace(id); value != "" {
		return value
	}
	if value := strings.TrimSpace(p.DefaultEnvironment); value != "" {
		return value
	}
	if len(p.Environments) > 0 {
		return p.Environments[0].ID
	}
	return ""
}

// Component describes the source and future release settings for one component.
type Component struct {
	ID              string            `yaml:"id" json:"id"`
	Name            string            `yaml:"name" json:"name"`
	Path            string            `yaml:"path" json:"path"`
	BuildCommand    string            `yaml:"build_command" json:"buildCommand"`
	BuildCommands   map[string]string `yaml:"build_commands,omitempty" json:"buildCommands,omitempty"`
	OutputDirectory string            `yaml:"output_directory" json:"outputDirectory"`
	Package         PackageConfig     `yaml:"package" json:"package"`
}

// PackageConfig controls whether a component's build output is archived.
type PackageConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Filename string `yaml:"filename" json:"filename"`
}

const DefaultPackageFilenameTemplate = "{project}-{component}-v{version}-{date}.zip"

// UnmarshalYAML accepts both the Phase 3 nested package format and the Phase 1
// flat fields so existing project files continue to load safely.
func (c *Component) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		ID              string            `yaml:"id"`
		Name            string            `yaml:"name"`
		Path            string            `yaml:"path"`
		BuildCommand    string            `yaml:"build_command"`
		BuildCommands   map[string]string `yaml:"build_commands"`
		OutputDirectory string            `yaml:"output_directory"`
		Package         *PackageConfig    `yaml:"package"`
		PackageEnabled  *bool             `yaml:"package_enabled"`
		PackageFilename string            `yaml:"package_filename"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	c.ID = raw.ID
	c.Name = raw.Name
	c.Path = raw.Path
	c.BuildCommand = raw.BuildCommand
	c.BuildCommands = raw.BuildCommands
	c.OutputDirectory = raw.OutputDirectory
	if raw.Package != nil {
		c.Package = *raw.Package
	} else {
		if raw.PackageEnabled != nil {
			c.Package.Enabled = *raw.PackageEnabled
		}
		c.Package.Filename = raw.PackageFilename
	}
	return nil
}

// MarshalYAML always writes the current nested package format. This upgrades
// legacy flat configuration after the next successful save.
func (c Component) MarshalYAML() (interface{}, error) {
	return struct {
		ID              string            `yaml:"id"`
		Name            string            `yaml:"name"`
		Path            string            `yaml:"path"`
		BuildCommand    string            `yaml:"build_command"`
		BuildCommands   map[string]string `yaml:"build_commands,omitempty"`
		OutputDirectory string            `yaml:"output_directory"`
		Package         PackageConfig     `yaml:"package"`
	}{
		ID:              c.ID,
		Name:            c.Name,
		Path:            c.Path,
		BuildCommand:    c.BuildCommand,
		BuildCommands:   c.BuildCommands,
		OutputDirectory: c.OutputDirectory,
		Package:         c.Package,
	}, nil
}

// CommandForEnvironment returns a named environment override when present,
// otherwise retaining the legacy build_command fallback.
func (c Component) CommandForEnvironment(environment string) string {
	if command := c.BuildCommands[strings.TrimSpace(environment)]; strings.TrimSpace(command) != "" {
		return command
	}
	return c.BuildCommand
}

// ValidationIssue identifies a configuration field that needs attention.
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
