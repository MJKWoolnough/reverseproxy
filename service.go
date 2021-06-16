package reverseproxy

import "strings"

// MatchServiceName allows differing ways of matching a service name to a service
type MatchServiceName interface {
	MatchService(string) bool
}

// HostName represents an exact hostname to match on
type HostName string

// MatchService implements the MatchServiceName interface
func (h HostName) MatchService(serviceName string) bool {
	return string(h) == serviceName
}

// HostNameSuffix represents a partial hostname to match the end on
type HostNameSuffix string

// MatchService implements the MatchServiceName interface
func (h HostNameSuffix) MatchService(serviceName string) bool {
	return strings.HasSuffix(serviceName, string(h))
}

// Hosts represents a list of servicenames to match against
type Hosts []MatchServiceName

// MatchService implements the MatchServiceName interface
func (h Hosts) MatchService(serviceName string) bool {
	for _, s := range h {
		if s.MatchService(serviceName) {
			return true
		}
	}
	return false
}
