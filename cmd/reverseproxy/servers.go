package main

type servers map[string]*server

func (s server) Init() {
	for _, server := range s {
		server.Init()
	}
}

type server struct {
	Redirects map[uint64]redirect `json:"redirects"`
	Commands  map[uint64]command  `json:"commands"`
	lastRID   uint64              `json:"-"`
	lastCID   uint64              `json:"-"`
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
