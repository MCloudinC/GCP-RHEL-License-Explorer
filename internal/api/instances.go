package api

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	"google.golang.org/api/compute/v1"
)

// Instance represents a GCP compute instance with relevant information
type Instance struct {
	Name         string
	Zone         string
	MachineType  string
	Status       string
	IP           string
	LicenseCodes []string // License codes
	DiskType     string   // Disk type
	DiskSizeGB   int64    // Disk size
	Project      string   // Add project ID
}

// ListInstances retrieves all instances in the specified project
func ListInstances(ctx context.Context, projectID string, computeService *compute.Service) ([]Instance, error) {
	// Get instances across all zones using AggregatedList
	req := computeService.Instances.AggregatedList(projectID)
	var instances []Instance

	// Make the API call
	if err := req.Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		// Iterate through the items (zones)
		for zoneKey, instanceList := range page.Items {
			// Skip if no instances in this zone
			if instanceList.Instances == nil || len(instanceList.Instances) == 0 {
				continue
			}

			// Extract zone name from the key
			zoneName := strings.TrimPrefix(zoneKey, "zones/")

			// Process each instance in this zone
			for _, instance := range instanceList.Instances {
				// Extract machine type shortname
				machineType := instance.MachineType
				if parts := strings.Split(machineType, "/"); len(parts) > 0 {
					machineType = parts[len(parts)-1]
				}

				// Get external IP if available
				var ip string
				if len(instance.NetworkInterfaces) > 0 && len(instance.NetworkInterfaces[0].AccessConfigs) > 0 {
					ip = instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
				}

				// Extract license information from disks
				var licenseCodes []string
				var diskType string
				var diskSizeGB int64

				if len(instance.Disks) > 0 {
					// Use the boot disk (first disk) for license info
					bootDisk := instance.Disks[0]

					// For disk type, we only have the interface type (SCSI, NVME, etc.)
					// and disk type (PERSISTENT, SCRATCH)
					diskType = bootDisk.Type
					if bootDisk.Interface != "" {
						diskType = bootDisk.Interface + "-" + diskType
					}

					diskSizeGB = bootDisk.DiskSizeGb

					// Extract licenses from the boot disk
					for _, license := range bootDisk.Licenses {
						// Extract just the license name from the full URL
						licenseName := path.Base(license)

						// Try to extract the  license code
						parts := strings.Split(license, "/")
						if len(parts) >= 6 {
							// Format is usually: https://www.googleapis.com/compute/v1/projects/PROJECT/global/licenses/LICENSE
							project := parts[len(parts)-4]
							licenseCode := parts[len(parts)-1]
							licenseCodes = append(licenseCodes, fmt.Sprintf("%s:%s", project, licenseCode))
						} else {
							// Fallback if the format is different
							licenseCodes = append(licenseCodes, licenseName)
						}
					}
				}

				// Add instance to our list
				instances = append(instances, Instance{
					Name:         instance.Name,
					Zone:         zoneName,
					MachineType:  machineType,
					Status:       instance.Status,
					IP:           ip,
					LicenseCodes: licenseCodes,
					DiskType:     diskType,
					DiskSizeGB:   diskSizeGB,
					Project:      projectID,
				})
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to list instances: %v", err)
	}

	return instances, nil
}

// StartInstance turns on an instance
func StartInstance(ctx context.Context, instance Instance, computeService *compute.Service) error {
	op, err := computeService.Instances.Start(instance.Project, instance.Zone, instance.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to start instance: %v", err)
	}

	fmt.Printf("Operation in progress: %s\n", op.Name)
	return nil
}

// StopInstance turns off an instance
func StopInstance(ctx context.Context, instance Instance, computeService *compute.Service) error {
	op, err := computeService.Instances.Stop(instance.Project, instance.Zone, instance.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to stop instance: %v", err)
	}

	fmt.Printf("Operation in progress: %s\n", op.Name)
	return nil
}

// ReplaceLicense replaces the license URL for an instance
func ReplaceLicense(ctx context.Context, instance Instance, newLicenseURL string, computeService *compute.Service) error {
	// First, need to get the current instance to check its disks
	instanceObj, err := computeService.Instances.Get(instance.Project, instance.Zone, instance.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get instance details: %v", err)
	}

	// Find the boot disk
	if len(instanceObj.Disks) == 0 {
		return fmt.Errorf("instance has no disks")
	}

	fmt.Println("Note: Changing licenses typically requires recreating the instance.")
	fmt.Println("This feature is limited in direct API usage.")
	fmt.Println("Alternative: Set custom metadata to track license information.")

	// Set metadata with license information (this doesn't actually change the license)
	fingerprint := instanceObj.Metadata.Fingerprint
	items := instanceObj.Metadata.Items

	// Add or update license metadata
	licenseFound := false
	for i, item := range items {
		if item.Key == "license" {
			items[i].Value = &newLicenseURL
			licenseFound = true
			break
		}
	}

	if !licenseFound {
		items = append(items, &compute.MetadataItems{
			Key:   "license",
			Value: &newLicenseURL,
		})
	}

	// Create the new metadata
	newMetadata := &compute.Metadata{
		Fingerprint: fingerprint,
		Items:       items,
	}

	// Set the metadata on the instance
	op, err := computeService.Instances.SetMetadata(
		instance.Project,
		instance.Zone,
		instance.Name,
		newMetadata).Context(ctx).Do()

	if err != nil {
		return fmt.Errorf("failed to set license metadata: %v", err)
	}

	fmt.Println("License information added to instance metadata.")
	fmt.Printf("Operation in progress: %s\n", op.Name)
	fmt.Println("Note: This does not change the actual license, only records it in metadata.")

	return nil
}

// DisplayInstances prints instances in a simplified one-line format without IP and disk info
func DisplayInstances(instances []Instance, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	// Create a tabwriter for clean alignment
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(tw, "NAME\tZONE\tMACHINE TYPE\tSTATUS\tLICENSES")

	// Print each instance on one line
	for _, instance := range instances {
		licenses := "none"
		if len(instance.LicenseCodes) > 0 {
			licenses = strings.Join(instance.LicenseCodes, ", ")
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			instance.Name,
			instance.Zone,
			instance.MachineType,
			instance.Status,
			licenses)
	}

	tw.Flush()
}

// FormatInstanceName returns a simple string representation of an instance
func FormatInstanceName(instance Instance) string {
	return fmt.Sprintf("%s (%s/%s)", instance.Name, instance.Zone, instance.Status)
}

// The ConvertToPAYG function is now implemented in payg_converter.go
// Do not define it here to avoid duplicate definition errors
