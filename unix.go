package reverseproxy

import (
	"net"
	"os"
	"os/exec"
	"syscall"
)

const maxBufSize = 1<<16 + 1<<16 + 2 + 2 + 1

type unixService struct {
	MatchServiceName
	conn *net.UnixConn
}

func (u *unixService) Transfer(buf []byte, conn net.Conn) error {
	if cf, ok := conn.(interface{ File() (*os.File, error) }); ok {
		f, err := cf.File()
		if err == nil {
			_, _, err = u.conn.WriteMsgUnix(buf, syscall.UnixRights(int(f.Fd())), nil)
		}
		return err
	}
	return nil
}

// RegisterCmd runs the given command and waits for incoming listeners from it
func RegisterCmd(msn MatchServiceName, cmd *exec.Cmd) error {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	fconn, _ := net.FileConn(os.NewFile(uintptr(fds[0]), ""))
	conn := fconn.(*net.UnixConn)
	cmd.ExtraFiles = append([]*os.File{}, os.NewFile(uintptr(fds[1]), ""))
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
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
				break
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
	}()
	return err
}
