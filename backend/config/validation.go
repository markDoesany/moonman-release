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
		environmentIDs := make(map[string]bool)
		for environmentIndex, environment := range project.Environments {
			environmentPrefix := projectPrefix + ".environments[" + strconv.Itoa(environmentIndex) + "]"
			id := strings.TrimSpace(environment.ID)
			if id == "" {
				issues = append(issues, models.ValidationIssue{Field: environmentPrefix + ".id", Message: "environment ID cannot be empty"})
			} else if environmentIDs[id] {
				issues = append(issues, models.ValidationIssue{Field: environmentPrefix + ".id", Message: "environment ID must be unique within the project"})
			}
			environmentIDs[id] = true
			if strings.TrimSpace(environment.Name) == "" {
				issues = append(issues, models.ValidationIssue{Field: environmentPrefix + ".name", Message: "environment name cannot be empty"})
			}
		}
		if strings.TrimSpace(project.DefaultEnvironment) != "" && len(project.Environments) > 0 && !environmentIDs[strings.TrimSpace(project.DefaultEnvironment)] {
			issues = append(issues, models.ValidationIssue{Field: projectPrefix + ".defaultEnvironment", Message: "default environment must reference a configured environment"})
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
			hasBuildCommand := strings.TrimSpace(component.BuildCommand) != ""
			for environment, command := range component.BuildCommands {
				if strings.TrimSpace(environment) == "" {
					issues = append(issues, models.ValidationIssue{Field: componentPrefix + ".buildCommands", Message: "environment names cannot be empty"})
				}
				if strings.TrimSpace(command) != "" {
					hasBuildCommand = true
				}
			}
			if !hasBuildCommand {
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
