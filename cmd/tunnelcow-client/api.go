package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	mux.Handle("/api/domains", authMiddleware(http.HandlerFunc(api.handleDomains)))

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

func (s *APIServer) handleDomains(w http.ResponseWriter, r *http.Request) {
	mgr := State.GetManager()
	if mgr == nil || !State.IsConnected() {
		http.Error(w, "Not connected to server", 503)
		return
	}

	switch r.Method {
	case "POST":
		var req struct {
			Domain     string `json:"domain"`
			PublicPort int    `json:"public_port"`
			Mode       string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := mgr.AddDomain(req.Domain, req.PublicPort, req.Mode); err != nil {
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
