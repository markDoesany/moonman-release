package models

import "time"

// DaliConfig controls how release archives are sent to DevOps.
type DaliConfig struct {
	Executable  string `yaml:"executable" json:"executable"`
	PeerName    string `yaml:"peer_name" json:"peerName"`
	PeerAddress string `yaml:"peer_address" json:"peerAddress"`
	Auto        bool   `yaml:"auto" json:"auto"`
	Wait        bool   `yaml:"wait" json:"wait"`
}

// DefaultDaliConfig is safe for a one-peer local network and matches the Dali CLI name.
func DefaultDaliConfig() DaliConfig {
	return DaliConfig{Executable: "dali", Auto: true}
}

type TransferStatus string

const (
	TransferStatusReady     TransferStatus = "ready"
	TransferStatusSending   TransferStatus = "sending"
	TransferStatusSuccess   TransferStatus = "success"
	TransferStatusFailed    TransferStatus = "failed"
	TransferStatusSkipped   TransferStatus = "skipped"
	TransferStatusCancelled TransferStatus = "cancelled"
)

type TransferRunStatus string

const (
	TransferRunStatusRunning   TransferRunStatus = "running"
	TransferRunStatusCompleted TransferRunStatus = "completed"
	TransferRunStatusFailed    TransferRunStatus = "failed"
	TransferRunStatusCancelled TransferRunStatus = "cancelled"
)

// TransferRequest identifies packaged artifacts to send.
type TransferRequest struct {
	ProjectID        string            `json:"projectId"`
	ComponentIDs     []string          `json:"componentIds"`
	Version          string            `json:"version"`
	FilenameTemplate string            `json:"filenameTemplate,omitempty"`
	PackageNames     map[string]string `json:"packageNames,omitempty"`
	Environment      string            `json:"environment,omitempty"`
	ReleaseDirectory string            `json:"releaseDirectory,omitempty"`
}

// TransferPlanItem describes the archive that will be sent.
type TransferPlanItem struct {
	ComponentID      string `json:"componentId"`
	ComponentName    string `json:"componentName"`
	Selected         bool   `json:"selected"`
	Enabled          bool   `json:"enabled"`
	PackagePath      string `json:"packagePath"`
	FilenameTemplate string `json:"filenameTemplate,omitempty"`
	ResolvedFilename string `json:"resolvedFilename,omitempty"`
	Exists           bool   `json:"exists"`
	Error            string `json:"error,omitempty"`
}

type TransferPlan struct {
	ProjectID        string             `json:"projectId"`
	ProjectName      string             `json:"projectName"`
	Version          string             `json:"version"`
	ReleaseDirectory string             `json:"releaseDirectory"`
	FilenameTemplate string             `json:"filenameTemplate"`
	Components       []TransferPlanItem `json:"components"`
	HasMissing       bool               `json:"hasMissing"`
}

type TransferComponentState struct {
	ComponentID   string          `json:"componentId"`
	ComponentName string          `json:"componentName"`
	Selected      bool            `json:"selected"`
	Status        TransferStatus  `json:"status"`
	Message       string          `json:"message"`
	Result        *TransferResult `json:"result,omitempty"`
}

type TransferRun struct {
	ID          string                   `json:"id"`
	ProjectID   string                   `json:"projectId"`
	ProjectName string                   `json:"projectName"`
	Environment string                   `json:"environment,omitempty"`
	Version     string                   `json:"version"`
	Status      TransferRunStatus        `json:"status"`
	Components  []TransferComponentState `json:"components"`
	StartTime   time.Time                `json:"startTime"`
	EndTime     time.Time                `json:"endTime,omitempty"`
	Error       string                   `json:"error,omitempty"`
}

type TransferResult struct {
	ProjectID        string         `json:"projectId"`
	ProjectName      string         `json:"projectName"`
	ComponentID      string         `json:"componentId"`
	ComponentName    string         `json:"componentName"`
	Success          bool           `json:"success"`
	Status           TransferStatus `json:"status"`
	PackagePath      string         `json:"packagePath"`
	Version          string         `json:"version"`
	FilenameTemplate string         `json:"filenameTemplate,omitempty"`
	ResolvedFilename string         `json:"resolvedFilename,omitempty"`
	Executable       string         `json:"executable"`
	Arguments        []string       `json:"arguments"`
	PeerName         string         `json:"peerName,omitempty"`
	PeerAddress      string         `json:"peerAddress,omitempty"`
	ExitCode         int            `json:"exitCode"`
	StartTime        time.Time      `json:"startTime"`
	EndTime          time.Time      `json:"endTime"`
	DurationMs       int64          `json:"durationMs"`
	Error            string         `json:"error,omitempty"`
}

type TransferRecord struct {
	Timestamp        time.Time      `json:"timestamp"`
	RunID            string         `json:"runId"`
	ProjectID        string         `json:"projectId"`
	ProjectName      string         `json:"projectName"`
	ComponentID      string         `json:"componentId"`
	ComponentName    string         `json:"componentName"`
	PackagePath      string         `json:"packagePath"`
	Version          string         `json:"version"`
	FilenameTemplate string         `json:"filenameTemplate,omitempty"`
	ResolvedFilename string         `json:"resolvedFilename,omitempty"`
	Executable       string         `json:"executable"`
	Arguments        []string       `json:"arguments"`
	PeerName         string         `json:"peerName,omitempty"`
	PeerAddress      string         `json:"peerAddress,omitempty"`
	ExitCode         int            `json:"exitCode"`
	StartTime        time.Time      `json:"startTime"`
	EndTime          time.Time      `json:"endTime"`
	DurationMs       int64          `json:"durationMs"`
	Success          bool           `json:"success"`
	Status           TransferStatus `json:"status"`
	Error            string         `json:"error,omitempty"`
	TechnicalError   string         `json:"technicalError,omitempty"`
}
