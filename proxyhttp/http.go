package proxyhttp

import (
	"bytes"
	"io"
	"net"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy"
)

const Name = "HTTP"

var (
	eoh        = []byte("\r\n\r\n")
	eol        = eoh[:2]
	host       = []byte("\r\nHost: ")
	badRequest = []byte("HTTP/1.0 400\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
)

func New(l net.Listener) *reverseproxy.Proxy {
	return reverseproxy.NewProxy(l, service{})
}

type service struct{}

func (service) GetServerName(c io.Reader, buf []byte) (uint, []byte, error) {
	end := -1
	n := 0
	for end < 0 {
		if n == cap(buf) {
			return uint(n), nil, ErrInvalidHeaders
		}
		m, err := c.Read(buf[n:cap(buf)])
		n += m
		if err != nil {
			if terr, ok := err.(net.Error); !ok || !terr.Temporary() {
				return uint(n), nil, errors.WithContext("error reading headers: ", err)
			}
		}
		buf = buf[:n]
		end = bytes.Index(buf, eoh)
	}

	buf = buf[:end+2]

	hi := bytes.Index(buf, host)
	if hi < 0 {
		return uint(n), nil, ErrNoHost
	}
	buf = buf[hi+len(host):]
	lineEnd := bytes.Index(buf, eol)
	if lineEnd < 0 {
		if w, ok := c.(io.Writer); ok {
			w.Write(badRequest)
		}
		return uint(n), nil, ErrNoHost
	}
	return uint(n), bytes.TrimSpace(buf[:lineEnd]), nil
}

func (service) Name() string {
	return Name
}

const (
	ErrInvalidHeaders errors.Error = "invalid headers"
	ErrNoHost         errors.Error = "no host header"
)
