package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
	"vimagination.zapto.org/memio"
)

const (
	broadcastList = -1 - iota
)

type socket struct {
	*jsonrpc.Server
	conn *websocket.Conn
	id   uint64
}

var (
	connMu sync.RWMutex
	conns  = make(map[*socket]struct{})
	nextID uint64
)

func NewConn(conn *websocket.Conn) {
	s := socket{
		conn: conn,
	}
	s.Server = jsonrpc.New(conn, &s)

	connMu.Lock()
	nextID++
	s.id = nextID
	conns[&s] = struct{}{}
	connMu.Unlock()

	s.SendData(buildInitialMessage())

	s.Handle()

	connMu.Lock()
	delete(conns, &s)
	connMu.Unlock()
}

func ShutdownRPC() {
	connMu.Lock()
	for c := range conns {
		c.conn.Close()
	}
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

func buildInitialMessage() json.RawMessage {
	config.mu.RLock()
	buf := memio.Buffer{'{'}
	f := true
	for name, server := range config.Servers {
		if f {
			f = false
		} else {
			buf = append(buf, ',')
		}
		fmt.Fprintf(&buf, "%q:[{", name)
		first := true
		for id, redirect := range server.Redirects {
			if first {
				first = false
			} else {
				buf = append(buf, ',')
			}
			fmt.Fprintf(&buf, "\"%d\":[%d,%q,%t,%q]", id, redirect.From, redirect.To, redirect.Start, redirect.err)
		}
		buf = append(buf, '}', ',', '{')
		first = true
		for id, cmd := range server.Commands {
			if first {
				first = false
			} else {
				buf = append(buf, ',')
			}
			fmt.Fprintf(&buf, "\"%d\":[%s, [", id, cmd.Exe)
			for n, param := range cmd.Params {
				if n > 0 {
					buf = append(buf, ',')
				}
				fmt.Fprintf(&buf, "%q", param)
			}
			buf = append(buf, ']', ',', '{')
			o := true
			for key, value := range cmd.Env {
				if o {
					o = false
				} else {
					buf = append(buf, ',')
				}
				fmt.Fprintf(&buf, "%q:%q", key, value)
			}
			fmt.Fprintf(&buf, "},%d,%q]", cmd.status, cmd.err)
		}
		buf = append(buf, '}', ']')
	}
	buf = append(buf, '}')
	config.mu.RUnlock()
	return buildMessage(-1, json.RawMessage(buf))
}

const broadcastStart = "{\"id\": -0,\"result\":"

func buildMessage(id int, data json.RawMessage) json.RawMessage {
	l := len(broadcastStart) + len(data) + 1
	dat := make(json.RawMessage, l)
	copy(dat, broadcastStart)
	copy(dat[len(broadcastStart):], data)
	id = -id
	if id > 9 {
		dat[6] = '-'
		dat[7] = byte('0' + id/10)
	}
	dat[8] = byte('0' + id%10)
	dat[l-1] = '}'
	return dat
}

func broadcast(id int, data json.RawMessage, except uint64) {
	dat := buildMessage(id, data)
	connMu.RLock()
	for c := range conns {
		if c.id != except {
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
