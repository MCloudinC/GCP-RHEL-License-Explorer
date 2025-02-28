package export

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Instance struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
}

type InstanceList struct {
	Instances []Instance `yaml:"instances"`
}

func ExportInstancesToYAML(instances []Instance, filePath string) error {
	instanceList := InstanceList{Instances: instances}
	data, err := yaml.Marshal(&instanceList)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}