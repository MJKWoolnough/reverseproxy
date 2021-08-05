package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"golang.org/x/net/websocket"
)

type hash [sha256.Size]byte

func (h *hash) MarshalJSON() ([]byte, error) {
	r := make([]byte, sha256.Size<<1+2)
	r[0] = '"'
	r[sha256.Size<<1+1] = '"'
	for n, b := range *h {
		if t := b >> 4; t > 9 {
			r[n<<1] = 'A' - 10 + t
		} else {
			r[n<<1] = '0' + t
		}
		if t := b & 15; t > 9 {
			r[n<<1+1] = 'A' - 10 + t
		} else {
			r[n<<1+1] = '0' + t
		}
	}
	return r, nil
}

var ErrInvalidPasswordHash = errors.New("invalid password hash")

func (h *hash) UnmarshalJSON(data []byte) error {
	if len(data) != sha256.Size<<1+2 || data[0] != '"' || data[sha256.Size<<1+1] != '"' {
		return ErrInvalidPasswordHash
	}
	for n, b := range data[1 : sha256.Size<<1+1] {
		var v byte
		if b >= '0' && b <= '9' {
			v = b - '0'
		} else if b >= 'A' && b <= 'F' {
			v = b - 'A' + 10
		} else if b >= 'a' && b <= 'f' {
			v = b - 'a' + 10
		} else {
			return ErrInvalidPasswordHash
		}
		if n&1 == 0 {
			(*h)[n>>1] = v << 4
		} else {
			(*h)[n>>1] |= v
		}
	}
	return nil
}

type config struct {
	Port     uint16
	Username string
	Password hash
	Servers  servers
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
	var (
		configFile string
		define     bool
	)
	flag.StringVar(&configFile, "c", "", "config file")
	flag.BoolVar(&define, "d", false, "define settings for config file")
	flag.Parse()
	if configFile == "" {
		return errors.New("no config file specified")
	}
	if define {
		return defineConfig(configFile)
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
	c.Servers.Init()
	var s = http.Server{
		Handler: &c,
	}
	go s.Serve(l)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
	signal.Stop(sc)
	close(sc)
	s.Close()
	c.Servers.Shutdown()
	return nil
}

func defineConfig(configFile string) error {
	var c config
	f, err := os.Open(configFile)
	if err == nil {
		json.NewDecoder(f).Decode(&c)
		f.Close()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error opening config file: %w", err)
	}
	r := bufio.NewReader(os.Stdin)
	var skipPort, skipCredentials bool
	if c.Port != 0 {
		if err := getInput(r, fmt.Sprintf("Do you want to set a new management port (%d)? Y/N: ", c.Port), func(ans string) bool {
			switch ans {
			case "Y", "y":
				skipPort = true
			case "N", "n":
			default:
				return false
			}
			return true
		}); err != nil {
			return err
		}
	}
	if !skipPort {
		if err := getInput(r, "Please enter a port number for the management console (1-65535): ", func(ans string) bool {
			p, err := strconv.ParseUint(ans, 10, 16)
			if err != nil {
				return false
			}
			c.Port = uint16(p)
			return true
		}); err != nil {
			return err
		}
	}
	if c.Username != "" {
		if err := getInput(r, "Do you want to set new management credentials? Y/N: ", func(ans string) bool {
			switch ans {
			case "Y", "y":
				skipCredentials = true
			case "N", "n":
			default:
				return false
			}
			return true
		}); err != nil {
			return err
		}
	}
	if !skipCredentials {
		if err := getInput(r, "Username: ", func(ans string) bool {
			if ans == "" {
				return false
			}
			c.Username = ans
			return true
		}); err != nil {
			return err
		}
		if err := getInput(r, "Password: ", func(ans string) bool {
			if ans == "" {
				return false
			}
			c.Password = sha256.Sum256([]byte(ans))
			return true
		}); err != nil {
			return err
		}
	}
	if !skipPort && !skipCredentials {
		f, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("error creating new config file: %w", err)
		}
		if err = json.NewEncoder(f).Encode(&c); err != nil {
			return fmt.Errorf("error writing config file: %w", err)
		}
		if err = f.Close(); err != nil {
			return fmt.Errorf("error closing config file: %w", err)
		}
	}
	return nil
}

func getInput(r *bufio.Reader, question string, checkFn func(string) bool) error {
	for {
		fmt.Print(question)
		ans, err := r.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error returned when reading stdin: %w", err)
		}
		if checkFn(ans[:len(ans)-1]) {
			return nil
		}
		fmt.Println("\nDid not understand response")
	}
}
