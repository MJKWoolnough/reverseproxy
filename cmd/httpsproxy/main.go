package main

import (
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

type serverNames []string

func (s *serverNames) String() string {
	return ""
}

func (s *serverNames) Set(serverName string) error {
	*s = append(*s, serverName)
	return nil
}

func copyConn(a io.Writer, b io.Reader, wg *sync.WaitGroup) {
	wg.Add(1)
	io.Copy(a, b)
	wg.Done()
}

func proxyConn(c net.Conn, p string, wg *sync.WaitGroup) {
	pc, err := net.Dial("tcp", p)
	if err != nil {
		c.Close()
		return
	}
	go copyConn(c, pc, wg)
	go copyConn(pc, c, wg)
}

func proxySSL(l net.Listener, p string, wg *sync.WaitGroup) {
	for {
		c, err := l.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				continue
			}
			return
		}
		go proxyConn(c, p, wg)
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
		proxy       string
		wg          sync.WaitGroup
		server      http.Server
	)
	flag.Var(&serverNames, "s", "server name(s) for TLS")
	flag.StringVar(&proxy, "p", "proxy address")
	flag.Parse()
	if len(serverNames) == 0 {
		return errors.New("need server name")
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
	}), proxy, &wg)
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
