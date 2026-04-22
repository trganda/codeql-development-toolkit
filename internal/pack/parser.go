package pack

import "strings"

// GetPackScope returns the scope part of the pack name (before "/").
func GetPackScope(packName string) string {
	if idx := strings.Index(packName, "/"); idx >= 0 {
		return packName[:idx]
	}
	return ""
}

// GetPackName returns the name part after the scope.
func GetPackName(packName string) string {
	if idx := strings.Index(packName, "/"); idx >= 0 {
		return packName[idx+1:]
	}
	return packName
}
