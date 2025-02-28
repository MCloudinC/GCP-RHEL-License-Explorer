package api

import (
	"context"
	"fmt"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// Project represents a GCP project
type Project struct {
	ID   string
	Name string
}

// ListProjects retrieves all projects accessible to the authenticated user
func ListProjects(ctx context.Context, cloudResourceManagerService *cloudresourcemanager.Service) ([]Project, error) {
	req := cloudResourceManagerService.Projects.List()
	var projects []Project

	if err := req.Pages(ctx, func(page *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range page.Projects {
			projects = append(projects, Project{
				ID:   project.ProjectId,
				Name: project.Name,
			})
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to list projects: %v", err)
	}

	return projects, nil
}