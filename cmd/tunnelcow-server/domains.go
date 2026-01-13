package main

import (
	"encoding/json"
	"os"
	"sync"
)

type DomainEntry struct {
	PublicPort int    `json:"public_port"`
	Mode       string `json:"mode"`
}

type DomainManager struct {
	File string
	Mu   sync.RWMutex

	Domains map[string]DomainEntry
}

var serverDomains *DomainManager

func initDomainManager() {
	serverDomains = &DomainManager{
		File:    "data/domains.json",
		Domains: make(map[string]DomainEntry),
	}

	os.MkdirAll("data", 0755)
	serverDomains.load()
}

func (dm *DomainManager) load() {
	data, err := os.ReadFile(dm.File)
	if err == nil {
		json.Unmarshal(data, &dm.Domains)
	}
}

func (dm *DomainManager) save() {
	data, _ := json.MarshalIndent(dm.Domains, "", "  ")
	os.MkdirAll("data", 0755)
	os.WriteFile(dm.File, data, 0644)
}

func (dm *DomainManager) Add(domain string, port int, mode string) {
	dm.Mu.Lock()
	defer dm.Mu.Unlock()
	if mode == "" {
		mode = "auto"
	}
	dm.Domains[domain] = DomainEntry{PublicPort: port, Mode: mode}
	dm.save()
}

func (dm *DomainManager) Remove(domain string) {
	dm.Mu.Lock()
	defer dm.Mu.Unlock()
	delete(dm.Domains, domain)
	dm.save()
}

func (dm *DomainManager) Get(domain string) (DomainEntry, bool) {
	dm.Mu.RLock()
	defer dm.Mu.RUnlock()
	e, ok := dm.Domains[domain]
	return e, ok
}

func (dm *DomainManager) Exists(domain string) bool {
	dm.Mu.RLock()
	defer dm.Mu.RUnlock()
	_, ok := dm.Domains[domain]
	return ok
}

func (dm *DomainManager) GetPort(domain string) (int, bool) {
	dm.Mu.RLock()
	defer dm.Mu.RUnlock()
	e, ok := dm.Domains[domain]
	return e.PublicPort, ok
}
