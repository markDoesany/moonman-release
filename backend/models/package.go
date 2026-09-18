package models

import "time"

// PackageStatus is the lifecycle state of a package operation.
type PackageStatus string

const (
	PackageStatusReady     PackageStatus = "ready"
	PackageStatusPackaging PackageStatus = "packaging"
	PackageStatusSuccess   PackageStatus = "success"
	PackageStatusFailed    PackageStatus = "failed"
	PackageStatusSkipped   PackageStatus = "skipped"
	PackageStatusCancelled PackageStatus = "cancelled"
)

// PackageRunStatus is the lifecycle state of a sequential package run.
type PackageRunStatus string

const (
	PackageRunStatusRunning   PackageRunStatus = "running"
	PackageRunStatusCompleted PackageRunStatus = "completed"
	PackageRunStatusFailed    PackageRunStatus = "failed"
	PackageRunStatusCancelled PackageRunStatus = "cancelled"
)

// PackageRequest identifies a project, components, and release version.
type PackageRequest struct {
	ProjectID        string            `json:"projectId"`
	ComponentIDs     []string          `json:"componentIds"`
	Version          string            `json:"version"`
	Overwrite        bool              `json:"overwrite"`
	FilenameTemplate string            `json:"filenameTemplate,omitempty"`
	PackageNames     map[string]string `json:"packageNames,omitempty"`
}

// PackagePlanItem describes one potential archive and any overwrite conflict.
type PackagePlanItem struct {
	ComponentID      string `json:"componentId"`
	ComponentName    string `json:"componentName"`
	Selected         bool   `json:"selected"`
	Enabled          bool   `json:"enabled"`
	SourcePath       string `json:"sourcePath"`
	PackagePath      string `json:"packagePath"`
	FilenameTemplate string `json:"filenameTemplate,omitempty"`
	ResolvedFilename string `json:"resolvedFilename,omitempty"`
	Existing         bool   `json:"existing"`
	Error            string `json:"error,omitempty"`
}

// PackagePlan is returned before packaging so the UI can request overwrite confirmation.
type PackagePlan struct {
	ProjectID        string            `json:"projectId"`
	ProjectName      string            `json:"projectName"`
	Version          string            `json:"version"`
	ReleaseDirectory string            `json:"releaseDirectory"`
	FilenameTemplate string            `json:"filenameTemplate"`
	Components       []PackagePlanItem `json:"components"`
	HasConflicts     bool              `json:"hasConflicts"`
}

// PackageComponentState is the frontend-facing state for one package operation.
type PackageComponentState struct {
	ComponentID   string         `json:"componentId"`
	ComponentName string         `json:"componentName"`
	Selected      bool           `json:"selected"`
	Status        PackageStatus  `json:"status"`
	Message       string         `json:"message"`
	Result        *PackageResult `json:"result,omitempty"`
}

// PackageRun describes an asynchronous sequential package run.
type PackageRun struct {
	ID          string                  `json:"id"`
	ProjectID   string                  `json:"projectId"`
	ProjectName string                  `json:"projectName"`
	Version     string                  `json:"version"`
	Status      PackageRunStatus        `json:"status"`
	Components  []PackageComponentState `json:"components"`
	StartTime   time.Time               `json:"startTime"`
	EndTime     time.Time               `json:"endTime,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

// PackageResult contains the outcome and validated archive information.
type PackageResult struct {
	ProjectID        string        `json:"projectId"`
	ProjectName      string        `json:"projectName"`
	ComponentID      string        `json:"componentId"`
	ComponentName    string        `json:"componentName"`
	Success          bool          `json:"success"`
	Status           PackageStatus `json:"status"`
	SourcePath       string        `json:"sourcePath"`
	PackagePath      string        `json:"packagePath"`
	Version          string        `json:"version"`
	FilenameTemplate string        `json:"filenameTemplate,omitempty"`
	ResolvedFilename string        `json:"resolvedFilename,omitempty"`
	SizeBytes        int64         `json:"sizeBytes"`
	StartTime        time.Time     `json:"startTime"`
	EndTime          time.Time     `json:"endTime"`
	DurationMs       int64         `json:"durationMs"`
	Error            string        `json:"error,omitempty"`
}

// PackageRecord is the structured local history entry for one package attempt.
type PackageRecord struct {
	Timestamp        time.Time     `json:"timestamp"`
	RunID            string        `json:"runId"`
	ProjectID        string        `json:"projectId"`
	ProjectName      string        `json:"projectName"`
	ComponentID      string        `json:"componentId"`
	ComponentName    string        `json:"componentName"`
	SourcePath       string        `json:"sourcePath"`
	PackagePath      string        `json:"packagePath"`
	Version          string        `json:"version"`
	FilenameTemplate string        `json:"filenameTemplate,omitempty"`
	ResolvedFilename string        `json:"resolvedFilename,omitempty"`
	StartTime        time.Time     `json:"startTime"`
	EndTime          time.Time     `json:"endTime"`
	DurationMs       int64         `json:"durationMs"`
	SizeBytes        int64         `json:"sizeBytes"`
	Success          bool          `json:"success"`
	Status           PackageStatus `json:"status"`
	Error            string        `json:"error,omitempty"`
	TechnicalError   string        `json:"technicalError,omitempty"`
}
