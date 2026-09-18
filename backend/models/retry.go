package models

// RetryStage identifies the failed part of a run that should be attempted
// again. Retries are deliberately explicit; the backend never retries on its
// own.
type RetryStage string

const (
	RetryStageBuild    RetryStage = "build"
	RetryStagePackage  RetryStage = "package"
	RetryStageTransfer RetryStage = "transfer"
)

type RetryRequest struct {
	RunID        string     `json:"runId"`
	Stage        RetryStage `json:"stage"`
	ComponentIDs []string   `json:"componentIds,omitempty"`
}

// RetryRun is a typed envelope for the Wails API. Only the field matching
// Operation is populated.
type RetryRun struct {
	Operation    string       `json:"operation"`
	Stage        RetryStage   `json:"stage"`
	Attempt      int          `json:"attempt"`
	RetryOfRunID string       `json:"retryOfRunId"`
	Build        *BuildRun    `json:"build,omitempty"`
	Package      *PackageRun  `json:"package,omitempty"`
	Release      *ReleaseRun  `json:"release,omitempty"`
	Transfer     *TransferRun `json:"transfer,omitempty"`
}
