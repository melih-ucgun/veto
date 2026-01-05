package utils

import (
	"regexp"
)

// Regex for valid system names (users, groups, etc.)
// Starts with letter/underscore, contains letters, numbers, underscores, dashes.
// Typical Linux username length limit is 32, but we'll be slightly lenient on length here, focusing on chars.
var NameRegex = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

// IsValidName checks if the given name is a valid system identifier.
func IsValidName(name string) bool {
	return NameRegex.MatchString(name)
}

// IsOneOf checks if the value is one of the allowed options.
func IsOneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// IsValidPort checks if the port is within lawful range
func IsValidPort(port int) bool {
	return port > 0 && port <= 65535
}

// IsValidProtocol checks if protocol is tcp, udp or any
func IsValidProtocol(proto string) bool {
	return IsOneOf(proto, "tcp", "udp", "any")
}
