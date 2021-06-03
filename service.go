package reverseproxy

import "strings"

// MatchServiceName allows differing ways of matching a service name to a service
type MatchServiceName interface {
	matchService(string) bool
}

// HostName represents an exact hostname to match on
type HostName string

func (h HostName) matchService(serviceName string) bool {
	return string(h) == serviceName
}

// HostNameSuffix represents a partial hostname to match the end on
type HostNameSuffix string

func (h HostNameSuffix) matchService(serviceName string) bool {
	return strings.HasSuffix(serviceName, string(h))
}

// Hosts represents a list of servicenames to match against
type Hosts []MatchServiceName

func (h Hosts) matchService(serviceName string) bool {
	for _, s := range h {
		if s.matchService(serviceName) {
			return true
		}
	}
	return false
}
