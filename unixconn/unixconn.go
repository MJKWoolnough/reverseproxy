package unixconn

import (
	"crypto/tls"
	"errors"
	"net"
	"os"
	"sync"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/memio"
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

func ListenHTTP(network, address string) (net.Listener, error) {
	return requestListener(network, address, false)
}

func ListenTLS(network, address string, config *tls.Config) (net.Listener, error) {
	if config == nil || len(config.Certificates) == 0 && config.GetCertificate == nil && config.GetConfigForClient == nil {
		return nil, errors.New("need valid tls.Config")
	}
	l, err := requestListener(network, address, true)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(l, config), nil
}

func requestListener(network, address string, isTLS bool) (net.Listener, error) {
	if uc == nil {
		return net.Listen(network, address)
	}
	buf := make(memio.Buffer, 0, len(network)+len(address)+5)
	w := byteio.StickyLittleEndianWriter{Writer: &buf}
	w.WriteString16(network)
	w.WriteString16(address)
	w.WriteBool(isTLS)
	var (
		errNum [4]byte
		oob    [8]byte
	)
	ucMu.Lock()
	uc.Write(buf)
	_, oobn, flags, _, err := uc.ReadMsgUnix(errNum[:], oob[:])
	ucMu.Unlock()

	return nil, nil
}
