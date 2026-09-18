package models

import "time"

// ReleaseRunStatus is the lifecycle state of a build-and-package run.
type ReleaseRunStatus string

const (
	ReleaseRunStatusRunning   ReleaseRunStatus = "running"
	ReleaseRunStatusCompleted ReleaseRunStatus = "completed"
	ReleaseRunStatusFailed    ReleaseRunStatus = "failed"
	ReleaseRunStatusCancelled ReleaseRunStatus = "cancelled"
)

// ReleaseComponentState combines build and package state for one component.
type ReleaseComponentState struct {
	ComponentID     string          `json:"componentId"`
	ComponentName   string          `json:"componentName"`
	Selected        bool            `json:"selected"`
	BuildStatus     BuildStatus     `json:"buildStatus"`
	BuildMessage    string          `json:"buildMessage"`
	BuildResult     *BuildResult    `json:"buildResult,omitempty"`
	PackageStatus   PackageStatus   `json:"packageStatus"`
	PackageMessage  string          `json:"packageMessage"`
	PackageResult   *PackageResult  `json:"packageResult,omitempty"`
	TransferStatus  TransferStatus  `json:"transferStatus"`
	TransferMessage string          `json:"transferMessage"`
	TransferResult  *TransferResult `json:"transferResult,omitempty"`
}

// ReleaseRun describes an asynchronous build-and-package pipeline run.
type ReleaseRun struct {
	ID          string                  `json:"id"`
	ProjectID   string                  `json:"projectId"`
	ProjectName string                  `json:"projectName"`
	Environment string                  `json:"environment,omitempty"`
	Version     string                  `json:"version"`
	Status      ReleaseRunStatus        `json:"status"`
	Components  []ReleaseComponentState `json:"components"`
	StartTime   time.Time               `json:"startTime"`
	EndTime     time.Time               `json:"endTime,omitempty"`
	Error       string                  `json:"error,omitempty"`
}
