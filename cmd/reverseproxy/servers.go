package main

import "sync"

type servers map[string]*server

func (s server) Init() {
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
		params: params,
		Env:    env,
	}
	s.mu.Unlock()
	return id
}

type redirect struct {
	From uint16 `json:"from"`
	To   uint16 `json:"to"`
}

func (r *redirect) Init() {}

type command struct {
	Exe    string            `json:"exe"`
	Params []string          `json:"params"`
	Env    map[string]string `json:"env"`
}

func (c *command) Init() {}
