package main

type servers map[string]server

type server struct {
	Redirects []redirect
	Commands  []command
}

type redirect struct {
	From, To uint16
}

type command struct {
	Exe    string
	Params []string
	Env    map[string]string
}
