package main

type servers map[string]server

type server struct {
	Redirects []redirect `json:"redirects"`
	Commands  []command  `json:"commands"`
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
