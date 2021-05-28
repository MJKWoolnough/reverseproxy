package reverseproxy

type service interface {
	IsService(string) bool
}

type HostName string

func (h HostName) IsService(serviceName string) bool {
	return string(h) == serviceName
}
