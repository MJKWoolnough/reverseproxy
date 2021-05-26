package reverseproxy

import (
	"net"
	"os"
	"os/exec"
	"syscall"

	"vimagination.zapto.org/byteio"
)

const maxBufSize = 1<<16 + 1<<16 + 2 + 2 + 1

type UnixServer struct {
}

func (p *Proxy) createUnixConn(cmd *exec.Cmd) error {
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
			buf [maxBufSize]byte
			r   byteio.StickyLittleEndianReader
		)
		for {
			n, _, _, _, err := conn.ReadMsgUnix(buf[:], nil)
			r.Reader = buf[:n]
			if n == 2 {
				socketID := r.ReadUint16()
				// close socket
			} else {
				network := r.ReadString16()
				address := r.ReadString16()
				isTLS := r.ReadBool()
			}
			// read unix conn, get listen details
			// read unix conn, get close details
			// register listener
			// listen on listener -> forward close/data
		}
	}()
}
