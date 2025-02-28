package ui

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "gcp-instance-explorer/internal/api"
)

// SelectProject asks the user to input a project ID directly without listing all projects
func SelectProject(projects []api.Project) (api.Project, error) {
    fmt.Print("\nEnter Project ID: ")
    reader := bufio.NewReader(os.Stdin)
    input, err := reader.ReadString('\n')
    if err != nil {
        return api.Project{}, fmt.Errorf("failed to read input: %v", err)
    }

    input = strings.TrimSpace(input)
    
    // Create a project with the given ID (without validation)
    return api.Project{
        ID:   input,
        Name: input, // We don't know the name, so use ID as name
    }, nil
}