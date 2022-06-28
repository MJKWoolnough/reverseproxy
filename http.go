package reverseproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
)

var (
	eoh  = []byte("\r\n\r\n")
	eol  = eoh[:2]
	host = []byte("\r\nHost: ")
)

func readHTTPServerName(r io.Reader, buf []byte) (string, []byte, error) {
	n := 1
	h := -1
	l := -1
	for l < 0 {
		m, err := r.Read(buf[n:])
		n += m
		h = bytes.Index(buf[:n], host)
		if e := bytes.Index(buf[:n], eoh); h > 0 && (e > h || e == -1) {
			h += len(host)
			l = bytes.Index(buf[h:n], eol)
		} else if e >= 0 {
			return "", buf, errNoServerHeader
		}
		if err != nil {
			if terr, ok := err.(net.Error); !ok || !terr.Temporary() {
				return "", buf, fmt.Errorf("error reading headers: %w", err)
			}
		}
	}
	return string(buf[h : h+l]), buf[:n], nil
}

var errNoServerHeader = errors.New("no server header")
