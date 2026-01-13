package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"tunnelcow/internal/ui"

	"golang.org/x/crypto/bcrypt"
)

type ClientConfig struct {
	Debug bool `json:"debug"`
}

func main() {
	ui.ClearScreen()

	ui.Select("TunnelCow Setup", []string{
		"Welcome to the TunnelCow Configuration Wizard.",
		"Press ENTER to continue...",
	})

	ui.ClearScreen()

	action := ui.Select("Step 1: Security", []string{
		"Set/Change Password",
		"Skip",
	})

	if action == 0 {
		password := ui.Input("Security Setup", "Enter new dashboard password:", true)
		if len(password) > 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			content := fmt.Sprintf(`{"password_hash": "%s"}`, hash)
			os.MkdirAll("data", 0755)
			os.WriteFile("data/auth_config.json", []byte(content), 0644)

			ui.ClearScreen()
			ui.DrawCenteredBox("Success", []string{
				"",
				"✓ Password Updated Securely",
				"",
			})
			time.Sleep(1 * time.Second)
		}
	}

	config := ClientConfig{Debug: false}
	if data, err := os.ReadFile("data/client_config.json"); err == nil {
		json.Unmarshal(data, &config)
	}

	currentStatus := "DISABLED"
	if config.Debug {
		currentStatus = "ENABLED"
	}

	ui.ClearScreen()

	debugAction := ui.Select(fmt.Sprintf("Debug Mode (Current: %s)", currentStatus), []string{
		"Enable",
		"Disable",
		"Keep Unchanged",
	})

	switch debugAction {
	case 0:
		config.Debug = true
	case 1:
		config.Debug = false
	}

	bytes, _ := json.MarshalIndent(config, "", "  ")
	os.MkdirAll("data", 0755)
	os.WriteFile("data/client_config.json", bytes, 0644)

	ui.ClearScreen()
	ui.Select("Setup Complete", []string{
		"Configuration saved.",
		"Ready to launch tunnelcow-client.exe",
		"",
		"[ Exit ]",
	})
}
