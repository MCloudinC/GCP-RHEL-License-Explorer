package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/api/option"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"golang.org/x/oauth2/google"
)

// Authenticate tries multiple authentication methods and returns service clients
func Authenticate() (*cloudresourcemanager.Service, *compute.Service, error) {
	ctx := context.Background()
	
	// Check for GOOGLE_APPLICATION_CREDENTIALS environment variable
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if (credPath != "") {
		fmt.Println("Using credentials from GOOGLE_APPLICATION_CREDENTIALS")
	} else {
		fmt.Println("GOOGLE_APPLICATION_CREDENTIALS not set, trying application default credentials...")
	}
	
	// Try to find default credentials
	creds, err := google.FindDefaultCredentials(ctx, 
		cloudresourcemanager.CloudPlatformScope,
		compute.CloudPlatformScope)
	
	if err != nil {
		// If we couldn't find credentials, suggest solutions
		homeDir, _ := os.UserHomeDir()
		adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
		
		return nil, nil, fmt.Errorf("failed to obtain credentials: %v\n\nPossible solutions:\n"+
			"1. Run 'gcloud auth application-default login'\n"+
			"2. Set GOOGLE_APPLICATION_CREDENTIALS to point to a service account key file\n"+
			"3. Check if %s exists\n", err, adcPath)
	}
	
	fmt.Println("Successfully obtained credentials")
	
	// Create the Cloud Resource Manager service
	crmService, err := cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Cloud Resource Manager service: %v\n\n"+
			"Make sure the Cloud Resource Manager API is enabled in your GCP project", err)
	}
	
	// Create the Compute service
	computeService, err := compute.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Compute service: %v\n\n"+
			"Make sure the Compute Engine API is enabled in your GCP project", err)
	}
	
	return crmService, computeService, nil
}

// HandleError checks for errors and prints them with helpful context
func HandleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// TestAuth is a simple function to test authentication and print projects
// Use this to verify credentials are working
func TestAuth() {
	fmt.Println("Testing GCP authentication...")
	crmService, _, err := Authenticate()
	if err != nil {
		HandleError(err)
	}
	
	resp, err := crmService.Projects.List().Do()
	if err != nil {
		HandleError(fmt.Errorf("failed to list projects: %v\n\n"+
			"Check that your account has permission to list projects", err))
	}
	
	fmt.Printf("Successfully authenticated! Found %d projects\n", len(resp.Projects))
}