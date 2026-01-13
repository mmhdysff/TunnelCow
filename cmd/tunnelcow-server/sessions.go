package main

import (
	"sync"
)

type SessionManager struct {
	Mu       sync.RWMutex
	Sessions map[int]*ClientSession
}

var GlobalSessions = &SessionManager{
	Sessions: make(map[int]*ClientSession),
}

func (sm *SessionManager) Register(publicPort int, session *ClientSession) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	sm.Sessions[publicPort] = session
}

func (sm *SessionManager) Unregister(publicPort int) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	delete(sm.Sessions, publicPort)
}

func (sm *SessionManager) Get(publicPort int) (*ClientSession, bool) {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()
	sess, ok := sm.Sessions[publicPort]
	return sess, ok
}
