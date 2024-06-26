package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/crypto/acme/autocert"
	"vimagination.zapto.org/form"
	"vimagination.zapto.org/httpgzip"
	"vimagination.zapto.org/reverseproxy/unixconn"
)

type http2https struct {
	http.Handler
}

func (hh http2https) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil {
		url := "https://" + r.Host + r.URL.Path

		if len(r.URL.RawQuery) != 0 {
			url += "?" + r.URL.RawQuery
		}

		http.Redirect(w, r, url, http.StatusMovedPermanently)

		return
	}

	hh.Handler.ServeHTTP(w, r)
}

type paths []http.FileSystem

func (p *paths) String() string {
	return ""
}

func (p *paths) Set(path string) error {
	*p = append(*p, http.Dir(path))

	return nil
}

type serverNames []string

func (s *serverNames) String() string {
	return ""
}

func (s *serverNames) Set(serverName string) error {
	*s = append(*s, serverName)

	return nil
}

type contact struct {
	Template *template.Template
	From, To string
	Host     string
	Auth     smtp.Auth
}

type values struct {
	Name    string `form:"name,post"`
	Email   string `form:"email,required,post"`
	Phone   string `form:"phone,post"`
	Subject string `form:"subject,post"`
	Message string `form:"message,post"`
	Errors  form.ErrorMap
	Done    bool
}

func (c *contact) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var v values

	if r.Method == http.MethodPost {
		r.ParseForm()

		if r.Form.Get("submit") != "" {
			if err := form.Process(r, &v); err == nil {
				go smtp.SendMail(c.Host, c.Auth, c.From, []string{c.To}, []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: Message Received\r\n\r\nName: %s\nEmail: %s\nPhone: %s\nSubject: %s\nMessage: %s", c.To, c.From, v.Name, v.Email, v.Phone, v.Subject, v.Message)))

				v.Done = true
			} else {
				v.Errors = err.(form.ErrorMap)
			}
		}
	}

	c.Template.Execute(w, &v)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
	}
}

func run() error {
	var (
		contactTmpl string
		sPaths      paths
		sNames      serverNames
	)

	flag.StringVar(&contactTmpl, "c", "", "contact form template")
	flag.Var(&sNames, "s", "server name(s) for TLS")
	flag.Var(&sPaths, "p", "server path")
	flag.Parse()

	if len(sPaths) == 0 {
		return errors.New("")
	}

	server := &http.Server{
		Handler: http.DefaultServeMux,
	}

	if contactTmpl != "" {
		from := os.Getenv("contactFrom")
		to := os.Getenv("contactTo")
		addr := os.Getenv("contactAddr")
		username := os.Getenv("contactUsername")
		password := os.Getenv("contactPassword")
		p := strings.IndexByte(addr, ':')
		addrNoPort := addr

		os.Unsetenv("contactFrom")
		os.Unsetenv("contactTo")
		os.Unsetenv("contactAddr")
		os.Unsetenv("contactUsername")
		os.Unsetenv("contactPassword")

		if p > 0 {
			addrNoPort = addrNoPort[:p]
		}

		http.Handle("/contact.html", &contact{
			Template: template.Must(template.ParseFiles(contactTmpl)),
			From:     from,
			To:       to,
			Host:     addr,
			Auth:     smtp.PlainAuth("", username, password, addrNoPort),
		})
	}

	http.Handle("/", httpgzip.FileServer(sPaths[0], sPaths[1:]...))

	l, err := unixconn.Listen("tcp", ":80")
	if err != nil {
		return errors.New("unable to open port 80")
	}

	if len(sNames) > 0 {
		tl, err := unixconn.Listen("tcp", ":443")
		if err != nil {
			return errors.New("unable to open port 443")
		}

		leManager := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache("./certcache/"),
			HostPolicy: autocert.HostWhitelist(sNames...),
		}
		server.Handler = leManager.HTTPHandler(http2https{server.Handler})
		server.TLSConfig = &tls.Config{
			GetCertificate: leManager.GetCertificate,
			NextProtos:     []string{"h2", "http/1.1"},
		}

		go server.ServeTLS(tl, "", "")
	}

	go func() {
		if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, os.Interrupt)

	<-sc

	signal.Stop(sc)
	close(sc)

	return server.Shutdown(context.Background())
}
