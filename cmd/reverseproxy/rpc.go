package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
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
	broadcastRemoveRedirect
	broadcastRemoveCommand
	broadcastStartRedirect
	broadcastStartCommand
	broadcastStopRedirect
	broadcastStopCommand
	broadcastCommandStopped
	broadcastCommandError
)

type socket struct {
	*jsonrpc.Server
	conn *websocket.Conn
	id   uint64
}

var (
	connMu sync.Mutex
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
	case "removeRedirect":
		return s.removeRedirect(data)
	case "removeCommand":
		return s.removeCommand(data)
	case "startRedirect":
		return s.startRedirect(data)
	case "startCommand":
		return s.startCommand(data)
	case "stopRedirect":
		return s.stopRedirect(data)
	case "stopCommand":
		return s.stopCommand(data)
	case "getCommandPorts":
		return s.getCommandPorts(data)
	}

	return nil, nil
}

func buildInitialMessage() json.RawMessage {
	config.mu.RLock()

	buf := []byte{'['}
	f := true

	for name, server := range config.Servers {
		if f {
			f = false
		} else {
			buf = append(buf, ',')
		}

		buf = fmt.Appendf(buf, "[%q,[", name)
		first := true

		for id, redirect := range server.Redirects {
			if first {
				first = false
			} else {
				buf = append(buf, ',')
			}

			buf = fmt.Appendf(buf, "[%d,%d,%q,%t,%q,", id, redirect.From, redirect.To, redirect.Start, redirect.err)

			for n, m := range redirect.Match {
				if n > 0 {
					buf = append(buf, ',')
				}

				buf = fmt.Appendf(buf, "[%t,%q]", m.IsSuffix, m.Name)
			}

			buf = append(buf, ']')
		}

		buf = append(buf, ']', ',', '[')
		first = true

		for id, cmd := range server.Commands {
			if first {
				first = false
			} else {
				buf = append(buf, ',')
			}

			buf = fmt.Appendf(buf, "[%d,%q,[", id, cmd.Exe)

			for n, param := range cmd.Params {
				if n > 0 {
					buf = append(buf, ',')
				}

				buf = fmt.Appendf(buf, "%q", param)
			}

			buf = fmt.Appendf(buf, "],%q,{", cmd.WorkDir)
			o := true

			for key, value := range cmd.Env {
				if o {
					o = false
				} else {
					buf = append(buf, ',')
				}

				buf = fmt.Appendf(buf, "%q:%q", key, value)
			}

			buf = fmt.Appendf(buf, "},%d,%q,", cmd.status, cmd.err)

			if cmd.User != nil {
				buf = fmt.Appendf(buf, "{\"uid\":%d,\"gid\":%d}", cmd.User.UID, cmd.User.GID)
			} else {
				buf = append(buf, 'n', 'u', 'l', 'l')
			}

			for _, m := range cmd.Match {
				buf = fmt.Appendf(buf, ",[%t,%q]", m.IsSuffix, m.Name)
			}

			buf = append(buf, ']')
		}

		buf = append(buf, ']', ']')
	}

	buf = append(buf, ']')
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
	connMu.Lock()

	var dat json.RawMessage

	for c := range conns {
		if c.id != except {
			if len(dat) == 0 {
				dat = buildMessage(id, data)
			}

			go c.SendData(dat)
		}
	}

	connMu.Unlock()
}

type nameID struct {
	Server string `json:"server"`
	ID     uint64 `json:"id"`
}

func (s *socket) getRedirect(n nameID, fn func(*server, *redirect) error) error {
	config.mu.Lock()
	defer config.mu.Unlock()

	serv, ok := config.Servers[n.Server]
	if !ok {
		return ErrNoServer
	}

	r, ok := serv.Redirects[n.ID]
	if !ok {
		return ErrUnknownRedirect
	}

	if r.Start {
		return ErrServerRunning
	}

	if err := fn(serv, r); err != nil {
		return err
	}

	if err := saveConfig(); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	return nil
}

func (s *socket) getCommand(n nameID, fn func(*server, *command) error) error {
	config.mu.Lock()
	defer config.mu.Unlock()

	serv, ok := config.Servers[n.Server]
	if !ok {
		return ErrNoServer
	}

	c, ok := serv.Commands[n.ID]
	if !ok {
		return ErrUnknownCommand
	}

	if c.status == 1 {
		return ErrServerRunning
	}

	if err := fn(serv, c); err != nil {
		return err
	}

	if err := saveConfig(); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	return nil
}

