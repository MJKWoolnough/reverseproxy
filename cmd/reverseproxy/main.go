package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"golang.org/x/net/websocket"
)

type config struct {
	Port     uint16
	Username string
	Password [sha256.Size]byte
}

var unauthorised = []byte(`<html>
	<head>
		<title>Unauthorised</title>
	</head>
	<body>
		<h1>Not Authorised</h1>
	</body>
</html>
`)

func (c *config) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u, p, ok := r.BasicAuth(); ok && u == c.Username && sha256.Sum256([]byte(p)) == c.Password {
		switch r.URL.Path {
		case "/":
			index(w, r)
		case "/socket":
			websocket.Handler(NewConn).ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
		return
	}
	w.Header().Set("WWW-Authenticate", "Basic realm=\"Enter Credentials\"")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(unauthorised)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var configFile, logFile string
	flag.StringVar(&configFile, "c", "", "config file")
	flag.StringVar(&logFile, "l", "-", "log file")
	flag.Parse()
	if configFile == "" {
		return errors.New("no config file specified")
	}
	var c config
	f, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("error while opening config file: %w", err)
	}
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return fmt.Errorf("error while decoding config file: %w", err)
	}
	f.Close()
	l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(c.Port)})
	if err != nil {
		return fmt.Errorf("error opening management interface port: %w", err)
	}
	var s = http.Server{
		Handler: &c,
	}
	go s.Serve(l)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
	signal.Stop(sc)
	close(sc)
	if err = s.Close(); err != nil {
		return fmt.Errorf("error closing management interface: %w", err)
	}
	return nil
}
