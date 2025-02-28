package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gcp-instance-explorer/internal/api"

	"google.golang.org/api/compute/v1"
)

// SelectInstance prompts the user to select an instance from the list
func SelectInstance(instances []api.Instance) (*api.Instance, error) {
	fmt.Println("\nSelect an instance:")
	for i, instance := range instances {
		fmt.Printf("[%d] %s (%s, %s)\n", i+1, instance.Name, instance.Zone, instance.Status)
	}

	fmt.Print("\nEnter instance number (or 0 to cancel): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %v", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %s", input)
	}

	if choice == 0 {
		return nil, nil
	}

	if choice < 1 || choice > len(instances) {
		return nil, fmt.Errorf("invalid choice: %d", choice)
	}

	return &instances[choice-1], nil
}

// ManageInstances displays management options and handles user choices
// Returns true if a refresh is needed, false otherwise
func ManageInstances(ctx context.Context, instances []api.Instance, computeService *compute.Service, projectID string) bool {
	for {
		fmt.Println("\nManagement Options:")
		fmt.Println("[1] Turn ON an instance")
		fmt.Println("[2] Turn OFF an instance")
		fmt.Println("[3] BYOS to PAYG Mass Mover")
		fmt.Println("[4] Refresh instance list")
		fmt.Println("[5] Export list to file")
		fmt.Println("[0] Exit")

		fmt.Print("\nEnter choice: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a number")
			continue
		}

		switch choice {
		case 0:
			return false // Exit the program
		case 1:
			handleStartInstance(ctx, instances, computeService)
			return true // Refresh the instance list and return to main menu
		case 2:
			handleStopInstance(ctx, instances, computeService)
			return true // Refresh the instance list and return to main menu
		case 3:
			handleBYOStoPAYG(ctx, instances, computeService, projectID)
			return true // Refresh the instance list after conversion
		case 4:
			fmt.Println("Refreshing instance list...")
			return true // Refresh the instance list and return to main menu
		case 5:
			handleExportInstances(ctx, instances, projectID)
			continue // Return to management menu without refreshing
		default:
			fmt.Println("Invalid choice")
			continue
		}
	}
}

// handleStartInstance handles the process of starting an instance
func handleStartInstance(ctx context.Context, instances []api.Instance, computeService *compute.Service) {
	instance, err := SelectInstance(instances)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if instance == nil {
		return
	}

	fmt.Printf("\nStarting instance: %s\n", instance.Name)
	err = api.StartInstance(ctx, *instance, computeService)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Instance start initiated successfully")
	}
}

// handleStopInstance handles the process of stopping an instance
func handleStopInstance(ctx context.Context, instances []api.Instance, computeService *compute.Service) {
	instance, err := SelectInstance(instances)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if instance == nil {
		return
	}

	fmt.Printf("\nStopping instance: %s\n", instance.Name)
	err = api.StopInstance(ctx, *instance, computeService)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Instance stop initiated successfully")
	}
}

// handleReplaceLicense handles the process of replacing a license
func handleReplaceLicense(ctx context.Context, instances []api.Instance, computeService *compute.Service) {
	instance, err := SelectInstance(instances)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if instance == nil {
		return
	}

	fmt.Print("\nEnter new license URL: ")
	reader := bufio.NewReader(os.Stdin)
	licenseURL, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return
	}

	licenseURL = strings.TrimSpace(licenseURL)
	if licenseURL == "" {
		fmt.Println("License URL cannot be empty")
		return
	}

	fmt.Printf("\nReplacing license for instance: %s\n", instance.Name)
	err = api.ReplaceLicense(ctx, *instance, licenseURL, computeService)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("License replacement initiated successfully")
	}
}

// handleExportInstances handles exporting instances to a YAML file
func handleExportInstances(ctx context.Context, instances []api.Instance, projectID string) {
	if len(instances) == 0 {
		fmt.Println("No instances to export.")
		return
	}

	err := api.ExportInstancesToYAML(instances, projectID)
	if err != nil {
		fmt.Printf("Error exporting instances: %v\n", err)
		return
	}

	fmt.Printf("Instances exported to %s-instances.yml\n", projectID)
}

// handleBYOStoPAYG handles the process of converting BYOS to PAYG
func handleBYOStoPAYG(ctx context.Context, instances []api.Instance, computeService *compute.Service, projectID string) {
	fmt.Println("\nBYOS to PAYG Mass Mover")
	fmt.Println("-----------------------")

	// Check for matching instances from file
	matchedInstances, err := api.CheckInstancesFromFile(projectID, instances)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display the instances that will be converted
	fmt.Printf("\nFound %d instances to convert:\n", len(matchedInstances))
	for _, instance := range matchedInstances {
		fmt.Printf("%s  %s  %s  %s\n",
			instance.Zone,
			instance.Name,
			strings.Join(instance.LicenseCodes, ", "),
			instance.Status)
	}

	// Confirm with user
	fmt.Print("\nAre these the instances you want to convert to PAYG? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return
	}

	input = strings.TrimSpace(input)
	if strings.ToLower(input) != "y" {
		fmt.Println("Conversion cancelled.")
		return
	}

	// Perform conversion
	fmt.Println("\nConverting instances to PAYG licensing...")
	conversions, err := api.ConvertToPAYG(ctx, matchedInstances, computeService)
	if err != nil {
		fmt.Printf("Error during conversion: %v\n", err)
		return
	}

	// Verify conversion
	fmt.Println("\nVerifying license changes...")
	verifiedConversions := api.VerifyConversion(ctx, conversions, computeService)

	// Display results
	fmt.Println("\nConversion Results:")
	fmt.Println("-----------------")

	successful := 0
	for _, conversion := range verifiedConversions {
		status := "✓ Success"
		if !conversion.Success {
			status = "✗ Failed"
		} else {
			successful++
		}

		fmt.Printf("%s  %s  %s\n", status, conversion.Instance.Name, conversion.Instance.Zone)
		fmt.Printf("  Before: %s\n", conversion.OriginalOS)
		fmt.Printf("  After:  %s\n", conversion.NewOS)
		fmt.Println()
	}

	fmt.Printf("\nConverted %d/%d instances successfully.\n", successful, len(verifiedConversions))

	// Press enter to continue
	fmt.Print("\nPress Enter to continue...")
	reader.ReadString('\n')
}
