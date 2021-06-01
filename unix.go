package reverseproxy

import (
	"net"
	"os"
	"os/exec"
	"syscall"
)

const maxBufSize = 1<<16 + 1<<16 + 2 + 2 + 1

type unixPacket struct {
	*socket
	*conn
}

func RegisterCmd(service service, cmd *exec.Cmd) error {
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
	ns := make(chan uint16)
	socket2ID := make(map[*socket]uint16)
	id2Socket := make(map[uint16]*socket)
	go func() {
		var buf [2]byte
		for {
			n, _, _, _, err := conn.ReadMsgUnix(buf[:], nil)
			if n < 2 {
				continue
			}
			port := uint16(byte[1]<<8) | uint16(byte[0])
			s, ok := id2Socket[port]
			if !ok {
				ns <- port
			} else {
				s.Close()
				// close socket
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
	return err
}
