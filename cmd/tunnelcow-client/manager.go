package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"tunnelcow/internal/tunnel"

	"github.com/hashicorp/yamux"
)

type TunnelConfig struct {
	LocalPort  int
	PublicPort int
}

type ClientDomainEntry struct {
	PublicPort int    `json:"public_port"`
	Mode       string `json:"mode"`
}

type ClientManager struct {
	Control net.Conn
	Session *yamux.Session
	Tunnels map[int]int
	Domains map[string]ClientDomainEntry
	Mu      sync.RWMutex
	Debug   bool
}

func NewClientManager(control net.Conn, session *yamux.Session, debug bool) *ClientManager {
	return &ClientManager{
		Control: control,
		Session: session,
		Tunnels: make(map[int]int),
		Domains: make(map[string]ClientDomainEntry),
		Debug:   debug,
	}
}

func (m *ClientManager) AddDomain(domain string, publicPort int, mode string) error {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	if _, exists := m.Tunnels[publicPort]; !exists {
		return fmt.Errorf("public port %d is not active", publicPort)
	}

	req := tunnel.ReqDomainMapPayload{
		Domain:     domain,
		PublicPort: publicPort,
		Mode:       mode,
	}

	msg := tunnel.ControlMessage{
		Type:    tunnel.MsgTypeReqDomainMap,
		Payload: mustMarshal(req),
	}

	if err := json.NewEncoder(m.Control).Encode(msg); err != nil {
		return err
	}

	m.Domains[domain] = ClientDomainEntry{PublicPort: publicPort, Mode: mode}
	m.saveDomains()
	log.Printf("Mapped domain %s -> :%d (Mode: %s)", domain, publicPort, mode)
	return nil
}

func (m *ClientManager) RemoveDomain(domain string) error {
	req := tunnel.ReqDomainUnmapPayload{
		Domain: domain,
	}

	msg := tunnel.ControlMessage{
		Type:    tunnel.MsgTypeReqDomainUnmap,
		Payload: mustMarshal(req),
	}

	if err := json.NewEncoder(m.Control).Encode(msg); err != nil {
		return err
	}

	delete(m.Domains, domain)
	m.saveDomains()
	log.Printf("Unmapped domain %s", domain)
	return nil
}

func (m *ClientManager) saveDomains() {
	type savedDomain struct {
		Domain string `json:"domain"`
		Port   int    `json:"port"`
		Mode   string `json:"mode"`
	}
	var list = []savedDomain{}
	for d, e := range m.Domains {
		list = append(list, savedDomain{Domain: d, Port: e.PublicPort, Mode: e.Mode})
	}
	file, _ := json.MarshalIndent(list, "", "  ")
	os.MkdirAll("data", 0755)
	_ = os.WriteFile("data/client_domains.json", file, 0644)
}

func (m *ClientManager) restoreDomains() {
	data, err := os.ReadFile("data/client_domains.json")
	if err != nil {
		return
	}
	type savedDomain struct {
		Domain string `json:"domain"`
		Port   int    `json:"port"`
		Mode   string `json:"mode"`
	}
	var list []savedDomain
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}
	log.Printf("Restoring %d domains...", len(list))
	for _, d := range list {
		if d.Mode == "" {
			d.Mode = "auto"
		}
		if err := m.AddDomain(d.Domain, d.Port, d.Mode); err != nil {
			log.Printf("Failed to restore domain %s: %v", d.Domain, err)
		}
	}
}

