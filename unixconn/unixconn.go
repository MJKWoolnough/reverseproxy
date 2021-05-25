package unixconn

import (
	"net"
	"os"
	"sync"
)

var (
	unMu sync.Mutex
	uc   *net.UnixConn
)

func init() {
	c, err := net.FileConn(os.NewFile(4, ""))
	if err == nil {
		u, ok := c.(*net.UnixConn)
		if ok {
			uc = c
		}
	}
}
