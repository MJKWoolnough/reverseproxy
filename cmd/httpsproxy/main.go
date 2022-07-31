package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"golang.org/x/crypto/acme/autocert"
	"vimagination.zapto.org/reverseproxy/unixconn"
)

const bufSize = 1<<16 + 16

var (
	forwardHeader bool
	headerPool    = sync.Pool{
		New: func() interface{} {
			return &[bufSize]byte{}
		},
	}
	eol     = []byte{'\r', '\n'}
	forward = []byte{'\r', '\n', 'F', 'o', 'r', 'w', 'a', 'r', 'd', 'e', 'd', ':', ' ', 'f', 'o', 'r', '='}
	proxy   string
	wg      sync.WaitGroup
)

type serverNames []string

func (s *serverNames) String() string {
	return ""
}

func (s *serverNames) Set(serverName string) error {
	*s = append(*s, serverName)
	return nil
}

func copyConn(a io.Writer, b io.Reader) {
	io.Copy(a, b)
	wg.Done()
}

func proxyConn(c net.Conn) {
	defer wg.Done()
	pc, err := net.Dial("tcp", proxy)
	if err != nil {
		c.Close()
		return
	}
	if forwardHeader {
		buf := headerPool.Get().(*[bufSize]byte)
		n := 0
		l := 0
		for {
			m, err := c.Read(buf[n:])
			n += m
			if l = bytes.Index(buf[:n], eol); l >= 0 {
				pc.Write(buf[:l])
				pc.Write(forward)
				io.WriteString(pc, c.RemoteAddr().String())
				break
			}
			if err != nil {
				return
			}
		}
		pc.Write(buf[l:n])
		for p := range buf[:n] {
			buf[p] = 0
		}
		headerPool.Put(buf)
	}
	wg.Add(2)
	go copyConn(c, pc)
	go copyConn(pc, c)
}

func proxySSL(l net.Listener) {
	wg.Add(1)
	for {
		c, err := l.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				continue
			}
			wg.Done()
			return
		}
		wg.Add(1)
		go proxyConn(c)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
	}
}

func run() error {
	var (
		serverNames serverNames
		server      http.Server
	)
	flag.Var(&serverNames, "s", "server name(s) for TLS")
	flag.StringVar(&proxy, "p", "", "proxy address")
	flag.BoolVar(&forwardHeader, "f", false, "add forward headers")
	flag.Parse()
	if len(serverNames) == 0 {
		return errors.New("need server name")
	}
	if proxy == "" {
		return errors.New("need proxy address")
	}
	leManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("./certcache/"),
		HostPolicy: autocert.HostWhitelist(serverNames...),
	}
	l, err := unixconn.Listen("tcp", ":80")
	if err != nil {
		return errors.New("unable to open port 80")
	}
	sl, err := unixconn.Listen("tcp", ":443")
	if err != nil {
		return errors.New("unable to open port 443")
	}
	server.Handler = leManager.HTTPHandler(nil)
	go proxySSL(tls.NewListener(sl, &tls.Config{
		GetCertificate: leManager.GetCertificate,
		NextProtos:     []string{"http/1.1"},
	}))
	go server.Serve(l)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
	signal.Stop(sc)
	close(sc)
	server.Shutdown(context.Background())
	sl.Close()
	wg.Wait()
	return nil
}
