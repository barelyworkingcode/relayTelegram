package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SessionMapping struct {
	EveSessionID string    `json:"eveSessionId"`
	LastActive   time.Time `json:"lastActive"`
}

type ChatMapping struct {
	ProjectID   string                    `json:"projectId"`
	ProjectName string                    `json:"projectName"`
	Sessions    map[string]SessionMapping `json:"sessions"` // threadId -> session
}

type MappingsData struct {
	ChatMappings map[string]*ChatMapping `json:"chatMappings"` // chatId -> mapping
}

type Mappings struct {
	mu   sync.Mutex
	data MappingsData
	path string
}

func mappingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "relay", "telegram-mappings.json"), nil
}

func LoadMappings() (*Mappings, error) {
	p, err := mappingsPath()
	if err != nil {
		return nil, err
	}

	m := &Mappings{
		path: p,
		data: MappingsData{
			ChatMappings: make(map[string]*ChatMapping),
		},
	}

	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(raw, &m.data); err != nil {
		return nil, err
	}
	if m.data.ChatMappings == nil {
		m.data.ChatMappings = make(map[string]*ChatMapping)
	}
	return m, nil
}

func (m *Mappings) save() error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, raw, 0644)
}

func (m *Mappings) GetChatMapping(chatID string) *ChatMapping {
	m.mu.Lock()
	defer m.mu.Unlock()
	cm := m.data.ChatMappings[chatID]
	if cm == nil {
		return nil
	}
	// Return a copy so callers can read fields without holding the lock
	sessions := make(map[string]SessionMapping, len(cm.Sessions))
	for k, v := range cm.Sessions {
		sessions[k] = v
	}
	return &ChatMapping{
		ProjectID:   cm.ProjectID,
		ProjectName: cm.ProjectName,
		Sessions:    sessions,
	}
}

func (m *Mappings) LinkChat(chatID, projectID, projectName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data.ChatMappings[chatID] = &ChatMapping{
		ProjectID:   projectID,
		ProjectName: projectName,
		Sessions:    make(map[string]SessionMapping),
	}
	return m.save()
}

func (m *Mappings) UnlinkChat(chatID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data.ChatMappings, chatID)
	return m.save()
}

func (m *Mappings) GetSession(chatID, threadID string) *SessionMapping {
	m.mu.Lock()
	defer m.mu.Unlock()

	cm := m.data.ChatMappings[chatID]
	if cm == nil {
		return nil
	}
	s, ok := cm.Sessions[threadID]
	if !ok {
		return nil
	}
	return &s
}

func (m *Mappings) SetSession(chatID, threadID, eveSessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cm := m.data.ChatMappings[chatID]
	if cm == nil {
		return nil
	}
	cm.Sessions[threadID] = SessionMapping{
		EveSessionID: eveSessionID,
		LastActive:   time.Now(),
	}
	return m.save()
}

func (m *Mappings) ClearSession(chatID, threadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cm := m.data.ChatMappings[chatID]
	if cm == nil {
		return nil
	}
	delete(cm.Sessions, threadID)
	return m.save()
}
