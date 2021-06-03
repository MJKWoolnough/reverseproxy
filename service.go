package reverseproxy

import "strings"

type matchServiceName interface {
	MatchService(string) bool
}

// HostName represents an exact hostname to match on
type HostName string

func (h HostName) MatchService(serviceName string) bool {
	return string(h) == serviceName
}

// HostNameSuffix represents a partial hostname to match the end on
type HostNameSuffix string

func (h HostNameSuffix) MatchService(serviceName string) bool {
	return strings.HasSuffix(serviceName, string(h))
}

// Hosts represents a list of servicenames to match against
type Hosts []matchServiceName

func (h Hosts) MatchService(serviceName string) bool {
	for _, s := range h {
		if s.MatchService(serviceName) {
			return true
		}
	}
	return false
}
