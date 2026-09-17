package app

import (
	"context"
	"fmt"

	"release-launcher/backend/config"
	"release-launcher/backend/logging"
	"release-launcher/backend/models"
)

// App is the small Wails-facing facade. Domain logic lives in backend/config.
type App struct {
	config  *config.Service
	logger  *logging.Logger
	loadErr error
}

func New() (*App, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	logger := logging.New(paths.LogFile)
	service := config.NewService(paths, logger)
	return &App{config: service, logger: logger}, nil
}

func (a *App) Startup(_ context.Context) {
	a.logger.Info("Application started")
	if err := a.config.Load(); err != nil {
		a.loadErr = err
	}
}

func (a *App) Shutdown(_ context.Context) {
	a.logger.Info("Application stopped")
}

// GetProjects returns every configured project.
func (a *App) GetProjects() ([]models.Project, error) {
	if a.loadErr != nil {
		return nil, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.Projects()
}

// GetProject returns one project by its stable ID.
func (a *App) GetProject(id string) (models.Project, error) {
	if a.loadErr != nil {
		return models.Project{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.Project(id)
}

// SaveProject creates or updates a project and returns the canonical saved value.
func (a *App) SaveProject(project models.Project) (models.Project, error) {
	if a.loadErr != nil {
		return models.Project{}, fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.SaveProject(project)
}

// DeleteProject deletes a project and all of its component definitions.
func (a *App) DeleteProject(id string) error {
	if a.loadErr != nil {
		return fmt.Errorf("configuration unavailable: %w", a.loadErr)
	}
	return a.config.DeleteProject(id)
}

// ValidateProject validates a project before it is saved.
func (a *App) ValidateProject(project models.Project) []models.ValidationIssue {
	return a.config.Validate([]models.Project{project})
}