func (m *ClientManager) AddTunnel(publicPort, localPort int) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	State.Mu.RLock()
	dashPort := State.DashboardPort
	serverAddr := State.ServerAddr
	State.Mu.RUnlock()

	if localPort == dashPort || publicPort == dashPort {
		return fmt.Errorf("cannot use dashboard port %d for tunneling", dashPort)
	}

	if parts := strings.Split(serverAddr, ":"); len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			if publicPort == p || localPort == p {
				return fmt.Errorf("port %d conflicts with server control port", p)
			}
		}
	}

	if tcpAddr, ok := m.Control.LocalAddr().(*net.TCPAddr); ok {
		if localPort == tcpAddr.Port || publicPort == tcpAddr.Port {
			return fmt.Errorf("port %d is the active link to server (ephemeral). dangerous to tunnel.", tcpAddr.Port)
		}
	}

	if _, exists := m.Tunnels[publicPort]; exists {
		return fmt.Errorf("public port %d is already active. delete it first.", publicPort)
	}

	req := tunnel.ReqBindPayload{
		PublicPort: publicPort,
		LocalPort:  localPort,
	}

	msg := tunnel.ControlMessage{
		Type:    tunnel.MsgTypeReqBind,
		Payload: mustMarshal(req),
	}

	if err := json.NewEncoder(m.Control).Encode(msg); err != nil {
		return err
	}

	m.Tunnels[publicPort] = localPort
	m.saveTunnels()
	if State.Debug {
		log.Printf("Requested tunnel: Local :%d <-> Public :%d", localPort, publicPort)
	}
	return nil
}

func (m *ClientManager) AddRange(publicStr, localStr string) error {
	if strings.Contains(publicStr, "-") {
		pParts := strings.Split(publicStr, "-")
		lParts := strings.Split(localStr, "-")

		if len(pParts) != 2 || len(lParts) != 2 {
			return fmt.Errorf("invalid range format")
		}

		pStart, _ := strconv.Atoi(strings.TrimSpace(pParts[0]))
		pEnd, _ := strconv.Atoi(strings.TrimSpace(pParts[1]))
		lStart, _ := strconv.Atoi(strings.TrimSpace(lParts[0]))

		if pEnd < pStart {
			return fmt.Errorf("invalid range order")
		}

		count := pEnd - pStart
		for i := 0; i <= count; i++ {
			p := pStart + i
			l := lStart + i
			if err := m.AddTunnel(p, l); err != nil {
				log.Printf("Failed to add range item %d->%d: %v", p, l, err)
			}
		}
		return nil
	}

	p, err := strconv.Atoi(publicStr)
	if err != nil {
		return err
	}
	l, err := strconv.Atoi(localStr)
	if err != nil {
		return err
	}

	return m.AddTunnel(p, l)
}

func (m *ClientManager) removeTunnelInternal(publicPort int, save bool) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	req := tunnel.ReqUnbindPayload{
		PublicPort: publicPort,
	}
	msg := tunnel.ControlMessage{
		Type:    tunnel.MsgTypeReqUnbind,
		Payload: mustMarshal(req),
	}
	if err := json.NewEncoder(m.Control).Encode(msg); err != nil {
		log.Printf("Failed to send UNBIND for %d: %v", publicPort, err)
	}

	var orphanedDomains []string
	for d, e := range m.Domains {
		if e.PublicPort == publicPort {
			orphanedDomains = append(orphanedDomains, d)
		}
	}
	for _, d := range orphanedDomains {
		if err := m.RemoveDomain(d); err != nil {
			log.Printf("Failed to unmap orphan domain %s: %v", d, err)
		} else {
			log.Printf("Removed orphan domain %s linked to port %d", d, publicPort)
		}
	}

	delete(m.Tunnels, publicPort)
	if save {
		m.saveTunnels()
	}
	State.Mu.RLock()
	debug := State.Debug
	State.Mu.RUnlock()
	if debug {
		log.Printf("Removed tunnel for public port %d", publicPort)
	}
}

func (m *ClientManager) RemoveTunnel(publicPort int) {
	m.removeTunnelInternal(publicPort, true)
}

func (m *ClientManager) saveTunnels() {
	type savedTunnel struct {
		Public int `json:"public"`
		Local  int `json:"local"`
	}
	var list = []savedTunnel{}
	for p, l := range m.Tunnels {
		list = append(list, savedTunnel{Public: p, Local: l})
	}

	file, _ := json.MarshalIndent(list, "", "  ")
	os.MkdirAll("data", 0755)
	_ = os.WriteFile("data/tunnels.json", file, 0644)
}

