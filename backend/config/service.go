package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
	"release-launcher/backend/logging"
	"release-launcher/backend/models"
)

// Paths contains all application-local persistent paths.
type Paths struct {
	BaseDir         string
	ConfigDir       string
	ConfigFile      string
	BackupFile      string
	LogDir          string
	LogFile         string
	BuildLogFile    string
	PackageLogFile  string
	TransferLogFile string
	ActivityLogFile string
	ReleaseDir      string
}

// ResolvePaths selects the repository/app directory, with an override for tests and installations.
func ResolvePaths() (Paths, error) {
	baseDir := strings.TrimSpace(os.Getenv("RELEASE_LAUNCHER_DATA_DIR"))
	if baseDir == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return Paths{}, fmt.Errorf("get working directory: %w", err)
		}
		if _, err := os.Stat(filepath.Join(workingDir, "configs")); err == nil {
			baseDir = workingDir
		} else {
			executable, err := os.Executable()
			if err != nil {
				return Paths{}, fmt.Errorf("get executable path: %w", err)
			}
			baseDir = filepath.Dir(executable)
		}
	}

	baseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve application directory: %w", err)
	}
	configDir := filepath.Join(baseDir, "configs")
	logDir := filepath.Join(baseDir, "logs")
	return Paths{
		BaseDir:         baseDir,
		ConfigDir:       configDir,
		ConfigFile:      filepath.Join(configDir, "projects.yaml"),
		BackupFile:      filepath.Join(configDir, "projects.yaml.bak"),
		LogDir:          logDir,
		LogFile:         filepath.Join(logDir, "app.log"),
		BuildLogFile:    filepath.Join(logDir, "builds.jsonl"),
		PackageLogFile:  filepath.Join(logDir, "packaging.jsonl"),
		TransferLogFile: filepath.Join(logDir, "transfers.jsonl"),
		ActivityLogFile: filepath.Join(logDir, "activity.jsonl"),
		ReleaseDir:      filepath.Join(baseDir, "releases"),
	}, nil
}

// Service owns the validated in-memory project configuration and its persistence.
type Service struct {
	paths  Paths
	logger *logging.Logger
	mu     sync.RWMutex
	loaded bool
	issues []models.ValidationIssue
	data   []models.Project
	dali   models.DaliConfig
}

func NewService(paths Paths, logger *logging.Logger) *Service {
	return &Service{paths: paths, logger: logger}
}

// Load reads the primary configuration, initializes missing configuration, or recovers a valid backup.
func (s *Service) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.paths.ConfigDir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	contents, err := os.ReadFile(s.paths.ConfigFile)
	if errors.Is(err, os.ErrNotExist) {
		projects := []models.Project{}
		dali := models.DefaultDaliConfig()
		s.data = projects
		s.dali = dali
		s.loaded = true
		if err := s.writeLocked(projects, dali, false); err != nil {
			return fmt.Errorf("initialize configuration: %w", err)
		}
		s.logger.Info("Configuration initialized with no projects")
		s.logger.Info("Configuration loaded")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read configuration: %w", err)
	}

	projects, dali, primaryErr := parseAndValidate(contents)
	if primaryErr == nil {
		s.data = projects
		s.dali = dali
		s.issues = nil
		s.loaded = true
		s.logger.Info("Configuration loaded")
		return nil
	}

	backup, backupErr := os.ReadFile(s.paths.BackupFile)
	if backupErr == nil {
		backupProjects, backupDali, validBackupErr := parseAndValidate(backup)
		if validBackupErr == nil {
			s.data = backupProjects
			s.dali = backupDali
			s.loaded = true
			if restoreErr := s.replaceFileLocked(backup, false); restoreErr != nil {
				return fmt.Errorf("restore configuration backup: %w", restoreErr)
			}
			s.logger.Error("Configuration validation failed; restored backup")
			s.logger.Info("Configuration loaded")
			return nil
		}
	}

	s.loaded = false
	s.issues = []models.ValidationIssue{{Field: "configuration", Message: primaryErr.Error()}}
	s.logger.Error("Configuration validation failed: " + primaryErr.Error())
	return fmt.Errorf("configuration is invalid: %w", primaryErr)
}

func (s *Service) Projects() ([]models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.loaded {
		return nil, s.loadErrorLocked()
	}
	return cloneProjects(s.data), nil
}

