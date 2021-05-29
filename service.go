package reverseproxy

import "strings"

type matchServiceName interface {
	MatchService(string) bool
}

type HostName string

func (h HostName) MatchServiceName(serviceName string) bool {
	return string(h) == serviceName
}

type HostNameSuffix string

func (h HostNameSuffix) MatchServiceName(serviceName string) bool {
	return strings.HasSuffix(serviceName, string(h))
}

type Hosts []service

func (h Hosts) MatchServiceName(serviceName string) bool {
	for _, s := range h {
		if s.MatchServiceName(serviceName) {
			return true
		}
	}
	return false
}
