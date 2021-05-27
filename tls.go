package reverseproxy

import (
	"errors"
	"fmt"
	"io"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/memio"
)

const maxTLSRead = 5 + 65536

func readTLSServerName(c io.Reader) (string, []byte, error) {
	buf := make([]byte, maxTLSRead)
	mbuf := memio.Buffer(buf[:5])
	n, err := io.ReadFull(c, mbuf)
	if err != nil {
		return "", nil, err
	}
	r := byteio.StickyBigEndianReader{
		Reader: &mbuf,
	}
	if r.ReadUint8() != 22 {
		return "", nil, ErrNoHandshake
	}

	mbuf = mbuf[1:] // skip major version
	mbuf = mbuf[1:] // skip minor version

	length := r.ReadUint16()

	if cap(mbuf) < int(length) {
		return "", nil, io.ErrShortBuffer
	}

	mbuf = mbuf[:length]
	m, err := io.ReadFull(c, mbuf)
	if err != nil {
		return "", nil, err
	}
	n += m

	if r.ReadUint8() != 1 {
		return "", nil, ErrNoClientHello
	}

	l := r.ReadUint24()
	if l != uint32(length)-4 {
		return "", nil, fmt.Errorf("error reading body: %w", ErrInvalidLength)
	}

	mbuf = mbuf[1:] // skip major version
	mbuf = mbuf[1:] // skip minor version

	mbuf = mbuf[4:]  // skip gmt_unix_time
	mbuf = mbuf[28:] // skip random_bytes

	sessionLength := r.ReadUint8()
	if sessionLength > 32 || len(mbuf) < int(sessionLength) {
		// invalid length
		return "", nil, fmt.Errorf("error reading sesion id: %w", ErrInvalidLength)
	}
	mbuf = mbuf[sessionLength:] // skip session id

	cipherSuiteLength := r.ReadUint16()
	if cipherSuiteLength == 0 || len(mbuf) < int(cipherSuiteLength) {
		// invalid length
		return "", nil, fmt.Errorf("error reading cipher suites: %w", ErrInvalidLength)
	}
	mbuf = mbuf[cipherSuiteLength:] // skip cipher suites

	compressionMethodLength := r.ReadUint8()
	if compressionMethodLength < 1 || len(mbuf) < int(compressionMethodLength) {
		return "", nil, fmt.Errorf("error reading compressions: %e", ErrInvalidLength)
	}
	mbuf = mbuf[compressionMethodLength:] // skip compression methods

	extsLength := r.ReadUint16()
	if len(mbuf) < int(extsLength) {
		return "", nil, fmt.Errorf("error reading extensions: %w", ErrInvalidLength)
	}
	mbuf = mbuf[:extsLength]

	for len(mbuf) > 0 {
		extType := r.ReadUint16()
		extLength := r.ReadUint16()
		if len(mbuf) < int(extLength) {
			return "", nil, fmt.Errorf("error reading extension: %w", ErrInvalidLength)
		}
		if extType == 0 { // server_name
			l := r.ReadUint16()
			if l != extLength-2 {
				return "", nil, fmt.Errorf("error reading server name extension: %w", ErrInvalidLength)
			}

			mbuf = mbuf[1:] // skip name_type

			nameLength := r.ReadUint16()
			if len(mbuf) < int(nameLength) {
				return "", nil, fmt.Errorf("error reading server name: %w", ErrInvalidLength)
			}
			return string(mbuf[:nameLength]), buf[:n], nil
		} else {
			mbuf = mbuf[extLength:]
		}
	}
	return "", nil, ErrNoName
}

var (
	ErrNoHandshake   = errors.New("not a handshake")
	ErrNoClientHello = errors.New("not a client hello")
	ErrInvalidLength = errors.New("invalid length")
	ErrNoName        = errors.New("no server name")
)
