package httpproxy

import (
	"bytes"
	"io"

	"vimagination.zapto.org/errors"
)

const Name = "HTTP"

var (
	Service service

	eoh  = []byte("\r\n\r\n")
	eol  = eoh[:2]
	host = []byte("\r\nHost: ")
)

type service struct{}

func (service) GetServerName(c io.Reader, buf []byte) (int, []byte, error) {
	end := -1
	n := 0
	for end < 0 {
		if n == cap(buf) {
			return n, nil, ErrInvalidHeaders
		}
		m, err := c.Read(buf[n:cap(buf)])
		n += m
		if err != nil {
			if terr, ok := err.(interface {
				Temporary() bool
			}); !ok || !terr.Temporary() {
				return n, nil, errors.WithContext("error reading headers: ", err)
			}
		}
		buf = buf[:n]
		end = bytes.Index(buf, eoh)
	}

	buf = buf[:end-2]

	hi := bytes.Index(buf, host)
	if hi < 0 {
		return n, nil, ErrNoHost
	}
	buf = buf[hi+len(host):]
	lineEnd := bytes.Index(buf, eol)
	if lineEnd < 0 {
		return n, nil, ErrNoHost
	}
	return n, bytes.TrimSpace(buf[:lineEnd-2]), nil
}

func (service) Service() string {
	return Name
}

const (
	ErrInvalidHeaders errors.Error = "invalid headers"
	ErrNoHost         errors.Error = "no host header"
)
