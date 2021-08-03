package main

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
)

const (
	broadcastList = -1 - iota
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
		return add(data)
	case "modify":
		return modify(data)
	case "start":
		return start(data)
	case "stop":
		return stop(data)
	case "remove":
		return remove(data)
	}
	return nil, nil
}

const broadcastStart = "{\"id\": -0,\"result\":"

func broadcast(id int, data json.RawMessage, except ID) {
	l := len(broadcastStart) + len(data) + 1
	dat := make([]byte, l)
	copy(dat, broadcastStart)
	copy(dat[len(broadcastStart):], data)
	id = -id
	if id > 9 {
		dat[6] = '-'
		dat[7] = byte('0' + id/10)
	}
	dat[8] = byte('0' + id%10)
	dat[l-1] = '}'
	connMu.RLock()
	for c := range conns {
		if c.ID != except {
			go c.SendData(dat)
		}
	}
	connMu.RUnlock()
}

func add(data json.RawMessage) (interface{}, error) {
	return nil, nil
}

func modify(data json.RawMessage) (interface{}, error) {
	return nil, nil
}

func start(data json.RawMessage) (interface{}, error) {
	return nil, nil
}

func stop(data json.RawMessage) (interface{}, error) {
	return nil, nil
}

func remove(data json.RawMessage) (interface{}, error) {
	return nil, nil
}
