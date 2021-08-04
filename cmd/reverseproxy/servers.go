package main

import (
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

func (s *server) addRedirect(from, to uint16) uint64 {
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
	From             uint16  `json:"from"`
	To               uint16  `json:"to"`
	Match            []match `json:"match"`
	matchServiceName reverseproxy.MatchServiceName
}

func (r *redirect) Init() {
	r.matchServiceName = makeMatchService(r.Match)
}

type command struct {
	Exe              string            `json:"exe"`
	Params           []string          `json:"params"`
	Env              map[string]string `json:"env"`
	Match            []match           `json:"match"`
	matchServiceName reverseproxy.MatchServiceName
}

func (c *command) Init() {
	c.matchServiceName = makeMatchService(c.Match)
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