func (m *ClientManager) RestoreTunnels() {
	data, err := os.ReadFile("data/tunnels.json")
	if err != nil {
		return
	}

	type savedTunnel struct {
		Public int `json:"public"`
		Local  int `json:"local"`
	}
	var list []savedTunnel
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}

	log.Printf("Restoring %d tunnels...", len(list))
	for _, t := range list {
		if err := m.AddTunnel(t.Public, t.Local); err != nil {
			log.Printf("Failed to restore :%d->:%d : %v", t.Local, t.Public, err)
		}
	}
	m.restoreDomains()
}

func (m *ClientManager) ListenForStreams() {

	go m.readControlLoop()

	go m.startPingLoop()

	for {
		stream, err := m.Session.Accept()
		if err != nil {
			log.Printf("Session accept error: %v", err)
			return
		}
		go m.handleStream(stream)
	}
}

func (m *ClientManager) startPingLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UnixNano()
		payload, _ := json.Marshal(map[string]int64{"ts": now})
		msg := tunnel.ControlMessage{
			Type:    tunnel.MsgTypePing,
			Payload: payload,
		}

		m.Mu.Lock()
		err := json.NewEncoder(m.Control).Encode(msg)
		m.Mu.Unlock()

		if err != nil {
			return
		}
	}
}

func (m *ClientManager) readControlLoop() {
	decoder := json.NewDecoder(m.Control)
	for {
		var msg tunnel.ControlMessage
		err := decoder.Decode(&msg)
		if err != nil {
			return
		}

		switch msg.Type {
		case tunnel.MsgTypePing:
			var payload map[string]int64
			if err := json.Unmarshal(msg.Payload, &payload); err == nil {
				sent := payload["ts"]
				rtt := time.Now().UnixNano() - sent

				ms := rtt / 1_000_000
				atomic.StoreInt64(&tunnel.GlobalStats.LatencyMs, ms)
			}
		case tunnel.MsgTypeInspectData:
			m.handleInspectData(msg.Payload)
		}
	}
}

func (m *ClientManager) handleStream(stream net.Conn) {

	bufferedStream := bufio.NewReader(stream)

	headerBytes, err := bufferedStream.ReadBytes('\n')
	if err != nil {
		stream.Close()
		return
	}

	var header tunnel.ControlMessage
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		log.Printf("Invalid header: %v", err)
		stream.Close()
		return
	}

	var payload struct {
		PublicPort int `json:"public_port"`
	}
	if err := json.Unmarshal(header.Payload, &payload); err != nil {
		stream.Close()
		return
	}

	m.Mu.RLock()
	localPort, ok := m.Tunnels[payload.PublicPort]
	m.Mu.RUnlock()

	if !ok {
		log.Printf("Unknown tunnel for public port %d", payload.PublicPort)
		stream.Close()
		return
	}

	localConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		log.Printf("Failed to dial local service on port %d: %v", localPort, err)
		stream.Close()
		return
	}

	go func() {
		defer stream.Close()
		defer localConn.Close()

		reader := &tunnel.MonitoredReader{R: localConn, Counter: &tunnel.GlobalStats.BytesUp}
		io.Copy(stream, reader)
	}()

	go func() {
		defer stream.Close()
		defer localConn.Close()

		reader := &tunnel.MonitoredReader{R: bufferedStream, Counter: &tunnel.GlobalStats.BytesDown}
		io.Copy(localConn, reader)
	}()
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

var (
	InspectLogs   = make(map[int][]tunnel.InspectPayload)
	InspectLogsMu sync.RWMutex
)

func (c *ClientManager) handleInspectData(payload json.RawMessage) {
	var data tunnel.InspectPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Invalid INSPECT_DATA: %v", err)
		return
	}

	if c.Debug {
		log.Printf("[INSPECT] Received data for URL: %s", data.URL)
	}

	InspectLogsMu.Lock()
	defer InspectLogsMu.Unlock()

	list := InspectLogs[0]
	list = append(list, data)
	if len(list) > 100 {
		list = list[len(list)-100:]
	}
	InspectLogs[0] = list
}
