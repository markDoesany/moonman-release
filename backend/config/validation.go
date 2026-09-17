package config

import (
	"strconv"
	"strings"

	"release-launcher/backend/models"
)

// Validate returns all basic configuration issues without checking whether paths exist.
func Validate(projects []models.Project) []models.ValidationIssue {
	issues := make([]models.ValidationIssue, 0)
	projectIDs := make(map[string]bool)
	for projectIndex, project := range projects {
		projectPrefix := "projects[" + strconv.Itoa(projectIndex) + "]"
		if strings.TrimSpace(project.ID) == "" {
			issues = append(issues, models.ValidationIssue{Field: projectPrefix + ".id", Message: "project ID cannot be empty"})
		} else if projectIDs[project.ID] {
			issues = append(issues, models.ValidationIssue{Field: projectPrefix + ".id", Message: "project ID must be unique"})
		}
		projectIDs[project.ID] = true
		if strings.TrimSpace(project.Name) == "" {
			issues = append(issues, models.ValidationIssue{Field: projectPrefix + ".name", Message: "project name cannot be empty"})
		}

		componentIDs := make(map[string]bool)
		for componentIndex, component := range project.Components {
			componentPrefix := projectPrefix + ".components[" + strconv.Itoa(componentIndex) + "]"
			if strings.TrimSpace(component.ID) == "" {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".id", Message: "component ID cannot be empty"})
			} else if componentIDs[component.ID] {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".id", Message: "component ID must be unique within the project"})
			}
			componentIDs[component.ID] = true
			if strings.TrimSpace(component.Name) == "" {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".name", Message: "component name cannot be empty"})
			}
			if strings.TrimSpace(component.Path) == "" {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".path", Message: "component path cannot be empty"})
			}
			if strings.TrimSpace(component.BuildCommand) == "" {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".buildCommand", Message: "build command cannot be empty"})
			}
			if strings.TrimSpace(component.OutputDirectory) == "" {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".outputDirectory", Message: "output directory cannot be empty"})
			}
			if component.Package.Enabled && strings.TrimSpace(component.Package.Filename) == "" {
				issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".package.filename", Message: "package filename is required when packaging is enabled"})
			}
		}
	}
	return issues
}
