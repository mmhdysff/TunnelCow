package main

import (
	"sync"
	"time"
)

type GlobalState struct {
	Manager       *ClientManager
	StartTime     time.Time
	ServerAddr    string
	DashboardPort int
	Debug         bool
	Mu            sync.RWMutex
}

var State = &GlobalState{
	StartTime: time.Now(),
}

func (g *GlobalState) SetManager(m *ClientManager) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	g.Manager = m
}

func (g *GlobalState) GetManager() *ClientManager {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	return g.Manager
}

func (g *GlobalState) IsConnected() bool {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	return g.Manager != nil && g.Manager.Session != nil && !g.Manager.Session.IsClosed()
}
