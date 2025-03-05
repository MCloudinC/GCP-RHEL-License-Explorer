package api

import (
	"context"
	"encoding/json" // Add this import
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time" // Add this import

	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"gopkg.in/yaml.v3"
)

// PAYGConversion represents a BYOS to PAYG conversion operation
type PAYGConversion struct {
	Instance      Instance
	OriginalOS    string
	ConversionURL string
	Success       bool
	NewOS         string
}

// CheckInstancesFromFile checks if instances from a YAML file exist in the current project
func CheckInstancesFromFile(projectID string, instances []Instance) ([]Instance, error) {
	// Look for file with pattern: {projectID}-instances.yml
	filename := fmt.Sprintf("%s-instances.yml", projectID)

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s not found. Please export instance list first", filename)
	}

	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Parse YAML
	var fileInstances []InstanceExport
	if err := yaml.Unmarshal(data, &fileInstances); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	// Create map of current instances for quick lookup
	instanceMap := make(map[string]Instance)
	for _, instance := range instances {
		key := fmt.Sprintf("%s/%s", instance.Zone, instance.Name)
		instanceMap[key] = instance
	}

	// Match instances from file with current instances
	var matchedInstances []Instance
	var missingInstances []string

	for _, fileInstance := range fileInstances {
		key := fmt.Sprintf("%s/%s", fileInstance.Zone, fileInstance.Name)
		if instance, found := instanceMap[key]; found {
			matchedInstances = append(matchedInstances, instance)
		} else {
			missingInstances = append(missingInstances, key)
		}
	}

	// Report any missing instances
	if len(missingInstances) > 0 {
		fmt.Printf("Warning: %d instances from the file were not found in the current project:\n", len(missingInstances))
		for _, missing := range missingInstances {
			fmt.Printf("  - %s\n", missing)
		}
		fmt.Println()
	}

	if len(matchedInstances) == 0 {
		return nil, fmt.Errorf("no matching instances found between file and current project")
	}

	return matchedInstances, nil
}

// ConvertToPAYG converts instances from BYOS to PAYG licensing
func ConvertToPAYG(ctx context.Context, instances []Instance, computeService *compute.Service) ([]PAYGConversion, error) {
	var results []PAYGConversion

	for _, instance := range instances {
		// Create conversion record
		conversion := PAYGConversion{
			Instance:   instance,
			OriginalOS: strings.Join(instance.LicenseCodes, ", "),
		}

		// Log instance status clearly
		fmt.Printf("\n== Instance %s status: %s ==\n", instance.Name, instance.Status)
		if instance.Status != "RUNNING" {
			fmt.Printf("💡 Note: VM is NOT running. License will be applied to disk but VM needs to be started to use the new license.\n")
		}

		// Get the instance object to find disk details
		instanceObj, err := computeService.Instances.Get(instance.Project, instance.Zone, instance.Name).Context(ctx).Do()
		if err != nil {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Error getting instance details for %s: %v\n", instance.Name, err)
			continue
		}

		// Find the boot disk
		if len(instanceObj.Disks) == 0 {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Instance %s has no disks\n", instance.Name)
			continue
		}

		bootDisk := instanceObj.Disks[0]
		diskName := ""

		// Extract disk name from the source URL
		if bootDisk.Source != "" {
			parts := strings.Split(bootDisk.Source, "/")
			if len(parts) > 0 {
				diskName = parts[len(parts)-1]
			}
		}

		if diskName == "" {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Could not determine disk name for instance %s\n", instance.Name)
			continue
		}

		// Determine the appropriate PAYG license URL based on the current OS
		var paygLicense string

		// Mapping logic
		currentOS := strings.ToLower(strings.Join(instance.LicenseCodes, " "))

		switch {
		case strings.Contains(currentOS, "rhel-8"):
			paygLicense = "https://www.googleapis.com/compute/v1/projects/rhel-cloud/global/licenses/rhel-8-server"
		case strings.Contains(currentOS, "rhel-9"):
			paygLicense = "https://www.googleapis.com/compute/v1/projects/rhel-cloud/global/licenses/rhel-9-server"
		case len(instance.LicenseCodes) == 0:
			// No license codes found, check disk for any OS indicators
			fmt.Printf("No license codes found for VM %s. Attempting to determine OS version...\n", instance.Name)

			// Get disk details directly
			disk, err := computeService.Disks.Get(
				instance.Project,
				instance.Zone,
				diskName).Context(ctx).Do()

			if err != nil {
				fmt.Printf("Could not get disk details: %v. Defaulting to RHEL 9.\n", err)
				paygLicense = "https://www.googleapis.com/compute/v1/projects/rhel-cloud/global/licenses/rhel-9-server"
			} else if disk.SourceImage != "" && strings.Contains(strings.ToLower(disk.SourceImage), "rhel-8") {
				fmt.Printf("Detected RHEL 8 from disk source image: %s\n", disk.SourceImage)
				paygLicense = "https://www.googleapis.com/compute/v1/projects/rhel-cloud/global/licenses/rhel-8-server"
			} else {
				fmt.Printf("Could not determine specific OS version. Defaulting to RHEL 9.\n")
				paygLicense = "https://www.googleapis.com/compute/v1/projects/rhel-cloud/global/licenses/rhel-9-server"
			}
		default:
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Could not determine appropriate PAYG license for %s with OS: %s\n",
				instance.Name, strings.Join(instance.LicenseCodes, ", "))
			continue
		}

		// Use paths=licenses as shown in your example
		apiURL := fmt.Sprintf("https://www.googleapis.com/compute/alpha/projects/%s/zones/%s/disks/%s?paths=licenses",
			instance.Project, instance.Zone, diskName)
		conversion.ConversionURL = apiURL

		// Create the request body with licenses array containing full URLs
		requestBody := fmt.Sprintf(`{"name":"%s", "licenses":["%s"]}`, diskName, paygLicense)

		// Make the API call using a properly authenticated HTTP client
		req, err := http.NewRequest("PATCH", apiURL, strings.NewReader(requestBody))
		if err != nil {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Error creating request for %s: %v\n", instance.Name, err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Log what we're about to do
		fmt.Printf("Converting disk for %s to PAYG license: %s\n", instance.Name, paygLicense)

		client, err := google.DefaultClient(ctx, compute.ComputeScope)
		if err != nil {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Error creating HTTP client for %s: %v\n", instance.Name, err)
			continue
		}

		// Print the actual request being sent for debugging
		fmt.Printf("Making request to URL: %s\n", apiURL)

		resp, err := client.Do(req)
		if err != nil {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("Error making API request for %s: %v\n", instance.Name, err)
			continue
		}
		defer resp.Body.Close()

		// Always read and log the response body for debugging
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			conversion.Success = false
			results = append(results, conversion)
			fmt.Printf("❌ API request failed for %s: %d %s - %s\n",
				instance.Name, resp.StatusCode, resp.Status, string(body))
			continue
		} else {
			// Log successful response status
			fmt.Printf("✓ API response: %d %s\n", resp.StatusCode, resp.Status)

			// Parse the operation from the response
			var operation struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Zone   string `json:"zone"`
			}

			if err := json.Unmarshal(body, &operation); err == nil && operation.Name != "" {
				// Make it very clear this is the GCP operation status, not VM status
				fmt.Printf("  GCP Disk Update Operation '%s':\n", operation.Name)
				fmt.Printf("   - Operation Status: %s (this is the UPDATE operation, not the VM)\n", operation.Status)
				fmt.Printf("   - Target: Disk %s\n", diskName)

				// Wait a bit for the operation to make progress
				time.Sleep(5 * time.Second)
			}
		}

		conversion.Success = true
		if instance.Status != "RUNNING" {
			conversion.NewOS = fmt.Sprintf("PAYG license applied to disk (VM status: %s)", instance.Status)
		} else {
			conversion.NewOS = "PAYG: Converting to " + paygLicense
		}
		results = append(results, conversion)
	}

	return results, nil
}

