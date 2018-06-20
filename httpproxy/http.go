package httpproxy

import (
	"bytes"
	"net"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

const Name = "HTTP"

var (
	Service service

	eoh  = []byte("\r\n\r\n")
	eol  = eoh[:2]
	host = []byte("\r\nHost: ")
)

type service struct{}

func (service) GetServerName(c net.Conn, buf *buffer.Buffer) (string, error) {
	end := -1
	for end < 0 {
		if len(buf.LimitedBuffer) == cap(buf.LimitedBuffer) {
			return "", ErrInvalidHeaders
		}
		n, err := c.Read(buf.LimitedBuffer[len(buf.LimitedBuffer):cap(buf.LimitedBuffer)])
		buf.LimitedBuffer = buf.LimitedBuffer[:len(buf.LimitedBuffer)+n]
		if err != nil {
			if terr, ok := err.(interface {
				Temporary() bool
			}); !ok || !terr.Temporary() {
				return "", errors.WithContext("error reading headers: ", err)
			}
		}
		end = bytes.Index(buf.LimitedBuffer, eoh)
	}

	data := buf.LimitedBuffer[:end-2]

	hi := bytes.Index(data, host)
	if hi < 0 {
		return "", ErrNoHost
	}
	data = data[hi+len(hosts)]
	lineEnd := bytes.Index(data, eol)
	if lineEnd < 0 {
		return "", ErrNoHost
	}
	return string(bytes.TrimSpace(data[:lineEnd-2])), nil
}

func (service) Service() string {
	return Name
}

const (
	ErrInvalidHeaders errors.Error = "invalid headers"
	ErrNoHost         errors.Error = "no host header"
)
