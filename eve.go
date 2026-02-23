package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type EveClient struct {
	baseURL string
	client  *http.Client
}

type CreateSessionRequest struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name,omitempty"`
}

type CreateSessionResponse struct {
	SessionID string `json:"sessionId"`
	ProjectID string `json:"projectId"`
	Model     string `json:"model"`
}

type SendMessageRequest struct {
	Text string `json:"text"`
}

type SendMessageResponse struct {
	Response string          `json:"response"`
	Stats    json.RawMessage `json:"stats"`
	Error    string          `json:"error,omitempty"`
}

type EveProject struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Model    string `json:"model"`
	Disabled bool   `json:"disabled"`
}

func NewEveClient(baseURL string) *EveClient {
	return &EveClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 6 * time.Minute, // Longer than Eve's 5min timeout
		},
	}
}

func (e *EveClient) ListProjects() ([]EveProject, error) {
	resp, err := e.client.Get(e.baseURL + "/api/projects")
	if err != nil {
		return nil, fmt.Errorf("failed to reach Eve: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Eve returned %d: %s", resp.StatusCode, body)
	}

	var projects []EveProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}
	return projects, nil
}

func (e *EveClient) CreateSession(projectID, name string) (*CreateSessionResponse, error) {
	body, _ := json.Marshal(CreateSessionRequest{ProjectID: projectID, Name: name})
	resp, err := e.client.Post(e.baseURL+"/api/sessions", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to reach Eve: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Eve returned %d: %s", resp.StatusCode, respBody)
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

func (e *EveClient) SendMessage(sessionID, text string) (*SendMessageResponse, error) {
	body, _ := json.Marshal(SendMessageRequest{Text: text})
	resp, err := e.client.Post(
		e.baseURL+"/api/sessions/"+url.PathEscape(sessionID)+"/message",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Eve: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 409 {
		return nil, fmt.Errorf("session is busy processing another message")
	}
	if resp.StatusCode == 504 {
		return nil, fmt.Errorf("response timed out")
	}

	var result SendMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != 200 {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("Eve returned %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	return &result, nil
}
