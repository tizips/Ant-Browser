package browser

import "strings"

func ResolveProfileUsername(username string, profileName string) string {
	if value := strings.TrimSpace(username); value != "" {
		return value
	}
	return strings.TrimSpace(profileName)
}
