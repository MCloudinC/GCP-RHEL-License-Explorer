package main

import (
	"context"
	"fmt"
	"log"
	"os" // Add this import

	"gcp-instance-explorer/internal/api"
	"gcp-instance-explorer/internal/auth"
	"gcp-instance-explorer/internal/ui"
)

func main() {
	ctx := context.Background()

	// Authenticate the user and retrieve API services
	fmt.Println("Authenticating with GCP...")
	_, computeService, err := auth.Authenticate()
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println("Authentication successful!")

	// Skip listing all projects - go directly to project selection
	var projects []api.Project

	// Let the user enter a project ID directly
	selectedProject, err := ui.SelectProject(projects)
	if err != nil {
		log.Fatalf("Project selection failed: %v", err)
	}

	fmt.Printf("Using project: %s\n", selectedProject.ID)

	// Main program loop
	for {
		// List all instances in the selected project
		fmt.Printf("Fetching instances for project %s...\n", selectedProject.ID)
		instances, err := api.ListInstances(ctx, selectedProject.ID, computeService)
		if err != nil {
			log.Fatalf("Failed to list instances: %v", err)
		}

		// Output the instances using the simplified display format
		if len(instances) == 0 {
			fmt.Println("No instances found in this project.")
		} else {
			fmt.Printf("Found %d instances:\n\n", len(instances))
			// Use the new DisplayInstances function instead of the verbose output
			api.DisplayInstances(instances, os.Stdout)
		}

		fmt.Println() // Add a blank line for better spacing

		// Present the management menu
		refreshNeeded := ui.ManageInstances(ctx, instances, computeService, selectedProject.ID)

		// Exit if user chose to exit (option 0)
		if !refreshNeeded {
			fmt.Println("Goodbye!")
			break
		}
		// Otherwise loop continues with a refreshed instance list
	}
}
