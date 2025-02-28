package api

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// InstanceExport represents the simplified instance data for export
type InstanceExport struct {
	Name        string   `yaml:"name"`
	Zone        string   `yaml:"zone"`
	MachineType string   `yaml:"machineType"`
	Status      string   `yaml:"status"`
	Licenses    []string `yaml:"licenses,omitempty"`
}

// ExportInstancesToYAML exports instances to a YAML file with selected fields only
func ExportInstancesToYAML(instances []Instance, projectID string) error {
	// Create filename based on project ID
	filename := fmt.Sprintf("%s-instances.yml", projectID)

	// Create simplified export list with only the fields we want
	var exportData []InstanceExport
	for _, instance := range instances {
		exportInstance := InstanceExport{
			Name:        instance.Name,
			Zone:        instance.Zone,
			MachineType: instance.MachineType,
			Status:      instance.Status,
			Licenses:    instance.LicenseCodes,
		}
		exportData = append(exportData, exportInstance)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(exportData)
	if err != nil {
		return fmt.Errorf("failed to marshal instances to YAML: %v", err)
	}

	// Write to file
	err = os.WriteFile(filename, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write YAML to file: %v", err)
	}

	return nil
}
