package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// ContainerState represents the state of a container according to OCI spec
type ContainerState struct {
	Version     string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	PID         int               `json:"pid,omitempty"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Valid container states
const (
	StateCreating = "creating"
	StateCreated  = "created"
	StateRunning  = "running"
	StateStopped  = "stopped"
)

// StateManager handles container state operations
type StateManager struct {
	RootDir string
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		RootDir: "/run/simcon",
	}
}

// CreateState creates a new container state
func (m *StateManager) CreateState(id, bundle string) (*ContainerState, error) {
	state := &ContainerState{
		Version:     specs.Version,
		ID:          id,
		Status:      StateCreating,
		Bundle:      bundle,
		Annotations: make(map[string]string),
	}

	if err := m.saveState(state); err != nil {
		return nil, err
	}

	return state, nil
}

// UpdateState updates the container state
func (m *StateManager) UpdateState(state *ContainerState) error {
	return m.saveState(state)
}

// GetState retrieves the container state
func (m *StateManager) GetState(id string) (*ContainerState, error) {
	statePath := filepath.Join(m.RootDir, id, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}

	var state ContainerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %v", err)
	}

	return &state, nil
}

// DeleteState removes the container state
func (m *StateManager) DeleteState(id string) error {
	stateDir := filepath.Join(m.RootDir, id)
	return os.RemoveAll(stateDir)
}

// saveState saves the container state to disk
func (m *StateManager) saveState(state *ContainerState) error {
	stateDir := filepath.Join(m.RootDir, state.ID)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %v", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %v", err)
	}

	return nil
}
