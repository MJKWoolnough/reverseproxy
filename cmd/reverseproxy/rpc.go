package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
	"vimagination.zapto.org/memio"
)

const (
	broadcastList = -1 - iota
	broadcastAdd
	broadcastRename
	broadcastRemove
	broadcastAddRedirect
	broadcastAddCommand
	broadcastModifyRedirect
	broadcastModifyCommand
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
		return s.add(data)
	case "rename":
		return s.rename(data)
	case "remove":
		return s.remove(data)
	case "addRedirect":
		return s.addRedirect(data)
	case "addCommand":
		return s.addCommand(data)
	case "modifyRedirect":
		return s.modifyRedirect(data)
	case "modifyCommand":
		return s.modifyCommand(data)
	case "start":
		return start(data)
	case "stop":
		return stop(data)
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

func (s *socket) add(data json.RawMessage) (interface{}, error) {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return nil, err
	}
	config.mu.Lock()
	if _, ok := config.Servers[name]; ok {
		config.mu.Unlock()
		return nil, ErrNameExists
	}
	config.Servers[name] = &server{name: name}
	saveConfig()
	broadcast(broadcastAdd, data, s.id)
	config.mu.Unlock()
	return nil, nil
}

func (s *socket) rename(data json.RawMessage) (interface{}, error) {
	var name [2]string
	if err := json.Unmarshal(data, &name); err != nil {
		return nil, err
	}
	config.mu.Lock()
	if _, ok := config.Servers[name[1]]; ok {
		return nil, ErrNameExists
	}
	serv, ok := config.Servers[name[0]]
	if !ok {
		config.mu.Unlock()
		return nil, ErrNoServer
	}
	delete(config.Servers, name[0])
	config.Servers[name[1]] = serv
	saveConfig()
	broadcast(broadcastRename, data, s.id)
	config.mu.Unlock()
	return nil, nil
}

func (s *socket) remove(data json.RawMessage) (interface{}, error) {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return nil, err
	}
	config.mu.Lock()
	serv, ok := config.Servers[name]
	if !ok {
		config.mu.Unlock()
		return nil, ErrNoServer
	}
	for _, r := range serv.Redirects {
		if r.Start {
			config.mu.Unlock()
			return nil, ErrServerRunning
		}
	}
	for _, c := range serv.Commands {
		if c.status != 0 {
			config.mu.Unlock()
			return nil, ErrServerRunning
		}
	}
	delete(config.Servers, name)
	saveConfig()
	broadcast(broadcastRemove, data, s.id)
	config.mu.Unlock()
	return nil, nil
}

func (s *socket) addRedirect(data json.RawMessage) (interface{}, error) {
	var ar struct {
		Server string `json:"server"`
		redirectData
	}
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, err
	}
	config.mu.Lock()
	serv, ok := config.Servers[ar.Server]
	if !ok {
		config.mu.Unlock()
		return nil, ErrNoServer
	}
	id := serv.addRedirect(ar.redirectData)
	saveConfig()
	broadcast(broadcastAddRedirect, append(strconv.AppendUint(append(data[:len(data)-1], ",\"id\":"...), id, 10), '}'), s.id)
	config.mu.Unlock()
	return id, nil
}

func (s *socket) addCommand(data json.RawMessage) (interface{}, error) {
	var ac struct {
		Server string `json:"server"`
		commandData
	}
	if err := json.Unmarshal(data, &ac); err != nil {
		return nil, err
	}
	config.mu.Lock()
	serv, ok := config.Servers[ac.Server]
	if !ok {
		config.mu.Unlock()
		return nil, ErrNoServer
	}
	id := serv.addCommand(ac.commandData)
	saveConfig()
	broadcast(broadcastAddCommand, append(strconv.AppendUint(append(data[:len(data)-1], ",\"id\":"...), id, 10), '}'), s.id)
	config.mu.Unlock()
	return id, nil
}

func (s *socket) modifyRedirect(data json.RawMessage) (interface{}, error) {
	var mr struct {
		Server string `json:"server"`
		ID     uint64 `json:"id"`
		redirectData
	}
	if err := json.Unmarshal(data, &mr); err != nil {
		return nil, err
	}
	config.mu.Lock()
	serv, ok := config.Servers[mr.Server]
	if !ok {
		config.mu.Unlock()
		return nil, ErrNoServer
	}
	r, ok := serv.Redirects[mr.ID]
	if !ok {
		config.mu.Unlock()
		return nil, ErrUnknownRedirect
	}
	if r.Start {
		config.mu.Unlock()
		return nil, ErrServerRunning
	}
	r.redirectData = mr.redirectData
	saveConfig()
	broadcast(broadcastModifyRedirect, data, s.id)
	config.mu.Unlock()
	return nil, nil
}

func (s *socket) modifyCommand(data json.RawMessage) (interface{}, error) {
	var mc struct {
		Server string `json:"server"`
		ID     uint64 `json:"id"`
		commandData
	}
	if err := json.Unmarshal(data, &mc); err != nil {
		return nil, err
	}
	config.mu.Lock()
	serv, ok := config.Servers[mc.Server]
	if !ok {
		config.mu.Unlock()
		return nil, ErrNoServer
	}
	c, ok := serv.Commands[mc.ID]
	if !ok {
		config.mu.Unlock()
		return nil, ErrUnknownCommand
	}
	if c.Start {
		config.mu.Unlock()
		return nil, ErrServerRunning
	}
	c.commandData = mc.commandData
	saveConfig()
	broadcast(broadcastModifyCommand, data, s.id)
	config.mu.Unlock()
	return nil, nil
}

func start(data json.RawMessage) (interface{}, error) {
	return nil, nil
}

func stop(data json.RawMessage) (interface{}, error) {
	return nil, nil
}

var (
	ErrNameExists      = errors.New("name already exists")
	ErrNoServer        = errors.New("no server by that name exists")
	ErrServerRunning   = errors.New("cannot perform operation while server running")
	ErrUnknownRedirect = errors.New("unknown redirect")
	ErrUnknownCommand  = errors.New("unknown command")
)
