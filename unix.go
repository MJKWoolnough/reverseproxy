package reverseproxy

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
)

type unixService struct {
	MatchServiceName
	conn         *net.UnixConn
	transferring uint64
}

func (u *unixService) Transfer(buf []byte, conn *net.TCPConn) error {
	f, err := conn.File()
	conn.Close()
	if err == nil {
		atomic.AddUint64(&u.transferring, 1)
		_, _, err = u.conn.WriteMsgUnix(buf, syscall.UnixRights(int(f.Fd())), nil)
		atomic.AddUint64(&u.transferring, ^uint64(0))
		errr := f.Close()
		if err == nil {
			err = errr
		}
	}
	return err
}

func (u *unixService) Active() bool {
	return atomic.LoadUint64(&u.transferring) > 0
}

// UnixCmd holds the information required to control (close) a server and its
// resources
type UnixCmd struct {
	cmd  *exec.Cmd
	conn *net.UnixConn

	mu     sync.Mutex
	open   map[uint16]*Port
	closed bool
	exited bool
}

// Close closes all ports for the server and sends a signal to the server to
// close
func (u *UnixCmd) Close() error {
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		return ErrClosed
	}
	for port, p := range u.open {
		delete(u.open, port)
		p.Close()
	}
	err := u.conn.Close()
	errr := u.cmd.Process.Signal(os.Interrupt)
	u.closed = true
	u.mu.Unlock()
	if err != nil {
		return err
	}
	return errr
}

// Status retrieves the Status of the UnixCmd
func (u *UnixCmd) Status() Status {
	u.mu.Lock()
	closed := u.closed
	ports := make([]uint16, 0, len(u.open))
	for p := range u.open {
		ports = append(ports, p)
	}
	u.mu.Unlock()
	return Status{
		Ports:   ports,
		Closing: closed,
		Active:  !u.exited,
	}
}

// RegisterCmd runs the given command and waits for incoming listeners from it
func RegisterCmd(msn MatchServiceName, cmd *exec.Cmd) (*UnixCmd, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	nf := os.NewFile(uintptr(fds[0]), "")
	fconn, err := net.FileConn(nf)
	if err := nf.Close(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fds[1]), "")
	cmd.ExtraFiles = append([]*os.File{}, f)
	err = cmd.Start()
	f.Close()
	if err != nil {
		return nil, err
	}
	u := &UnixCmd{
		cmd:  cmd,
		conn: fconn.(*net.UnixConn),
		open: make(map[uint16]*Port),
	}
	go u.runCmdLoop(msn)
	return u, nil
}

func (u *UnixCmd) runCmdLoop(msn MatchServiceName) {
	var (
		buf [2]byte
		srv = &unixService{
			MatchServiceName: msn,
			conn:             u.conn,
		}
	)
	for {
		n, _, _, _, err := u.conn.ReadMsgUnix(buf[:], nil)
		if err != nil {
			u.mu.Lock()
			if !u.closed {
				for port, p := range u.open {
					delete(u.open, port)
					p.Close()
				}
				u.conn.Close()
				u.closed = true
			}
			u.mu.Unlock()
			u.cmd.Wait()
			u.mu.Lock()
			u.exited = true
			u.mu.Unlock()
			return
		}
		if n < 2 {
			continue
		}
		u.mu.Lock()
		if !u.closed {
			port := uint16(buf[1])<<8 | uint16(buf[0])
			if p, ok := u.open[port]; ok {
				delete(u.open, port)
				p.Close()
			} else {
				p, err = addPort(port, srv)
				if err != nil {
					errStr := err.Error()
					b := make([]byte, 2, 2+len(errStr))
					b[0] = buf[0]
					b[1] = buf[1]
					b = append(b, errStr...)
					u.conn.WriteMsgUnix(b, nil, nil)
				} else {
					u.open[port] = p
					u.conn.WriteMsgUnix(buf[:], nil, nil)
				}
			}
		}
		u.mu.Unlock()
	}
}

// Error
var (
	ErrClosed = errors.New("closed")
)
