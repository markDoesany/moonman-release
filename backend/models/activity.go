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
	ComponentNames       []string          `json:"componentNames,omitempty"`
	PackagePaths         []string          `json:"packagePaths,omitempty"`
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
	Command              string            `json:"command"`
	DaliCommands         map[string]string `json:"daliCommands,omitempty"`
}

type ActivityQuery struct {
	ProjectID   string `json:"projectId,omitempty"`
	Environment string `json:"environment,omitempty"`
	Status      string `json:"status,omitempty"`
	Search      string `json:"search,omitempty"`
	Page        int    `json:"page,omitempty"`
	PageSize    int    `json:"pageSize,omitempty"`
}

type ActivityPage struct {
	Runs       []RunSummary `json:"runs"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	Total      int          `json:"total"`
	TotalPages int          `json:"totalPages"`
}
