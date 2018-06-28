package addr

type Addr struct {
	Net, Addr string
}

func (a Addr) Network() string {
	return a.Net
}

func (a Addr) String() string {
	return a.Addr
}
