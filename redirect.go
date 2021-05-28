package reverseproxy

type Redirect struct {
	proxy   *Proxy
	service service
}

func (p *Proxy) RegisterRedirecter(service service) (*Redirect, error) {
	return Redirect{proxy: p, service: service}, nil
}