func (s *Service) Project(id string) (models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.loaded {
		return models.Project{}, s.loadErrorLocked()
	}
	for _, project := range s.data {
		if project.ID == id {
			return cloneProject(project), nil
		}
	}
	return models.Project{}, fmt.Errorf("project %q not found", id)
}

// DaliConfig returns the current Dali transfer settings.
func (s *Service) DaliConfig() (models.DaliConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.loaded {
		return models.DaliConfig{}, s.loadErrorLocked()
	}
	return s.dali, nil
}

// SaveDaliConfig persists Dali transfer settings alongside project configuration.
func (s *Service) SaveDaliConfig(settings models.DaliConfig) (models.DaliConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		return models.DaliConfig{}, s.loadErrorLocked()
	}
	settings = normalizeDaliConfig(settings)
	if err := validateDaliConfig(settings); err != nil {
		return models.DaliConfig{}, err
	}
	if err := s.writeLocked(s.data, settings, true); err != nil {
		return models.DaliConfig{}, err
	}
	s.dali = settings
	s.logger.Info("Dali configuration saved")
	return settings, nil
}

func (s *Service) SaveProject(project models.Project) (models.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		return models.Project{}, s.loadErrorLocked()
	}

	project = normalizeProject(project, s.data)
	updated := cloneProjects(s.data)
	found := false
	for i := range updated {
		if updated[i].ID == project.ID {
			updated[i] = cloneProject(project)
			found = true
			break
		}
	}
	if !found {
		updated = append(updated, cloneProject(project))
	}
	if issues := Validate(updated); len(issues) > 0 {
		s.issues = issues
		s.logger.Error("Configuration validation failed: " + formatIssues(issues))
		return models.Project{}, errors.New(formatIssues(issues))
	}
	if err := s.writeLocked(updated, s.dali, true); err != nil {
		return models.Project{}, err
	}
	s.data = updated
	s.issues = nil
	s.logger.Info("Configuration saved")
	return cloneProject(project), nil
}

func (s *Service) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loaded {
		return s.loadErrorLocked()
	}

	updated := make([]models.Project, 0, len(s.data))
	deleted := false
	for _, project := range s.data {
		if project.ID == id {
			deleted = true
			continue
		}
		updated = append(updated, cloneProject(project))
	}
	if !deleted {
		return fmt.Errorf("project %q not found", id)
	}
	if issues := Validate(updated); len(issues) > 0 {
		return errors.New(formatIssues(issues))
	}
	if err := s.writeLocked(updated, s.dali, true); err != nil {
		return err
	}
	s.data = updated
	s.logger.Info("Configuration saved")
	return nil
}

func (s *Service) Validate(projects []models.Project) []models.ValidationIssue {
	return Validate(projects)
}

func (s *Service) loadErrorLocked() error {
	if len(s.issues) > 0 {
		return errors.New(formatIssues(s.issues))
	}
	return errors.New("configuration is not loaded")
}

func (s *Service) writeLocked(projects []models.Project, dali models.DaliConfig, keepBackup bool) error {
	contents, err := yaml.Marshal(struct {
		Projects []models.Project  `yaml:"projects"`
		Dali     models.DaliConfig `yaml:"dali"`
	}{Projects: projects, Dali: dali})
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}
	return s.replaceFileLockedWithBackup(contents, keepBackup)
}

func (s *Service) replaceFileLockedWithBackup(contents []byte, keepBackup bool) error {
	if keepBackup {
		if previous, readErr := os.ReadFile(s.paths.ConfigFile); readErr == nil {
			if err := os.WriteFile(s.paths.BackupFile, previous, 0o644); err != nil {
				return fmt.Errorf("write configuration backup: %w", err)
			}
		}
	}
	temporary := s.paths.ConfigFile + ".tmp"
	if err := os.WriteFile(temporary, contents, 0o644); err != nil {
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := os.Remove(s.paths.ConfigFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace configuration: %w", err)
	}
	if err := os.Rename(temporary, s.paths.ConfigFile); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("activate configuration: %w", err)
	}
	return nil
}

func (s *Service) replaceFileLocked(contents []byte, keepBackup bool) error {
	return s.replaceFileLockedWithBackup(contents, keepBackup)
}

