package reverseproxy

import (
	"net"
	"os"
	"os/exec"
	"syscall"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/memio"
)

const maxBufSize = 1<<16 + 1<<16 + 2 + 2 + 1

type unixPacket struct {
	*socket
	*conn
}

type unixServer chan unixPacket

func (u unixServer) Shutdown() {
	close(u)
}

func (u unixServer) Transfer(socket *socket, conn *conn) {
	u <- unixPacket{
		socket,
		conn,
	}
}

type newSocket struct {
	socketID         uint16
	network, address string
	isTLS            bool
}

func (p *Proxy) createUnixConn(cmd *exec.Cmd) (unixServer, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	fconn, _ := net.FileConn(os.NewFile(uintptr(fds[0]), ""))
	conn := fconn.(*net.UnixConn)
	cmd.ExtraFiles = append([]*os.File{}, os.NewFile(uintptr(fds[1]), ""))
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	us := make(unixServer)
	ns := make(chan newSocket)
	socket2ID := make(map[*socket]uint16)
	id2Socket := make(map[uint16]*socket)
	var lastSocketID uint16
	go func() {
		var (
			buf [maxBufSize]byte
			r   byteio.StickyLittleEndianReader
		)
		for {
			n, _, _, _, err := conn.ReadMsgUnix(buf[:], nil)

			b := memio.Buffer(buf[:n])
			r.Reader = &b
			if n == 2 {
				socketID := r.ReadUint16()
				// close socket
			} else {
				network := r.ReadString16()
				address := r.ReadString16()
				isTLS := r.ReadBool()
				lastSocketID++
				ns <- newSocket{lastSocketID, network, address, isTLS}
			}
			// read unix conn, get listen details
			// read unix conn, get close details
			// register listener
			// listen on listener -> forward close/data
		}
	}()
	go func() {
		for {
			select {
			case c, ok := <-us:
				if ok {
					if id, ok := socket2ID[c.socket]; ok {

					} else {

					}
				} else {

				}
			case s := <-ns:

			}
		}
	}()
	return us, err
}
