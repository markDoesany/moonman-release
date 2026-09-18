package models

import "time"

// RunSummary is the durable, user-facing history record for a release action.
// It deliberately stores the approved naming plan so reruns can be reviewed
// without relying on the current project configuration.
type RunSummary struct {
	RunID                string            `json:"runId"`
	ProjectID            string            `json:"projectId"`
	ProjectName          string            `json:"projectName"`
	Environment          string            `json:"environment,omitempty"`
	Operation            string            `json:"operation"`
	Version              string            `json:"version,omitempty"`
	ComponentIDs         []string          `json:"componentIds"`
	StartTime            time.Time         `json:"startTime"`
	EndTime              time.Time         `json:"endTime,omitempty"`
	Status               string            `json:"status"`
	ErrorSummary         string            `json:"errorSummary,omitempty"`
	FilenameTemplate     string            `json:"filenameTemplate,omitempty"`
	ApprovedPackageNames map[string]string `json:"approvedPackageNames,omitempty"`
	ReleaseDirectory     string            `json:"releaseDirectory,omitempty"`
	RetryOfRunID         string            `json:"retryOfRunId,omitempty"`
	Attempt              int               `json:"attempt,omitempty"`
	RetryStage           string            `json:"retryStage,omitempty"`
}
