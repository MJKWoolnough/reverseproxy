package main

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
)

type socket struct {
	*jsonrpc.Server
	id uint64
}

var (
	connMu sync.RWMutex
	conns  = make(map[*socket]struct{})
	nextID uint64
)

func NewConn(conn *websocket.Conn) {
	var s socket
	s.Server = jsonrpc.New(conn, &s)

	connMu.Lock()
	nextID++
	s.id = nextID
	conns[&s] = struct{}{}
	connMu.Unlock()

	s.Handle()

	connMu.Lock()
	delete(conns, &s)
	connMu.Unlock()
}

func (s *socket) HandleRPC(method string, data json.RawMessage) (interface{}, error) {
	switch method {
	case "add":
	case "modify":
	case "start":
	case "stop":
	case "remove":
	}
	return nil, nil
}
