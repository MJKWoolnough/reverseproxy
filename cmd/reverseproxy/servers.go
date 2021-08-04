package main

type servers map[string]server

type server struct {
	Redirects map[uint64]redirect `json:"redirects"`
	Commands  map[uint64]command  `json:"commands"`
	nextRID   uint64              `json:"-"`
	nextCID   uint64              `json:"-"`
}

type redirect struct {
	From uint16 `json:"from"`
	To   uint16 `json:"to"`
}

type command struct {
	Exe    string            `json:"exe"`
	Params []string          `json:"params"`
	Env    map[string]string `json:"env"`
}
