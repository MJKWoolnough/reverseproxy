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
)

type config struct {
	Port     uint16
	Username string
	Password [sha256.Size]byte
}

type auth struct {
	Config *config
	Mux    *http.ServeMux
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

func (a *auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u, p, ok := r.BasicAuth(); ok && u == a.Config.Username && sha256.Sum256([]byte(p)) == a.Config.Password {
		a.Mux.ServeHTTP(w, r)
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
	var (
		m http.ServeMux
		s = http.Server{
			Handler: &auth{Config: &c, Mux: &m},
		}
	)
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
