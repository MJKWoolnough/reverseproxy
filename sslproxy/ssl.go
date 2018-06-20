package sslproxy

import (
	"io"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/memio"
)

const Name = "SSL"

var Service service

type service struct{}

func (service) GetServerName(c io.Reader, buf []byte) (int, []byte, error) {
	mbuf := memio.Buffer(buf[:5])
	n, err := io.ReadFull(c, mbuf)
	if err != nil {
		return n, nil, err
	}
	r := byteio.StickyBigEndianReader{
		Reader: &mbuf,
	}
	if r.ReadUint8() != 22 {
		return n, nil, ErrNoHandshake
	}

	mbuf = mbuf[1:] // skip major version
	mbuf = mbuf[1:] // skip minor version

	length := r.ReadUint16()

	if cap(mbuf) < int(length) {
		return n, nil, io.ErrShortBuffer
	}

	mbuf = mbuf[:length]
	m, err := io.ReadFull(c, mbuf)
	if err != nil {
		return n, nil, err
	}
	n += m

	if r.ReadUint8() != 1 {
		return n, nil, ErrNoClientHello
	}

	l := r.ReadUint24()
	if l != uint32(length)-4 {
		return n, nil, errors.WithContext("error reading body: ", ErrInvalidLength)
	}

	mbuf = mbuf[1:] // skip major version
	mbuf = mbuf[1:] // skip minor version

	mbuf = mbuf[4:]  // skip gmt_unix_time
	mbuf = mbuf[28:] // skip random_bytes

	sessionLength := r.ReadUint8()
	if sessionLength > 32 || len(mbuf) < int(sessionLength) {
		// invalid length
		return n, nil, errors.WithContext("error reading sesion id: ", ErrInvalidLength)
	}
	mbuf = mbuf[sessionLength:] // skip session id

	cipherSuiteLength := r.ReadUint16()
	if cipherSuiteLength == 0 || len(mbuf) < int(cipherSuiteLength) {
		// invalid length
		return n, nil, errors.WithContext("error reading cipher suites: ", ErrInvalidLength)
	}
	mbuf = mbuf[cipherSuiteLength:] // skip cipher suites

	compressionMethodLength := r.ReadUint8()
	if compressionMethodLength < 1 || len(mbuf) < int(compressionMethodLength) {
		return n, nil, errors.WithContext("error reading compressions: ", ErrInvalidLength)
	}
	mbuf = mbuf[compressionMethodLength:] // skip compression methods

	extsLength := r.ReadUint16()
	if len(mbuf) < int(extsLength) {
		return n, nil, errors.WithContext("error reading extensions: ", ErrInvalidLength)
	}
	mbuf = mbuf[:extsLength]

	for len(mbuf) > 0 {
		extType := r.ReadUint16()
		extLength := r.ReadUint16()
		if len(mbuf) < int(extLength) {
			return n, nil, errors.WithContext("error reading extension: ", ErrInvalidLength)
		}
		if extType == 0 { // server_name
			l := r.ReadUint16()
			if l != extLength-2 {
				return n, nil, errors.WithContext("error reading server name extension: ", ErrInvalidLength)
			}

			mbuf = mbuf[1:] // skip name_type

			nameLength := r.ReadUint16()
			if len(mbuf) < int(nameLength) {
				return n, nil, errors.WithContext("error reading server name: ", ErrInvalidLength)
			}
			return n, mbuf[:nameLength], nil
		} else {
			mbuf = mbuf[extLength:]
		}
	}
	return n, nil, ErrNoName
}

func (service) Service() string {
	return Name
}

const (
	ErrNoHandshake   errors.Error = "not a handshake"
	ErrNoClientHello errors.Error = "not a client hello"
	ErrInvalidLength errors.Error = "invalid length"
	ErrNoName        errors.Error = "no server name"
)
