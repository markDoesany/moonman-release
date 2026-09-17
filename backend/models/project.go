package models

// Project describes a releaseable product and its components.
type Project struct {
	ID         string      `yaml:"id" json:"id"`
	Name       string      `yaml:"name" json:"name"`
	Components []Component `yaml:"components" json:"components"`
}

// Component describes the source and future release settings for one component.
type Component struct {
	ID              string `yaml:"id" json:"id"`
	Name            string `yaml:"name" json:"name"`
	Path            string `yaml:"path" json:"path"`
	BuildCommand    string `yaml:"build_command" json:"buildCommand"`
	OutputDirectory string `yaml:"output_directory" json:"outputDirectory"`
	PackageEnabled  bool   `yaml:"package_enabled" json:"packageEnabled"`
	PackageFilename string `yaml:"package_filename" json:"packageFilename"`
}

// ValidationIssue identifies a configuration field that needs attention.
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