func (s *socket) add(data json.RawMessage) (interface{}, error) {
	var name string

	if err := json.Unmarshal(data, &name); err != nil {
		return nil, err
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	if _, ok := config.Servers[name]; ok {
		return nil, ErrNameExists
	}

	config.Servers[name] = &server{
		Redirects: make(map[uint64]*redirect),
		Commands:  make(map[uint64]*command),
		name:      name,
	}

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	broadcast(broadcastAdd, data, s.id)

	return nil, nil
}

func (s *socket) rename(data json.RawMessage) (interface{}, error) {
	var name [2]string

	if err := json.Unmarshal(data, &name); err != nil {
		return nil, err
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	if _, ok := config.Servers[name[1]]; ok {
		return nil, ErrNameExists
	}

	serv, ok := config.Servers[name[0]]
	if !ok {
		return nil, ErrNoServer
	}

	delete(config.Servers, name[0])

	config.Servers[name[1]] = serv

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	broadcast(broadcastRename, data, s.id)

	return nil, nil
}

func (s *socket) remove(data json.RawMessage) (interface{}, error) {
	var name string

	if err := json.Unmarshal(data, &name); err != nil {
		return nil, err
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	serv, ok := config.Servers[name]
	if !ok {
		return nil, ErrNoServer
	}

	for _, r := range serv.Redirects {
		if r.Start {
			return nil, ErrServerRunning
		}
	}

	for _, c := range serv.Commands {
		if c.status != 0 {
			return nil, ErrServerRunning
		}
	}

	delete(config.Servers, name)

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	broadcast(broadcastRemove, data, s.id)

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
	defer config.mu.Unlock()

	serv, ok := config.Servers[ar.Server]
	if !ok {
		return nil, ErrNoServer
	}

	id := serv.addRedirect(ar.redirectData)

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	broadcast(broadcastAddRedirect, append(strconv.AppendUint(append(data[:len(data)-1], ",\"id\":"...), id, 10), '}'), s.id)

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
	defer config.mu.Unlock()

	serv, ok := config.Servers[ac.Server]
	if !ok {
		return nil, ErrNoServer
	}

	id := serv.addCommand(ac.commandData)

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	broadcast(broadcastAddCommand, append(strconv.AppendUint(append(data[:len(data)-1], ",\"id\":"...), id, 10), '}'), s.id)

	return id, nil
}

func (s *socket) modifyRedirect(data json.RawMessage) (interface{}, error) {
	var mr struct {
		nameID
		redirectData
	}

	if err := json.Unmarshal(data, &mr); err != nil {
		return nil, err
	}

	return nil, s.getRedirect(mr.nameID, func(_ *server, r *redirect) error {
		r.redirectData = mr.redirectData
		broadcast(broadcastModifyRedirect, data, s.id)

		return nil
	})
}

func (s *socket) modifyCommand(data json.RawMessage) (interface{}, error) {
	var mc struct {
		nameID
		commandData
	}

	if err := json.Unmarshal(data, &mc); err != nil {
		return nil, err
	}

	return nil, s.getCommand(mc.nameID, func(_ *server, c *command) error {
		c.commandData = mc.commandData
		broadcast(broadcastModifyCommand, data, s.id)

		return nil
	})
}

func (s *socket) removeRedirect(data json.RawMessage) (interface{}, error) {
	var rr nameID

	if err := json.Unmarshal(data, &rr); err != nil {
		return nil, err
	}

	return nil, s.getRedirect(rr, func(serv *server, _ *redirect) error {
		delete(serv.Redirects, rr.ID)
		broadcast(broadcastRemoveRedirect, data, s.id)

		return nil
	})
}

func (s *socket) removeCommand(data json.RawMessage) (interface{}, error) {
	var rc nameID

	if err := json.Unmarshal(data, &rc); err != nil {
		return nil, err
	}

	return nil, s.getCommand(rc, func(serv *server, _ *command) error {
		delete(serv.Commands, rc.ID)
		broadcast(broadcastRemoveCommand, data, s.id)

		return nil
	})
}

func (s *socket) startRedirect(data json.RawMessage) (interface{}, error) {
	var sr nameID

	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, err
	}

	return nil, s.getRedirect(sr, func(_ *server, r *redirect) error {
		r.Run()
		broadcast(broadcastStartRedirect, data, s.id)

		return nil
	})
}

func (s *socket) startCommand(data json.RawMessage) (interface{}, error) {
	var sc nameID

	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}

	return nil, s.getCommand(sc, func(_ *server, c *command) error {
		err := c.Run()
		if err != nil {
			broadcast(broadcastCommandError, append(strconv.AppendQuote(append(data[:len(data)-1], ',', '"', 'e', 'r', 'r', '"', ':'), err.Error()), '}'), s.id)
		} else {
			broadcast(broadcastStartCommand, data, s.id)
		}

		return err
	})
}

func (s *socket) stopRedirect(data json.RawMessage) (interface{}, error) {
	var sr nameID

	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, err
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	serv, ok := config.Servers[sr.Server]
	if !ok {
		return nil, ErrNoServer
	}

	r, ok := serv.Redirects[sr.ID]
	if !ok {
		return nil, ErrUnknownRedirect
	}

	if !r.Start {
		return nil, ErrServerNotRunning
	}

	r.Stop()
	broadcast(broadcastStopRedirect, data, s.id)

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	return nil, nil
}

func (s *socket) stopCommand(data json.RawMessage) (interface{}, error) {
	var sc nameID

	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	serv, ok := config.Servers[sc.Server]
	if !ok {
		return nil, ErrNoServer
	}

	c, ok := serv.Commands[sc.ID]
	if !ok {
		return nil, ErrUnknownCommand
	}

	if c.status != 1 {
		return nil, ErrServerNotRunning
	}

	c.Stop()
	broadcast(broadcastStopCommand, data, s.id)

	if err := saveConfig(); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	return nil, nil
}

func (s *socket) getCommandPorts(data json.RawMessage) (interface{}, error) {
	var cp nameID

	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	serv, ok := config.Servers[cp.Server]
	if !ok {
		return nil, ErrNoServer
	}

	c, ok := serv.Commands[cp.ID]
	if !ok {
		return nil, ErrUnknownCommand
	}

	if c.status != 1 {
		return nil, ErrServerNotRunning
	}

	ports := c.unixCmd.Status().Ports

	return ports, nil
}

var (
	ErrNameExists       = errors.New("name already exists")
	ErrNoServer         = errors.New("no server by that name exists")
	ErrServerRunning    = errors.New("cannot perform operation while server running")
	ErrServerNotRunning = errors.New("server not running")
	ErrUnknownRedirect  = errors.New("unknown redirect")
	ErrUnknownCommand   = errors.New("unknown command")
)
