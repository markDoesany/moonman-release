package models

import "gopkg.in/yaml.v3"

// Project describes a releaseable product and its components.
type Project struct {
	ID         string      `yaml:"id" json:"id"`
	Name       string      `yaml:"name" json:"name"`
	Components []Component `yaml:"components" json:"components"`
}

// Component describes the source and future release settings for one component.
type Component struct {
	ID              string        `yaml:"id" json:"id"`
	Name            string        `yaml:"name" json:"name"`
	Path            string        `yaml:"path" json:"path"`
	BuildCommand    string        `yaml:"build_command" json:"buildCommand"`
	OutputDirectory string        `yaml:"output_directory" json:"outputDirectory"`
	Package         PackageConfig `yaml:"package" json:"package"`
}

// PackageConfig controls whether a component's build output is archived.
type PackageConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Filename string `yaml:"filename" json:"filename"`
}

// UnmarshalYAML accepts both the Phase 3 nested package format and the Phase 1
// flat fields so existing project files continue to load safely.
func (c *Component) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		ID              string         `yaml:"id"`
		Name            string         `yaml:"name"`
		Path            string         `yaml:"path"`
		BuildCommand    string         `yaml:"build_command"`
		OutputDirectory string         `yaml:"output_directory"`
		Package         *PackageConfig `yaml:"package"`
		PackageEnabled  *bool          `yaml:"package_enabled"`
		PackageFilename string         `yaml:"package_filename"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	c.ID = raw.ID
	c.Name = raw.Name
	c.Path = raw.Path
	c.BuildCommand = raw.BuildCommand
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
		ID              string        `yaml:"id"`
		Name            string        `yaml:"name"`
		Path            string        `yaml:"path"`
		BuildCommand    string        `yaml:"build_command"`
		OutputDirectory string        `yaml:"output_directory"`
		Package         PackageConfig `yaml:"package"`
	}{
		ID:              c.ID,
		Name:            c.Name,
		Path:            c.Path,
		BuildCommand:    c.BuildCommand,
		OutputDirectory: c.OutputDirectory,
		Package:         c.Package,
	}, nil
}

// ValidationIssue identifies a configuration field that needs attention.
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