func parseAndValidate(contents []byte) ([]models.Project, models.DaliConfig, error) {
	var document struct {
		Projects []models.Project   `yaml:"projects"`
		Dali     *models.DaliConfig `yaml:"dali"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	if err := decoder.Decode(&document); err != nil {
		return nil, models.DaliConfig{}, fmt.Errorf("parse YAML: %w", err)
	}
	if issues := Validate(document.Projects); len(issues) > 0 {
		return nil, models.DaliConfig{}, errors.New(formatIssues(issues))
	}
	dali := models.DefaultDaliConfig()
	if document.Dali != nil {
		dali = normalizeDaliConfig(*document.Dali)
	}
	if err := validateDaliConfig(dali); err != nil {
		return nil, models.DaliConfig{}, err
	}
	return document.Projects, dali, nil
}

func normalizeDaliConfig(settings models.DaliConfig) models.DaliConfig {
	settings.Executable = strings.TrimSpace(settings.Executable)
	if settings.Executable == "" {
		settings.Executable = models.DefaultDaliConfig().Executable
	}
	settings.PeerName = strings.TrimSpace(settings.PeerName)
	settings.PeerAddress = strings.TrimSpace(settings.PeerAddress)
	return settings
}

func validateDaliConfig(settings models.DaliConfig) error {
	if strings.TrimSpace(settings.Executable) == "" {
		return errors.New("dali executable cannot be empty")
	}
	if strings.TrimSpace(settings.PeerName) != "" && strings.TrimSpace(settings.PeerAddress) != "" {
		return errors.New("configure either a Dali peer name or peer address, not both")
	}
	if !settings.Auto && strings.TrimSpace(settings.PeerName) == "" && strings.TrimSpace(settings.PeerAddress) == "" {
		return errors.New("configure a Dali peer name/address or enable automatic peer selection")
	}
	return nil
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeProject(project models.Project, existing []models.Project) models.Project {
	isNew := strings.TrimSpace(project.ID) == ""
	project.Name = strings.TrimSpace(project.Name)
	if strings.TrimSpace(project.ID) == "" {
		project.ID = uniqueID(slug(project.Name), project.ID, projectIDs(existing))
	}
	if isNew && len(project.Environments) == 0 {
		project.Environments = cloneEnvironments(models.DefaultEnvironmentProfiles)
	}
	project.DefaultEnvironment = strings.TrimSpace(project.DefaultEnvironment)
	if project.DefaultEnvironment == "" && len(project.Environments) > 0 {
		project.DefaultEnvironment = project.Environments[0].ID
	}
	for i := range project.Environments {
		project.Environments[i].ID = strings.TrimSpace(project.Environments[i].ID)
		project.Environments[i].Name = strings.TrimSpace(project.Environments[i].Name)
		if project.Environments[i].Commands == nil {
			project.Environments[i].Commands = map[string]string{}
		}
	}
	usedComponentIDs := make(map[string]bool)
	for _, component := range project.Components {
		usedComponentIDs[component.ID] = true
	}
	for i := range project.Components {
		component := &project.Components[i]
		component.Name = strings.TrimSpace(component.Name)
		if strings.TrimSpace(component.ID) == "" {
			component.ID = uniqueID(slug(component.Name), component.ID, usedComponentIDs)
		}
		usedComponentIDs[component.ID] = true
	}
	return project
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func uniqueID(candidate, current string, used map[string]bool) string {
	if candidate == "" {
		candidate = "item"
	}
	if current != "" {
		return current
	}
	base := candidate
	for i := 2; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return candidate
}

func projectIDs(projects []models.Project) map[string]bool {
	used := make(map[string]bool, len(projects))
	for _, project := range projects {
		used[project.ID] = true
	}
	return used
}

func cloneProjects(projects []models.Project) []models.Project {
	result := make([]models.Project, len(projects))
	for i, project := range projects {
		result[i] = cloneProject(project)
	}
	return result
}

func cloneProject(project models.Project) models.Project {
	project.Components = append([]models.Component(nil), project.Components...)
	project.Environments = cloneEnvironments(project.Environments)
	for i := range project.Components {
		project.Components[i].BuildCommands = cloneStringMap(project.Components[i].BuildCommands)
	}
	return project
}

func cloneEnvironments(environments []models.EnvironmentProfile) []models.EnvironmentProfile {
	result := make([]models.EnvironmentProfile, len(environments))
	for i, environment := range environments {
		result[i] = environment
		result[i].Commands = cloneStringMap(environment.Commands)
	}
	return result
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func formatIssues(issues []models.ValidationIssue) string {
	parts := make([]string, len(issues))
	for i, issue := range issues {
		parts[i] = issue.Field + ": " + issue.Message
	}
	return strings.Join(parts, "; ")
}

func issueField(index int, field string) string {
	return "projects[" + strconv.Itoa(index) + "]." + field
}
