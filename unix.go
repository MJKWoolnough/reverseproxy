package reverseproxy

import (
	"net"
	"os"
	"os/exec"
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

// RegisterCmd runs the given command and waits for incoming listeners from it
func RegisterCmd(msn MatchServiceName, cmd *exec.Cmd) error {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	nf := os.NewFile(uintptr(fds[0]), "")
	fconn, err := net.FileConn(nf)
	if err := nf.Close(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	conn := fconn.(*net.UnixConn)
	f := os.NewFile(uintptr(fds[1]), "")
	cmd.ExtraFiles = append([]*os.File{}, f)
	err = cmd.Start()
	f.Close()
	if err != nil {
		return err
	}
	go runCmdLoop(msn, conn)
	return nil
}

func runCmdLoop(msn MatchServiceName, conn *net.UnixConn) {
	var (
		buf  [2]byte
		open = make(map[uint16]*Port)
		srv  = &unixService{
			MatchServiceName: msn,
			conn:             conn,
		}
	)
	for {
		n, _, _, _, err := conn.ReadMsgUnix(buf[:], nil)
		if err != nil {
			for _, p := range open {
				p.Close()
			}
			conn.Close()
			return
		}
		if n < 2 {
			continue
		}
		port := uint16(buf[1])<<8 | uint16(buf[0])
		if p, ok := open[port]; ok {
			delete(open, port)
			p.Close()
		} else {
			p, err = addPort(port, srv)
			if err != nil {
				errStr := err.Error()
				b := make([]byte, 2, 2+len(errStr))
				b[0] = buf[0]
				b[1] = buf[1]
				b = append(b, errStr...)
				conn.WriteMsgUnix(b, nil, nil)
				continue
			}
			open[port] = p
			conn.WriteMsgUnix(buf[:], nil, nil)
		}
	}
}
