package models

import "time"

// BuildStatus is the lifecycle state of a component in a build run.
type BuildStatus string

const (
	BuildStatusReady     BuildStatus = "ready"
	BuildStatusBuilding  BuildStatus = "building"
	BuildStatusSuccess   BuildStatus = "success"
	BuildStatusFailed    BuildStatus = "failed"
	BuildStatusSkipped   BuildStatus = "skipped"
	BuildStatusCancelled BuildStatus = "cancelled"
)

// BuildRunStatus is the lifecycle state of the complete build run.
type BuildRunStatus string

const (
	BuildRunStatusRunning   BuildRunStatus = "running"
	BuildRunStatusCompleted BuildRunStatus = "completed"
	BuildRunStatusFailed    BuildRunStatus = "failed"
	BuildRunStatusCancelled BuildRunStatus = "cancelled"
)

// BuildRequest identifies the project and components to build.
type BuildRequest struct {
	ProjectID    string   `json:"projectId"`
	ComponentIDs []string `json:"componentIds"`
}

// BuildComponentState is the frontend-facing state for one component.
type BuildComponentState struct {
	ComponentID   string       `json:"componentId"`
	ComponentName string       `json:"componentName"`
	Selected      bool         `json:"selected"`
	Status        BuildStatus  `json:"status"`
	Message       string       `json:"message"`
	Result        *BuildResult `json:"result,omitempty"`
}

// BuildRun describes an asynchronous build run and its component states.
type BuildRun struct {
	ID          string                `json:"id"`
	ProjectID   string                `json:"projectId"`
	ProjectName string                `json:"projectName"`
	Status      BuildRunStatus        `json:"status"`
	Components  []BuildComponentState `json:"components"`
	StartTime   time.Time             `json:"startTime"`
	EndTime     time.Time             `json:"endTime,omitempty"`
	Error       string                `json:"error,omitempty"`
}

// BuildResult contains the outcome and artifact information for one attempt.
type BuildResult struct {
	ProjectID       string      `json:"projectId"`
	ProjectName     string      `json:"projectName"`
	ComponentID     string      `json:"componentId"`
	ComponentName   string      `json:"componentName"`
	Success         bool        `json:"success"`
	Status          BuildStatus `json:"status"`
	ExitCode        int         `json:"exitCode"`
	OutputDirectory string      `json:"outputDirectory"`
	OutputPath      string      `json:"outputPath"`
	StartTime       time.Time   `json:"startTime"`
	EndTime         time.Time   `json:"endTime"`
	DurationMs      int64       `json:"durationMs"`
	Error           string      `json:"error,omitempty"`
}

// BuildEvent is emitted through Wails while a run is active.
type BuildEvent struct {
	Type           string          `json:"type"`
	Phase          string          `json:"phase,omitempty"`
	RunID          string          `json:"runId"`
	ProjectID      string          `json:"projectId"`
	ProjectName    string          `json:"projectName"`
	ComponentID    string          `json:"componentId,omitempty"`
	ComponentName  string          `json:"componentName,omitempty"`
	Status         BuildStatus     `json:"status,omitempty"`
	RunStatus      BuildRunStatus  `json:"runStatus,omitempty"`
	Stream         string          `json:"stream,omitempty"`
	Text           string          `json:"text,omitempty"`
	Timestamp      time.Time       `json:"timestamp"`
	Version        string          `json:"version,omitempty"`
	Result         *BuildResult    `json:"result,omitempty"`
	Results        []BuildResult   `json:"results,omitempty"`
	PackageStatus  PackageStatus   `json:"packageStatus,omitempty"`
	PackageResult  *PackageResult  `json:"packageResult,omitempty"`
	PackageResults []PackageResult `json:"packageResults,omitempty"`
	Error          string          `json:"error,omitempty"`
}

// BuildRecord is the structured local history entry for one build attempt.
type BuildRecord struct {
	Timestamp       time.Time   `json:"timestamp"`
	RunID           string      `json:"runId"`
	ProjectID       string      `json:"projectId"`
	ProjectName     string      `json:"projectName"`
	ComponentID     string      `json:"componentId"`
	ComponentName   string      `json:"componentName"`
	BuildCommand    string      `json:"buildCommand"`
	ProjectPath     string      `json:"projectPath"`
	StartTime       time.Time   `json:"startTime"`
	EndTime         time.Time   `json:"endTime"`
	DurationMs      int64       `json:"durationMs"`
	ExitCode        int         `json:"exitCode"`
	Success         bool        `json:"success"`
	Status          BuildStatus `json:"status"`
	OutputDirectory string      `json:"outputDirectory"`
	OutputPath      string      `json:"outputPath"`
	Error           string      `json:"error,omitempty"`
	TechnicalError  string      `json:"technicalError,omitempty"`
}
