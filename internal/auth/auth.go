package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	PasswordHash string `json:"password_hash"`
}

var (
	configPath = "data/auth_config.json"
	mu         sync.RWMutex
	cfg        Config

	sessions = make(map[string]time.Time)
	sessMu   sync.RWMutex
)

func Initialize() {
	if loadConfig() == nil && cfg.PasswordHash != "" {
		return
	}

}

func HasPassword() bool {
	return loadConfig() == nil && cfg.PasswordHash != ""
}

func SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	cfg.PasswordHash = string(hash)
	saveConfig()
	return nil
}

func VerifyPassword(password string) bool {
	mu.RLock()
	defer mu.RUnlock()
	err := bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(password))
	return err == nil
}

func CreateSession() string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	sessMu.Lock()
	sessions[token] = time.Now().Add(24 * time.Hour)
	sessMu.Unlock()

	return token
}

func ValidateSession(r *http.Request) bool {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return false
	}

	token := cookie.Value
	sessMu.RLock()
	expiry, exists := sessions[token]
	sessMu.RUnlock()

	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		sessMu.Lock()
		delete(sessions, token)
		sessMu.Unlock()
		return false
	}

	return true
}

func loadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &cfg)
}

func saveConfig() {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.MkdirAll("data", 0755)
	os.WriteFile(configPath, data, 0600)
}
