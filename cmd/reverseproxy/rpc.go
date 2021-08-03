package main

import (
	"encoding/json"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
)

type socket struct {
	*jsonrpc.Server
}

func NewConn(conn *websocket.Conn) {
	var s socket
	s.Server = jsonrpc.New(conn, &s)
	s.Handle()
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
