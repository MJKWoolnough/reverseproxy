package sslproxy

import (
	"io"
	"net"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

const Name = "SSL"

var Service service

type service struct{}

func (service) GetServerName(c net.Conn, buf *buffer.Buffer) (string, error) {
	_, err := io.CopyN(buf, c, 5)
	if err != nil {
		return "", err
	}
	r := byteio.StickyBigEndianReader{
		Reader: buf,
	}
	if r.ReadUint8() != 22 {
		return "", ErrNoHandshake
	}

	buf.Skip(1) // skip major version
	buf.Skip(1) // skip minor version

	length := r.ReadUint16()

	if cap(buf.LimitedBuffer) < 5+int(length) {
		return "", io.ErrShortBuffer
	}

	_, err = io.CopyN(buf, c, int64(length))
	if err != nil {
		return "", err
	}

	if r.ReadUint8() != 1 {
		return "", ErrNoClientHello
	}

	l := r.ReadUint24()
	if l != uint32(length)-4 {
		return "", errors.WithContext("error reading body: ", ErrInvalidLength)
	}

	buf.Skip(1) // skip major version
	buf.Skip(1) // skip minor version

	buf.Skip(4)  // skip gmt_unix_time
	buf.Skip(28) // skip random_bytes

	sessionLength := r.ReadUint8()
	if sessionLength > 32 || len(buf.LimitedBuffer) < int(sessionLength) {
		// invalid length
		return "", errors.WithContext("error reading sesion id: ", ErrInvalidLength)
	}
	buf.Skip(int(sessionLength)) // skip session id

	cipherSuiteLength := r.ReadUint16()
	if cipherSuiteLength == 0 || len(buf.LimitedBuffer) < int(cipherSuiteLength) {
		// invalid length
		return "", errors.WithContext("error reading cipher suites: ", ErrInvalidLength)
	}
	buf.Skip(int(cipherSuiteLength)) // skip cipher suites

	compressionMethodLength := r.ReadUint8()
	if compressionMethodLength < 1 || len(buf.LimitedBuffer) < int(compressionMethodLength) {
		return "", errors.WithContext("error reading compressions: ", ErrInvalidLength)
	}
	buf.Skip(int(compressionMethodLength)) // skip compression methods

	extsLength := r.ReadUint16()
	if len(buf.LimitedBuffer) < int(extsLength) {
		return "", errors.WithContext("error reading extensions: ", ErrInvalidLength)
	}

	for len(buf.LimitedBuffer) > 0 {
		extType := r.ReadUint16()
		extLength := r.ReadUint16()
		if len(buf.LimitedBuffer) < int(extLength) {
			return "", errors.WithContext("error reading extension: ", ErrInvalidLength)
		}
		if extType == 0 { // server_name
			l := r.ReadUint16()
			if l != extLength-2 {
				return "", errors.WithContext("error reading server name extension: ", ErrInvalidLength)
			}

			buf.Skip(1) // skip name_type

			nameLength := r.ReadUint16()
			if len(buf.LimitedBuffer) < int(nameLength) {
				return "", errors.WithContext("error reading server name: ", ErrInvalidLength)
			}
			return string(buf.LimitedBuffer[:nameLength]), nil
		} else {
			buf.Skip(int(extLength))
		}
	}
	return "", ErrNoName
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
