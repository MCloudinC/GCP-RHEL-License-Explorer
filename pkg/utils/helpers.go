package utils

import (
	"log"
)

// CheckError is a utility function that logs and exits the application if an error occurs.
func CheckError(err error) {
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// FormatString is a utility function that formats a string for display.
func FormatString(s string) string {
	return "Formatted: " + s
}