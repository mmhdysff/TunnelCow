package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"tunnelcow/internal/tunnel"

	"github.com/hashicorp/yamux"
)

type ClientSession struct {
	Conn        net.Conn
	Session     *yamux.Session
	Control     net.Conn
	ControlPort int
	Listeners   map[int]net.Listener
	Mu          sync.Mutex
	Debug       bool
}

func NewClientSession(conn net.Conn, session *yamux.Session, control net.Conn, controlPort int, debug bool) *ClientSession {
	return &ClientSession{
		Conn:        conn,
		Session:     session,
		Control:     control,
		ControlPort: controlPort,
		Listeners:   make(map[int]net.Listener),
		Debug:       debug,
	}
}

func (c *ClientSession) HandleControlLoop() {
	defer c.Cleanup()

	decoder := json.NewDecoder(c.Control)

	for {
		var msg tunnel.ControlMessage
		err := decoder.Decode(&msg)
		if err != nil {
			if err != io.EOF {
				log.Printf("Control read error: %v", err)
			}
			return
		}

		if c.Debug {
			log.Printf("Received msg type: %s", msg.Type)
		}

		switch msg.Type {
		case tunnel.MsgTypeReqBind:
			c.handleReqBind(msg.Payload)
		case tunnel.MsgTypeReqUnbind:
			c.handleReqUnbind(msg.Payload)
		case tunnel.MsgTypePing:

			_ = json.NewEncoder(c.Control).Encode(msg)
		case tunnel.MsgTypeReqDomainMap:
			c.handleReqDomainMap(msg.Payload)
		case tunnel.MsgTypeReqDomainUnmap:
			c.handleReqDomainUnmap(msg.Payload)
		}
	}
}

func (c *ClientSession) handleReqBind(payload json.RawMessage) {
	var req tunnel.ReqBindPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("Invalid REQ_BIND payload: %v", err)
		return
	}

	if req.PublicPort < 1 || req.PublicPort > 65535 {
		log.Printf("Invalid port %d", req.PublicPort)
		return
	}

	c.Mu.Lock()
	defer c.Mu.Unlock()

	if req.PublicPort == c.ControlPort {
		log.Printf("Security Alert: Client tried to bind Control Port %d. Action Blocked.", req.PublicPort)
		return
	}

	if _, exists := c.Listeners[req.PublicPort]; exists {
		if c.Debug {
			log.Printf("Port %d already bound", req.PublicPort)
		}
		return
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", req.PublicPort))
	if err != nil {
		log.Printf("Failed to bind port %d: %v", req.PublicPort, err)

		return
	}

	c.Listeners[req.PublicPort] = ln
	GlobalSessions.Register(req.PublicPort, c)
	if c.Debug {
		log.Printf("Bound public port %d", req.PublicPort)
	}

	go c.acceptPublicConnections(ln, req.PublicPort)
}

func (c *ClientSession) handleReqUnbind(payload json.RawMessage) {
	var req tunnel.ReqUnbindPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("Invalid REQ_UNBIND payload: %v", err)
		return
	}

	c.Mu.Lock()
	defer c.Mu.Unlock()

	ln, exists := c.Listeners[req.PublicPort]
	if !exists {
		if c.Debug {
			log.Printf("Unbind requested for non-existent port %d", req.PublicPort)
		}
		return
	}

	ln.Close()
	delete(c.Listeners, req.PublicPort)
	GlobalSessions.Unregister(req.PublicPort)
	if c.Debug {
		log.Printf("Unbound public port %d", req.PublicPort)
	}
}

func (c *ClientSession) handleReqDomainMap(payload json.RawMessage) {
	var req tunnel.ReqDomainMapPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("Invalid REQ_DOMAIN_MAP: %v", err)
		return
	}
	serverDomains.Add(req.Domain, req.PublicPort, req.Mode, req.AuthUser, req.AuthPass, req.RateLimit, req.SmartShield)
	if c.Debug {
		log.Printf("Mapped domain %s -> :%d (Mode: %s)", req.Domain, req.PublicPort, req.Mode)
	}
}

func (c *ClientSession) handleReqDomainUnmap(payload json.RawMessage) {
	var req tunnel.ReqDomainUnmapPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("Invalid REQ_DOMAIN_UNMAP: %v", err)
		return
	}
	serverDomains.Remove(req.Domain)
	if c.Debug {
		log.Printf("Unmapped domain %s", req.Domain)
	}
}

func (c *ClientSession) acceptPublicConnections(ln net.Listener, publicPort int) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		go c.proxyConnection(conn, publicPort)
	}
}

func (c *ClientSession) proxyConnection(userConn net.Conn, publicPort int) {

	stream, err := c.Session.Open()
	if err != nil {
		log.Printf("Failed to open stream to client: %v", err)
		userConn.Close()
		return
	}

	header := tunnel.ControlMessage{
		Type:    tunnel.MsgTypeNewConn,
		Payload: json.RawMessage(fmt.Sprintf(`{"public_port": %d}`, publicPort)),
	}

	if err := json.NewEncoder(stream).Encode(header); err != nil {
		log.Printf("Failed to send header: %v", err)
		stream.Close()
		userConn.Close()
		return
	}

	go func() {
		defer stream.Close()
		defer userConn.Close()
		io.Copy(stream, userConn)
	}()

	go func() {
		defer stream.Close()
		defer userConn.Close()
		io.Copy(userConn, stream)
	}()
}

func (c *ClientSession) Cleanup() {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	for port, ln := range c.Listeners {
		ln.Close()
		GlobalSessions.Unregister(port)
		if c.Debug {
			log.Printf("Closed listener on port %d", port)
		}
	}
	c.Conn.Close()
	c.Session.Close()
}
