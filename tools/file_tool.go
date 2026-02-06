// Package tools provides utility functions for file operations and other common tasks.
package tools

import "os"

// IsFileExists checks whether a file exists at the given path.
func IsFileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
