package reverseproxy

import (
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type unixService struct {
	MatchServiceName
	conn *net.UnixConn
}

func (u *unixService) Transfer(buf []byte, conn *net.TCPConn) error {
	f, err := conn.File()
	if err == nil {
		_, _, err = u.conn.WriteMsgUnix(buf, syscall.UnixRights(int(f.Fd())), nil)
		errr := f.Close()
		if err == nil {
			err = errr
		}
	}
	return err
}

// UnixCmd holds the information required to control (close) a server and its
// resources
type UnixCmd struct {
	cmd  *exec.Cmd
	conn *net.UnixConn

	mu   sync.Mutex
	open map[uint16]*Port
}

// Close closes all ports for the server and sends a signal to the server to
// close
func (u *UnixCmd) Close() error {
	u.mu.Lock()
	for port, p := range u.open {
		delete(u.open, port)
		p.Close()
	}
	err := u.conn.Close()
	errr := u.cmd.Process.Signal(os.Interrupt)
	u.mu.Unlock()
	if err != nil {
		return err
	}
	return errr
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
	conn := fconn.(*net.UnixConn)
	f := os.NewFile(uintptr(fds[1]), "")
	cmd.ExtraFiles = append([]*os.File{}, f)
	err = cmd.Start()
	f.Close()
	if err != nil {
		return nil, err
	}
	u := &UnixCmd{
		cmd:  cmd,
		conn: conn,
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
			for port, p := range u.open {
				delete(u.open, port)
				p.Close()
			}
			u.conn.Close()
			u.mu.Unlock()
			return
		}
		if n < 2 {
			continue
		}
		u.mu.Lock()
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
		u.mu.Unlock()
	}
}
