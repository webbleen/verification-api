package models

// ProjectConfig represents project configuration
type ProjectConfig struct {
	ProjectID    string                 `json:"project_id"`
	ProjectName  string                 `json:"project_name"`
	APIKey       string                 `json:"api_key"`
	FromEmail    string                 `json:"from_email"`
	FromName     string                 `json:"from_name"`
	TemplateID   string                 `json:"template_id,omitempty"`
	CustomConfig map[string]interface{} `json:"custom_config,omitempty"`
	IsActive     bool                   `json:"is_active"`
	CreatedAt    int64                  `json:"created_at"`
	UpdatedAt    int64                  `json:"updated_at"`
}

// ProjectManager manages multiple projects using database
type ProjectManager struct {
	// Keep in-memory cache for performance
	projects map[string]*ProjectConfig
}

// NewProjectManager creates a new project manager
func NewProjectManager() *ProjectManager {
	return &ProjectManager{
		projects: make(map[string]*ProjectConfig),
	}
}

// LoadProjectsFromDB loads all projects from database
func (pm *ProjectManager) LoadProjectsFromDB() error {
	// This will be implemented when we update the middleware
	// For now, keep the existing in-memory logic
	return nil
}

// AddProject adds a new project
func (pm *ProjectManager) AddProject(config *ProjectConfig) {
	pm.projects[config.ProjectID] = config
}

// GetProject gets project configuration by ID
func (pm *ProjectManager) GetProject(projectID string) (*ProjectConfig, bool) {
	project, exists := pm.projects[projectID]
	return project, exists
}

// ValidateProject validates project API key
func (pm *ProjectManager) ValidateProject(projectID, apiKey string) bool {
	project, exists := pm.GetProject(projectID)
	if !exists {
		return false
	}
	return project.APIKey == apiKey && project.IsActive
}

// GetAllProjects returns all projects
func (pm *ProjectManager) GetAllProjects() map[string]*ProjectConfig {
	return pm.projects
}
