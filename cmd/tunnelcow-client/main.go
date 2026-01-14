package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"tunnelcow/internal/auth"
	"tunnelcow/internal/tunnel"
	"tunnelcow/internal/ui"

	"github.com/hashicorp/yamux"
	"golang.org/x/term"
)

var Version = "dev"

func main() {
	ui.InitLogger(1000)
	fmt.Printf("TunnelCow Client %s\n", Version)
	// Config Load
	serverFlag := flag.String("server", "", "Server address")
	tokenFlag := flag.String("token", "", "Auth token")
	flag.Parse()

	type ClientConfig struct {
		ServerAddr string `json:"server_addr"`
		Token      string `json:"token"`
		Debug      bool   `json:"debug"`
	}
	var clientCfg ClientConfig

	// Load Config
	configPath := "data/client_config.json"
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &clientCfg)
	}

	// 1. Determine Server/Token
	finalServer := *serverFlag
	if finalServer == "" {
		finalServer = clientCfg.ServerAddr
	}
	finalToken := *tokenFlag
	if finalToken == "" {
		finalToken = clientCfg.Token
	}

	// 2. Setup Wizard if missing
	if finalServer == "" || finalToken == "" || !auth.HasPassword() {
		ui.ClearScreen()
		ui.DrawCenteredBox("Setup Required", []string{
			"Welcome to TunnelCow!",
			"",
			"Let's get you connected.",
			"",
			"Press [ENTER] to start setup...",
		})
		ui.ReadKey()

		// Step 1: Server
		if finalServer == "" {
			for {
				finalServer = ui.Input("Step 1/4: Server", "Enter Server Address (e.g. 1.2.3.4:64290):", false)
				if finalServer != "" {
					break
				}
			}
			clientCfg.ServerAddr = finalServer
		}

		// Step 2: Token
		if finalToken == "" {
			for {
				finalToken = ui.Input("Step 2/4: Authentication", "Enter Server Token:", true) // masked
				if finalToken != "" {
					break
				}
			}
			clientCfg.Token = finalToken
		}

		// Step 3: Password (if needed)
		if !auth.HasPassword() {
			ui.ClearScreen() // Clear previous step
			pass := ui.Input("Step 3/4: Security", "Create Password:", true)
			if pass == "" || len(pass) < 1 {
				log.Fatal("Password cannot be empty")
			}
			if err := auth.SetPassword(pass); err != nil {
				log.Fatalf("Failed to save password: %v", err)
			}
		}

		// Step 4: Debug
		ui.ClearScreen()
		debugChoice := ui.Select("Step 4/4: Settings", []string{
			"Enable Debug Mode",
			"Disable Debug Mode",
		})
		clientCfg.Debug = (debugChoice == 0)

		// Save Config
		bytes, _ := json.MarshalIndent(clientCfg, "", "  ")
		os.MkdirAll("data", 0755)
		os.WriteFile(configPath, bytes, 0644)

		ui.ClearScreen()
		ui.DrawCenteredBox("Setup Complete", []string{
			"",
			"✓ Configuration Saved",
			"",
			"Starting Client...",
		})
		time.Sleep(1 * time.Second)
	}

	config := &tunnel.Config{
		ServerAddr: finalServer,
		Token:      finalToken,
	}

	State.Mu.Lock()
	State.ServerAddr = finalServer
	State.DashboardPort = tunnel.DefaultDashboardPort
	State.Debug = clientCfg.Debug
	State.Mu.Unlock()

	go startAPIServer()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	if clientCfg.Debug {
		ui.Debug("Debug Mode: ENABLED")
		ui.Info("Press 'o' to toggle Debug Mode.")
	}

	redraw := make(chan bool, 1)

	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}

			key := buf[0]
			if key == 'o' || key == 'O' {
				State.Mu.Lock()
				State.Debug = !State.Debug
				newDebug := State.Debug
				State.Mu.Unlock()

				clientCfg.Debug = newDebug
				bytes, _ := json.MarshalIndent(clientCfg, "", "  ")
				os.WriteFile("data/client_config.json", bytes, 0644)

				redraw <- true
			}
			if key == 3 {
				term.Restore(int(os.Stdin.Fd()), oldState)
				os.Exit(0)
			}
		}
	}()

	go func() {

		redraw <- true

		for range redraw {
			State.Mu.RLock()
			debug := State.Debug
			State.Mu.RUnlock()

			fmt.Print("\033[H\033[2J")

			color := ui.Cyan
			status := "OFF"
			if debug {
				color = ui.Red
				status = "ON "
			}

			ui.DrawBoxWithColor(fmt.Sprintf("TunnelCow Client %s", Version), []string{
				fmt.Sprintf("Target Server: %s", State.ServerAddr),
				fmt.Sprintf("Dashboard:     http://localhost:%d", State.DashboardPort),
				fmt.Sprintf("Debug Mode:    %s%s%s (Press 'o' to toggle)", color, status, ui.Reset),
				"Press Ctrl+C to Exit",
			}, 60, color)

			fmt.Print("\r\nLogs:\r\n")

			lines := ui.Logger.GetLines(debug, 0)
			for _, line := range lines {

				cleanLine := strings.ReplaceAll(line, "\n", "\r\n")
				fmt.Print(cleanLine + "\r\n")
			}
		}
	}()

	for {
		State.Mu.RLock()
		debug := State.Debug
		State.Mu.RUnlock()

		if !debug {
			fmt.Printf("\r\033[K%s➜ Connecting to server...%s", ui.Yellow, ui.Reset)
		} else {
			ui.Debug("Connecting to server %s...", config.ServerAddr)
		}

		err := connectAndServe(config)

		if err != nil {
			State.Mu.RLock()
			debug = State.Debug
			State.Mu.RUnlock()

			if debug {
				ui.Debug("Connection error: %v", err)
				ui.Debug("Reconnecting in 5 seconds...")
			} else {
				fmt.Printf("\r\033[K%s✖ Disconnected. Reconnecting in 5s...%s", ui.Red, ui.Reset)
			}
			State.SetManager(nil)
		}

		time.Sleep(5 * time.Second)
	}

}

func connectAndServe(cfg *tunnel.Config) error {
	conn, err := net.DialTimeout("tcp", cfg.ServerAddr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cfg.Token))
	if err != nil {
		return err
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}

	control, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open control stream: %v", err)
	}

	ui.Info("Connected to server!")

	State.Mu.RLock()
	dbg := State.Debug
	State.Mu.RUnlock()
	manager := NewClientManager(control, session, dbg)
	State.SetManager(manager)

	go manager.RestoreTunnels()

	manager.ListenForStreams()
	return fmt.Errorf("session closed")
}
