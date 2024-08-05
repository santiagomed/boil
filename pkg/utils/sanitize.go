package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizeInput removes any potentially harmful characters from the input string
func SanitizeInput(input string) string {
	// Remove any character that isn't alphanumeric, space, or common punctuation
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s\-_.,!?()[\]{}]`)
	sanitized := reg.ReplaceAllString(input, "")

	// Trim leading and trailing whitespace
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}

// SanitizeFilePath sanitizes a file path to prevent directory traversal attacks
func SanitizeFilePath(path string) string {
	// Convert to slash path
	path = filepath.ToSlash(path)

	// Remove any "." or ".." components
	parts := strings.Split(path, "/")
	var sanitizedParts []string
	for _, part := range parts {
		if part != "." && part != ".." {
			sanitizedParts = append(sanitizedParts, part)
		}
	}

	// Join the parts back together
	sanitized := strings.Join(sanitizedParts, "/")

	// Ensure the path doesn't start with a "/"
	sanitized = strings.TrimPrefix(sanitized, "/")

	return sanitized
}

// IsValidProjectName checks if the given project name is valid
func IsValidProjectName(name string) bool {
	// Project name should start with a letter or number,
	// and can contain letters, numbers, hyphens, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9\-_]*$`, name)
	return matched
}

func FormatProjectName(name string) string {
	// Replace spaces and other invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	formatted := reg.ReplaceAllString(name, "-")

	// Remove leading hyphens or underscores
	formatted = strings.TrimLeft(formatted, "-_")

	// If the name is not empty and starts with a number, prepend "project-"
	if len(formatted) > 0 && strings.IndexAny(formatted[0:1], "0123456789") == 0 {
		formatted = "project-" + formatted
	}

	// If the name is empty after formatting, use a default name
	if formatted == "" {
		formatted = "boilerplate-project"
	}

	return formatted
}

// TruncateString truncates a string to the specified length, adding an ellipsis if truncated
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