// VerifyConversion checks if instances were properly converted to PAYG
func VerifyConversion(ctx context.Context, conversions []PAYGConversion, computeService *compute.Service) []PAYGConversion {
	// Add a delay to allow changes to propagate
	fmt.Println("\nWaiting for license changes to propagate...")
	time.Sleep(15 * time.Second)

	for i, conversion := range conversions {
		if !conversion.Success {
			continue
		}

		fmt.Printf("\nVerifying license change for %s (VM status: %s)...\n",
			conversion.Instance.Name, conversion.Instance.Status)

		// First get the disk directly instead of via the instance
		instanceObj, err := computeService.Instances.Get(
			conversion.Instance.Project,
			conversion.Instance.Zone,
			conversion.Instance.Name).Context(ctx).Do()

		if err != nil {
			fmt.Printf("Error getting instance for disk info: %v\n", err)
			continue
		}

		if len(instanceObj.Disks) == 0 {
			fmt.Printf("No disks found for instance %s\n", conversion.Instance.Name)
			continue
		}

		// Extract disk name
		diskName := ""
		if instanceObj.Disks[0].Source != "" {
			parts := strings.Split(instanceObj.Disks[0].Source, "/")
			if len(parts) > 0 {
				diskName = parts[len(parts)-1]
			}
		}

		if diskName == "" {
			fmt.Printf("Could not determine disk name for %s\n", conversion.Instance.Name)
			continue
		}

		fmt.Printf("Checking disk '%s' for license changes...\n", diskName)

		// Get disk details directly
		disk, err := computeService.Disks.Get(
			conversion.Instance.Project,
			conversion.Instance.Zone,
			diskName).Context(ctx).Do()

		if err != nil {
			fmt.Printf("Error getting disk details: %v\n", err)
			continue
		}

		// Extract license information from disk
		var licenseCodes []string
		for _, license := range disk.Licenses {
			parts := strings.Split(license, "/")
			if len(parts) >= 6 {
				project := parts[len(parts)-4]
				licenseCode := parts[len(parts)-1]
				licenseCodes = append(licenseCodes, fmt.Sprintf("%s:%s", project, licenseCode))
			}
		}

		if len(licenseCodes) > 0 {
			fmt.Printf("✓ Found %d licenses on disk: %s\n", len(licenseCodes), strings.Join(licenseCodes, ", "))
			conversions[i].NewOS = strings.Join(licenseCodes, ", ")
		} else if conversion.Instance.Status != "RUNNING" {
			fmt.Printf("⚠️ No licenses found. VM is not running - start VM to apply license.\n")
			conversions[i].NewOS = "License changed, but VM needs to be started to verify"
		} else {
			fmt.Printf("⚠️ No licenses found, but VM is running. License change may be pending.\n")
			conversions[i].NewOS = "License change may be pending"
		}
	}

	return conversions
}
