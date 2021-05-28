package reverseproxy

import "strings"

type service interface {
	IsService(string) bool
}

type HostName string

func (h HostName) IsService(serviceName string) bool {
	return string(h) == serviceName
}

type HostNameSuffix string

func (h HostNameSuffix) IsService(serviceName string) bool {
	return strings.HasSuffix(serviceName, string(h))
}

type Hosts []service

func (h Hosts) IsService(serviceName string) bool {
	for _, s := range h {
		if s.IsService(serviceName) {
			return true
		}
	}
	return false
}
