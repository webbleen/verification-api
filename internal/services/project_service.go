package services

import (
	"fmt"
	"verification-api/internal/database"
	"verification-api/internal/models"

	"gorm.io/gorm"
)

// ProjectService provides project management operations
type ProjectService struct {
	db *gorm.DB
}

// NewProjectService creates a new project service
func NewProjectService() *ProjectService {
	return &ProjectService{
		db: database.GetDB(),
	}
}

// GetProjectByID gets project by ID
func (s *ProjectService) GetProjectByID(projectID string) (*models.Project, error) {
	var project models.Project
	result := s.db.Where("project_id = ? AND is_active = ?", projectID, true).First(&project)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project not found")
		}
		return nil, result.Error
	}
	return &project, nil
}

// GetProjectByAPIKey gets project by API key
func (s *ProjectService) GetProjectByAPIKey(apiKey string) (*models.Project, error) {
	var project models.Project
	result := s.db.Where("api_key = ? AND is_active = ?", apiKey, true).First(&project)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project not found")
		}
		return nil, result.Error
	}
	return &project, nil
}

// ValidateProject validates project ID and API key
func (s *ProjectService) ValidateProject(projectID, apiKey string) bool {
	project, err := s.GetProjectByID(projectID)
	if err != nil {
		return false
	}
	return project.APIKey == apiKey && project.IsActive
}

// GetProjectByBundleID gets project by bundle ID (iOS App identification)
func (s *ProjectService) GetProjectByBundleID(bundleID string) (*models.Project, error) {
	var project models.Project
	result := s.db.Where("bundle_id = ? AND is_active = ?", bundleID, true).First(&project)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project not found for bundle_id: %s", bundleID)
		}
		return nil, result.Error
	}
	return &project, nil
}

// GetProjectByPackageName gets project by package name (Android App identification)
func (s *ProjectService) GetProjectByPackageName(packageName string) (*models.Project, error) {
	var project models.Project
	result := s.db.Where("package_name = ? AND is_active = ?", packageName, true).First(&project)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project not found for package_name: %s", packageName)
		}
		return nil, result.Error
	}
	return &project, nil
}

// GetAllProjects gets all active projects
func (s *ProjectService) GetAllProjects() ([]*models.Project, error) {
	var projects []*models.Project
	result := s.db.Where("is_active = ?", true).Find(&projects)
	if result.Error != nil {
		return nil, result.Error
	}
	return projects, nil
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(project *models.Project) error {
	// Check if project ID already exists
	var existingProject models.Project
	result := s.db.Where("project_id = ?", project.ProjectID).First(&existingProject)
	if result.Error == nil {
		return fmt.Errorf("project with ID %s already exists", project.ProjectID)
	}

	// Check if API key already exists
	result = s.db.Where("api_key = ?", project.APIKey).First(&existingProject)
	if result.Error == nil {
		return fmt.Errorf("project with API key already exists")
	}

	// Create project
	if err := s.db.Create(project).Error; err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// UpdateProject updates an existing project
func (s *ProjectService) UpdateProject(projectID string, updates map[string]interface{}) error {
	result := s.db.Model(&models.Project{}).Where("project_id = ?", projectID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("project not found")
	}
	return nil
}

// DeleteProject soft deletes a project
func (s *ProjectService) DeleteProject(projectID string) error {
	result := s.db.Where("project_id = ?", projectID).Delete(&models.Project{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("project not found")
	}
	return nil
}

// GetProjectStats gets project statistics
// Note: Statistics removed - using Redis only, no persistent logging
func (s *ProjectService) GetProjectStats(projectID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	// No statistics available - verification codes are stored in Redis only
	return stats, nil
}
