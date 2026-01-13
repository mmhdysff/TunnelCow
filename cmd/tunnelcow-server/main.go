package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"tunnelcow/internal/tunnel"

	"github.com/hashicorp/yamux"
	"golang.org/x/crypto/acme/autocert"

	"math/rand"
	"time"
)

var Version = "dev"

func main() {
	fmt.Printf("TunnelCow Server %s\n", Version)
	portFlag := flag.Int("port", tunnel.DefaultControlPort, "Port to listen on for client connections")
	tokenFlag := flag.String("token", "", "Authentication token")
	debugFlag := flag.Bool("debug", false, "Enable verbose debug logging")
	flag.Parse()

	type ServerConfig struct {
		Token string `json:"token"`
		Port  int    `json:"port"`
		Debug bool   `json:"debug"`
	}
	var serverCfg ServerConfig
	configPath := "data/server_config.json"

	// Load Config
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &serverCfg)
	}

	// Determine Final Values
	finalToken := *tokenFlag
	if finalToken == "" {
		finalToken = serverCfg.Token
	}

	// Auto-Generate Token if missing
	if finalToken == "" {
		rand.Seed(time.Now().UnixNano())
		b := make([]byte, 16)
		rand.Read(b)
		finalToken = fmt.Sprintf("%x", b)
		fmt.Printf("Notice: No token provided. Generated one: %s\n", finalToken)

		// Save to config
		serverCfg.Token = finalToken
		serverCfg.Port = *portFlag
		serverCfg.Debug = *debugFlag

		bytes, _ := json.MarshalIndent(serverCfg, "", "  ")
		os.MkdirAll("data", 0755)
		os.WriteFile(configPath, bytes, 0644)
		fmt.Printf("Saved configuration to %s\n", configPath)
	}

	finalPort := *portFlag
	// If flag is default OR config has non-zero port, use config?
	// Usually flags override config.
	// But if flag is default (64290) and config is different (e.g. 7000), we should probably use config.
	// But we can't easily detect "flag provided" vs "default" easily with standard flags unless we use Visit.
	// Let's stick to: If flag is NOT default, use flag. Else use config if valid. Else default.
	if finalPort == tunnel.DefaultControlPort && serverCfg.Port != 0 {
		finalPort = serverCfg.Port
	}

	// Debug logic similar
	finalDebug := *debugFlag
	if !finalDebug && serverCfg.Debug {
		finalDebug = true
	}

	if finalDebug {
		log.Println("[DEBUG] Debug Mode: ENABLED")
	}

	addr := fmt.Sprintf(":%d", finalPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	log.Printf("TunnelCow Server listening on %s", addr)
	log.Printf("Auth Token: %s", finalToken)

	initDomainManager()

	go startHTTPSListener()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		if finalDebug {
			log.Printf("New connection from %s", conn.RemoteAddr())
		}
		go handleClient(conn, finalToken, finalPort, finalDebug)
	}
}

func startHTTPSListener() {

	m := &autocert.Manager{
		Cache:  autocert.DirCache("certs"),
		Prompt: autocert.AcceptTOS,
		HostPolicy: func(ctx context.Context, host string) error {

			if serverDomains.Exists(host) {
				return nil
			}
			return fmt.Errorf("domain %s not configured", host)
		},
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: m.TLSConfig(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}

			entry, ok := serverDomains.Get(host)
			if !ok {
				http.Error(w, "Domain not mapped", 404)
				return
			}

			if entry.Mode == "http" {
				http.Error(w, "HTTPS not enabled for this domain", 403)
				return
			}

			director := func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = fmt.Sprintf("127.0.0.1:%d", entry.PublicPort)
				req.Host = host
			}
			proxy := &httputil.ReverseProxy{Director: director}
			proxy.ServeHTTP(w, r)
		}),
	}

	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		entry, ok := serverDomains.Get(host)
		if !ok {
			http.Error(w, "Domain not mapped", 404)
			return
		}

		if entry.Mode == "http" {

			director := func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = fmt.Sprintf("127.0.0.1:%d", entry.PublicPort)
				req.Host = host
			}
			proxy := &httputil.ReverseProxy{Director: director}
			proxy.ServeHTTP(w, r)
		} else {

			target := "https://" + r.Host + r.URL.Path
			if len(r.URL.RawQuery) > 0 {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusTemporaryRedirect)
		}
	})

	go func() {
		log.Println("Starting HTTP-01 Listener on :80")

		wrappedHandler := m.HTTPHandler(httpHandler)

		if err := http.ListenAndServe(":80", wrappedHandler); err != nil {
			log.Printf("HTTP-01 Listener failed: %v", err)
		}
	}()

	log.Println("Starting TLS Server on :443...")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Printf("TLS Server failed: %v (Proceeding with Control Server only)", err)
	}
}

func handleClient(conn net.Conn, requiredToken string, controlPort int, debug bool) {
	buf := make([]byte, len(requiredToken))
	_, err := conn.Read(buf)
	if err != nil {
		log.Printf("Failed to read token: %v", err)
		conn.Close()
		return
	}

	if string(buf) != requiredToken {
		log.Printf("Invalid token from %s", conn.RemoteAddr())
		conn.Close()
		return
	}

	log.Printf("Client authenticated: %s", conn.RemoteAddr())

	session, err := yamux.Server(conn, nil)
	if err != nil {
		log.Printf("Yamux session init failed: %v", err)
		conn.Close()
		return
	}

	controlStream, err := session.Accept()
	if err != nil {
		log.Printf("Failed to accept control stream: %v", err)
		session.Close()
		return
	}

	log.Printf("Control stream established")

	client := NewClientSession(conn, session, controlStream, controlPort, debug)
	client.HandleControlLoop()
}
