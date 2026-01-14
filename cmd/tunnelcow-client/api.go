package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"tunnelcow/internal/auth"
	"tunnelcow/internal/tunnel"
)

type APIServer struct {
	Manager   *ClientManager
	StartTime time.Time
	Mu        sync.Mutex
}

func startAPIServer() {
	api := &APIServer{}

	mux := http.DefaultServeMux

	mux.HandleFunc("/api/login", api.handleLogin)
	mux.HandleFunc("/api/logout", api.handleLogout)

	mux.Handle("/api/status", authMiddleware(http.HandlerFunc(api.handleStatus)))
	mux.Handle("/api/tunnels", authMiddleware(http.HandlerFunc(api.handleTunnels)))
	mux.Handle("/api/tunnels/edit", authMiddleware(http.HandlerFunc(api.handleTunnelsEdit)))
	mux.Handle("/api/domains", authMiddleware(http.HandlerFunc(api.handleDomains)))
	mux.Handle("/api/inspect", authMiddleware(http.HandlerFunc(api.handleInspect)))
	mux.Handle("/api/replay", authMiddleware(http.HandlerFunc(api.handleReplay)))

	addr := fmt.Sprintf(":%d", tunnel.DefaultDashboardPort)
	log.Printf("Dashboard API listening on %s", addr)

	fs := getWebFileSystem()
	mux.Handle("/", http.FileServer(fs))

	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Printf("API Server failed: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		if !auth.ValidateSession(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Payload", 400)
		return
	}

	if auth.VerifyPassword(req.Password) {
		token := auth.CreateSession()
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   86400,
			SameSite: http.SameSiteLaxMode,
		})
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	} else {
		http.Error(w, "Invalid Password", 401)
	}
}

func (s *APIServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	mgr := State.GetManager()
	connected := State.IsConnected()

	tunnels := make(map[int]int)
	domains := make(map[string]interface{})
	if connected && mgr != nil {
		mgr.Mu.RLock()
		for k, v := range mgr.Tunnels {
			tunnels[k] = v
		}
		for k, v := range mgr.Domains {
			domains[k] = v
		}
		mgr.Mu.RUnlock()
	}

	status := map[string]interface{}{
		"connected":      connected,
		"server_addr":    State.ServerAddr,
		"dashboard_port": State.DashboardPort,
		"tunnels":        tunnels,
		"domains":        domains,
		"stats":          tunnel.GlobalStats,
		"uptime":         time.Since(State.StartTime).Seconds(),
	}
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) handleTunnels(w http.ResponseWriter, r *http.Request) {
	mgr := State.GetManager()
	if mgr == nil || !State.IsConnected() {
		http.Error(w, "Not connected to server", 503)
		return
	}

	switch r.Method {
	case "POST":
		var req struct {
			PublicPort string `json:"public_port"`
			LocalPort  string `json:"local_port"`
			Protocol   string `json:"protocol"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := mgr.AddRange(req.PublicPort, req.LocalPort); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case "DELETE":

		var req struct {
			PublicPort  int   `json:"public_port"`
			PublicPorts []int `json:"public_ports"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		count := 0
		if len(req.PublicPorts) > 0 {
			for _, p := range req.PublicPorts {
				mgr.RemoveTunnel(p)
				count++
			}
		} else if req.PublicPort > 0 {
			mgr.RemoveTunnel(req.PublicPort)
			count = 1
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted", "count": count})
	}
}

func (s *APIServer) handleTunnelsEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		PublicPort int `json:"public_port"`
		LocalPort  int `json:"local_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mgr := s.Manager
	if mgr == nil {
		http.Error(w, "Manager not initialized", 503)
		return
	}

	if err := mgr.EditTunnel(req.PublicPort, req.LocalPort); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) handleDomains(w http.ResponseWriter, r *http.Request) {
	mgr := State.GetManager()
	if mgr == nil || !State.IsConnected() {
		http.Error(w, "Not connected to server", 503)
		return
	}

	switch r.Method {
	case "POST":
		var req struct {
			Domain      string `json:"domain"`
			PublicPort  int    `json:"public_port"`
			Mode        string `json:"mode"`
			AuthUser    string `json:"auth_user"`
			AuthPass    string `json:"auth_pass"`
			RateLimit   int    `json:"rate_limit"`
			SmartShield bool   `json:"smart_shield"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := mgr.AddDomain(req.Domain, req.PublicPort, req.Mode, req.AuthUser, req.AuthPass, req.RateLimit, req.SmartShield); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case "DELETE":
		var req struct {
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := mgr.RemoveDomain(req.Domain); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

func (s *APIServer) handleInspect(w http.ResponseWriter, r *http.Request) {
	InspectLogsMu.RLock()
	defer InspectLogsMu.RUnlock()

	logs := InspectLogs[0]
	if logs == nil {
		logs = []tunnel.InspectPayload{}
	}

	json.NewEncoder(w).Encode(logs)
}

func (s *APIServer) handleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Payload", 400)
		return
	}

	InspectLogsMu.RLock()
	var logEntry *tunnel.InspectPayload
	if list, ok := InspectLogs[0]; ok {
		for _, l := range list {
			if l.ID == req.ID {
				entry := l
				logEntry = &entry
				break
			}
		}
	}
	InspectLogsMu.RUnlock()

	if logEntry == nil {
		http.Error(w, "Log not found", 404)
		return
	}

	mgr := State.GetManager()
	if mgr == nil {
		http.Error(w, "Manager not ready", 503)
		return
	}

	mgr.Mu.RLock()
	localPort, ok := mgr.Tunnels[logEntry.PublicPort]
	mgr.Mu.RUnlock()

	if !ok {

		http.Error(w, fmt.Sprintf("Tunnel for public port %d not found", logEntry.PublicPort), 404)
		return
	}

	targetURL := fmt.Sprintf("http://127.0.0.1:%d", localPort)

	u, err := http.NewRequest(logEntry.Method, logEntry.URL, nil)
	if err == nil {
		targetURL += u.URL.Path
		if u.URL.RawQuery != "" {
			targetURL += "?" + u.URL.RawQuery
		}
	} else {

		if len(logEntry.URL) > 0 && logEntry.URL[0] != '/' {
			targetURL += "/"
		}
		targetURL += logEntry.URL
	}

	var body io.Reader
	if logEntry.ReqBody != "" && logEntry.ReqBody != "[Request Body Too Large]" && logEntry.ReqBody != "[Binary Request Body]" {
		body = strings.NewReader(logEntry.ReqBody)
	}

	newReq, err := http.NewRequest(logEntry.Method, targetURL, body)
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), 500)
		return
	}

	for k, v := range logEntry.ReqHeaders {

		newReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(newReq)
	if err != nil {
		http.Error(w, "Replay failed: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      resp.Status,
		"status_code": resp.StatusCode,
		"replayed_to": targetURL,
	})
}
