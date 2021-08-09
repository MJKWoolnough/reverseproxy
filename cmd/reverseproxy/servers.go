package main

import (
	"net"
	"os/exec"
	"syscall"

	"vimagination.zapto.org/reverseproxy"
)

type servers map[string]*server

func (s servers) Init() {
	for name, server := range s {
		server.Init(name)
	}
}

func (s servers) Shutdown() {
	for _, server := range s {
		server.Shutdown()
	}
}

type server struct {
	Redirects map[uint64]*redirect `json:"redirects"`
	Commands  map[uint64]*command  `json:"commands"`
	name      string
	lastRID   uint64
	lastCID   uint64
}

func (s *server) Init(name string) {
	s.name = name
	for id, r := range s.Redirects {
		r.Init()
		if id > s.lastRID {
			s.lastRID = id
		}
	}
	for id, c := range s.Commands {
		c.Init(s)
		if id > s.lastCID {
			s.lastCID = id
		}
	}
}

func (s *server) addRedirect(rd redirectData) uint64 {
	s.lastRID++
	id := s.lastRID
	s.Redirects[id] = &redirect{
		redirectData:     rd,
		matchServiceName: makeMatchService(rd.Match),
	}
	saveConfig()
	return id
}

func (s *server) addCommand(cd commandData) uint64 {
	s.lastCID++
	id := s.lastCID
	s.Commands[id] = &command{
		commandData:      cd,
		matchServiceName: makeMatchService(cd.Match),
	}
	saveConfig()
	return id
}

func (s *server) Shutdown() {
	for _, r := range s.Redirects {
		r.Shutdown()
	}
	for _, c := range s.Commands {
		c.Shutdown()
	}
}

type redirectData struct {
	From  uint16  `json:"from"`
	To    string  `json:"to"`
	Match []match `json:"match"`
}

type redirect struct {
	redirectData
	matchServiceName reverseproxy.MatchServiceName
	Start            bool `json:"start"`
	port             *reverseproxy.Port
	err              string
}

func (r *redirect) Init() {
	r.matchServiceName = makeMatchService(r.Match)
	if r.Start {
		r.Run()
	}
}

func (r *redirect) Run() {
	if r.From > 0 && r.To != "" && r.port == nil {
		addr, err := net.ResolveTCPAddr("tcp", r.To)
		if err != nil {
			r.err = err.Error()
		} else if r.port, err = reverseproxy.AddRedirect(r.matchServiceName, r.From, addr); err != nil {
			r.err = err.Error()
		} else {
			r.Start = true
			saveConfig()
		}
	}
}

func (r *redirect) Stop() {
	r.Start = false
	r.Shutdown()
	saveConfig()
}

func (r *redirect) Shutdown() {
	if r.port != nil {
		r.port.Close()
		r.port = nil
	}
}

type user struct {
	UID uint32 `json:"uid"`
	GID uint32 `json:"gid"`
}

type commandData struct {
	Exe    string            `json:"exe"`
	Params []string          `json:"params"`
	Env    map[string]string `json:"env"`
	Match  []match           `json:"match"`
	User   *user             `json:"user,omitempty"`
}

type command struct {
	commandData
	matchServiceName reverseproxy.MatchServiceName
	Start            bool `json:"start"`
	status           int
	unixCmd          *reverseproxy.UnixCmd
	err              string
	server           *server
}

func (c *command) Init(server *server) {
	c.server = server
	c.matchServiceName = makeMatchService(c.Match)
	if c.Start {
		c.Run()
	}
}

func (c *command) Run() {
	if c.unixCmd == nil {
		cmd := exec.Command(c.Exe, c.Params...)
		cmd.Env = make([]string, 0, len(c.Env))
		for k, v := range c.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		if c.User != nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: c.User.UID,
					Gid: c.User.GID,
				},
			}
		}
		uc, err := reverseproxy.RegisterCmd(c.matchServiceName, cmd)
		if err != nil {
			c.err = err.Error()
			c.status = 0
		} else {
			c.status = 1
			c.unixCmd = uc
			go func() {
				cmd.Wait()
				config.mu.Lock()
				if c.unixCmd == uc {
					c.status = 0
				}
				config.mu.Unlock()
			}()
			c.Start = true
			saveConfig()
		}
	}
}

func (c *command) Stop() {
	c.Start = false
	c.Shutdown()
	saveConfig()
}

func (c *command) Shutdown() {
	if c.unixCmd != nil {
		c.status = 2
		c.unixCmd.Close()
		c.unixCmd = nil
	}
}

type match struct {
	IsSuffix bool
	Name     string
}

func (m match) makeMatchService() reverseproxy.MatchServiceName {
	if m.IsSuffix {
		return reverseproxy.HostNameSuffix(m.Name)
	}
	return reverseproxy.HostName(m.Name)
}

func makeMatchService(match []match) reverseproxy.MatchServiceName {
	if len(match) == 0 {
		return none{}
	} else if len(match) == 1 {
		return match[0].makeMatchService()
	}
	ms := make(reverseproxy.Hosts, len(match))
	for n, m := range match {
		ms[n] = m.makeMatchService()
	}
	return ms
}

type none struct{}

func (none) MatchService(_ string) bool { return false }
