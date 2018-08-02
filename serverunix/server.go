package serverunix

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

type Host struct {
	mu       sync.Mutex
	names    []string
	services map[*reverseproxy.Proxy]*service
}

func New(serverName string, aliases ...string) *Host {
	nms := make([]string, len(aliases), len(aliases)+1)
	copy(nms, aliases)
	nms = append(nms, serverName)
	return &Host{
		names:    nms,
		services: make(map[*reverseproxy.Proxy]*service),
	}
	return nil
}

func (h *Host) AddAlias(serverName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	done := make([]*reverseproxy.Proxy, 0, len(h.services))
	for p, s := range h.services {
		if err := p.Add(serverName, s); err != nil {
			for _, p := range done {
				p.Remove(serverName) // remove previous registrations
			}
			return err
		}
		done = append(done, p)
	}
	h.names = append(h.names, serverName)
	return nil
}

func (h *Host) RemoveAlias(serverName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var (
		name  bool
		n     int
		alias string
	)
	for n, alias = range h.names {
		if alias == serverName {
			name = true
			break
		}
	}
	if !name {
		return ErrUnknownAlias
	}
	for p := range h.services {
		p.Remove(serverName)
	}
	h.names[n] = h.names[len(h.names)-1]
	h.names = h.names[:len(h.names)-1]
	return nil
}

func (h *Host) RegisterCmd(cmd *exec.Cmd) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	socks := make(map[*reverseproxy.Proxy][2]int, len(h.services))
	for p := range h.services {
		fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		if err != nil {
			return errors.WithContext("error creating socket pair: ", err)
		}
		socks[p] = fds
	}
	for p, s := range h.services {
		s.mu.Lock()
		if s.conn != nil {
			s.conn.Close()
		}
		fds := socks[p]
		c, _ := net.FileConn(os.NewFile(uintptr(fds[0]), ""))
		s.conn = c.(*net.UnixConn)
		cmd.Env = append(cmd.Env, fmt.Sprintf("rproxy_%s=%d", p.Name(), len(cmd.ExtraFiles)+3))
		cmd.ExtraFiles = append(cmd.ExtraFiles, os.NewFile(uintptr(fds[1]), ""))
		s.mu.Unlock()
	}
	return nil
}

func (h *Host) RegisterProxy(p *reverseproxy.Proxy) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := &service{}
	for n, alias := range h.names {
		if err := p.Add(alias, s); err != nil {
			for _, alias := range h.names[:n] {
				p.Remove(alias) // remove previous aliases
			}
			return err
		}
	}
	h.services[p] = s
	return nil
}

func (h *Host) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var e error
	for _, s := range h.services {
		s.mu.Lock()
		if err := s.conn.Close(); err != nil && e != nil {
			e = err
		}
		s.conn = nil
		s.mu.Unlock()
	}
	return e
}

type service struct {
	mu     sync.Mutex
	conn   *net.UnixConn
	length buffer.BufferLength
}

func (s *service) Handle(c net.Conn, buf *buffer.Buffer, length uint) {
	cf, ok := c.(interface{ File() (*os.File, error) })
	if ok {
		f, err := cf.File()
		if err == nil {
			s.mu.Lock()
			if s.conn != nil {
				s.length.WriteUint(length)
				s.conn.WriteMsgUnix(s.length[:], syscall.UnixRights(int(f.Fd())), nil)
				s.conn.Write(buf[:length])
			}
			s.mu.Unlock()
		}
	}
	buffer.Put(buf)
}

func (s *service) Stop() {
	s.conn.Close()
}

const (
	ErrBadSocket    errors.Error = "bad socket"
	ErrUnknownAlias errors.Error = "unknown alias"
)
