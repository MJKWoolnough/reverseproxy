package reverseproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

var (
	eoh  = []byte("\r\n\r\n")
	eol  = eoh[:2]
	host = []byte("\r\nHost: ")
)

func readHTTPServerName(r io.Reader) (string, []byte, error) {
	n := 0
	h := -1
	l := -1
	buf := make([]byte, http.DefaultMaxHeaderBytes)
	for l >= 0 {
		m, err := r.Read(buf[n:])
		if err != nil {
			if terr, ok := err.(net.Error); !ok || !terr.Temporary() {
				return "", nil, fmt.Errorf("error reading headers: %w", err)
			}
		}
		h = bytes.Index(buf, host)
		if e := bytes.Index(buf, eoh); h > 0 && (e > h || e == -1) {
			l = bytes.Index(buf[h:], eol)
		} else if e >= 0 {
			return "", nil, errNoServerHeader
		}
		n += m
	}
	sh := h + len(host)
	return string(buf[sh : sh+l]), buf, nil
}

var (
	errNoServerHeader = errors.New("no server header")
)