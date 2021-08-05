package main

import (
	"net"
	"os/exec"
	"sync"

	"vimagination.zapto.org/reverseproxy"
)

type servers map[string]*server

func (s servers) Init() {
	for _, server := range s {
		server.Init()
	}
}

type server struct {
	mu        sync.RWMutex
	Redirects map[uint64]*redirect `json:"redirects"`
	Commands  map[uint64]*command  `json:"commands"`
	lastRID   uint64
	lastCID   uint64
}

func (s *server) Init() {
	for id, r := range s.Redirects {
		r.Init()
		if id > s.lastRID {
			s.lastRID = id
		}
	}
	for id, c := range s.Commands {
		c.Init()
		if id > s.lastCID {
			s.lastCID = id
		}
	}
}

func (s *server) addRedirect(from uint16, to string) uint64 {
	s.mu.Lock()
	s.lastRID++
	id := s.lastRID
	s.Redirects[id] = &redirect{
		From: from,
		To:   to,
	}
	s.mu.Unlock()
	return id
}

func (s *server) addCommand(exe string, params []string, env map[string]string) uint64 {
	s.mu.Lock()
	s.lastCID++
	id := s.lastCID
	s.Commands[id] = &command{
		Exe:    exe,
		Params: params,
		Env:    env,
	}
	s.mu.Unlock()
	return id
}

type redirect struct {
	mu               sync.RWMutex
	From             uint16  `json:"from"`
	To               string  `json:"to"`
	Match            []match `json:"match"`
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
	r.mu.Lock()
	if r.From > 0 && r.To != "" {
		addr, err := net.ResolveTCPAddr("tcp", r.To)
		if err != nil {
			r.err = err.Error()
		} else if r.port, err = reverseproxy.AddRedirect(r.matchServiceName, r.From, addr); err != nil {
			r.err = err.Error()
		}
	}
	r.mu.Unlock()
}

type command struct {
	mu               sync.RWMutex
	Exe              string            `json:"exe"`
	Params           []string          `json:"params"`
	Env              map[string]string `json:"env"`
	Match            []match           `json:"match"`
	matchServiceName reverseproxy.MatchServiceName
	Start            bool `json:"start"`
	unixCmd          *reverseproxy.UnixCmd
	err              string
}

func (c *command) Init() {
	c.matchServiceName = makeMatchService(c.Match)
	if c.Start {
		c.Run()
	}
}

func (c *command) Run() {
	c.mu.Lock()
	cmd := exec.Command(c.Exe, c.Params...)
	cmd.Env = make([]string, 0, len(c.Env))
	for k, v := range c.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var err error
	if c.unixCmd, err = reverseproxy.RegisterCmd(c.matchServiceName, cmd); err != nil {
		c.err = err.Error()
	}
	c.mu.Unlock()
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
