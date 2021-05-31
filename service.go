package reverseproxy

import "strings"

type matchServiceName interface {
	MatchService(string) bool
}

type HostName string

func (h HostName) MatchService(serviceName string) bool {
	return string(h) == serviceName
}

type HostNameSuffix string

func (h HostNameSuffix) MatchService(serviceName string) bool {
	return strings.HasSuffix(serviceName, string(h))
}

type Hosts []matchServiceName

func (h Hosts) MatchService(serviceName string) bool {
	for _, s := range h {
		if s.MatchService(serviceName) {
			return true
		}
	}
	return false
}
