package nimsforestsprites

import (
	"math/rand"
	"sync"
	"time"
)

// State is the interface for renderable state (implemented by ViewModel)
type State interface {
	// Lands returns all land tiles to render
	Lands() []Land
	// Processes returns all active processes
	Processes() []Process
}

// Land represents a renderable land/node
type Land struct {
	ID   string
	Name string
	X, Y float64 // Grid position
	Type string  // "normal", "mana", etc.
}

// Process represents an active process (tree, nim, etc.)
type Process struct {
	ID       string
	LandID   string  // Which land this process is on
	Type     string  // "tree", "nim", "mana", etc.
	Progress float64 // 0.0 to 1.0
	X, Y     float64 // Position within the land
}

// MockState provides fake state for demo/testing
type MockState struct {
	mu        sync.RWMutex
	lands     []Land
	processes []Process
	rng       *rand.Rand
}

// NewMockState creates a new mock state with a grid of lands
func NewMockState() *MockState {
	m := &MockState{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	m.initializeLands()
	m.initializeProcesses()
	return m
}

func (m *MockState) initializeLands() {
	m.lands = make([]Land, 0)

	// Create a 5x5 grid of lands
	landTypes := []string{"normal", "mana", "normal", "normal", "mana"}

	id := 0
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			m.lands = append(m.lands, Land{
				ID:   generateID(id),
				Name: generateName(row, col),
				X:    float64(col),
				Y:    float64(row),
				Type: landTypes[m.rng.Intn(len(landTypes))],
			})
			id++
		}
	}
}

func (m *MockState) initializeProcesses() {
	m.processes = make([]Process, 0)

	processTypes := []string{"tree", "nim", "mana", "harvest"}

	// Add some initial processes on random lands
	for i := 0; i < 8; i++ {
		landIdx := m.rng.Intn(len(m.lands))
		land := m.lands[landIdx]

		m.processes = append(m.processes, Process{
			ID:       generateProcessID(i),
			LandID:   land.ID,
			Type:     processTypes[m.rng.Intn(len(processTypes))],
			Progress: m.rng.Float64(),
			X:        land.X + (m.rng.Float64()-0.5)*0.5,
			Y:        land.Y + (m.rng.Float64()-0.5)*0.5,
		})
	}
}

// Lands returns all lands
func (m *MockState) Lands() []Land {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Land, len(m.lands))
	copy(result, m.lands)
	return result
}

// Processes returns all processes
func (m *MockState) Processes() []Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Process, len(m.processes))
	copy(result, m.processes)
	return result
}

// Randomize updates the mock state with random changes
func (m *MockState) Randomize() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update process progress
	for i := range m.processes {
		m.processes[i].Progress += 0.1
		if m.processes[i].Progress > 1.0 {
			m.processes[i].Progress = 0.0
		}

		// Slightly move processes
		m.processes[i].X += (m.rng.Float64() - 0.5) * 0.05
		m.processes[i].Y += (m.rng.Float64() - 0.5) * 0.05
	}

	// Occasionally add or remove a process
	if m.rng.Float64() < 0.2 {
		if len(m.processes) > 3 && m.rng.Float64() < 0.5 {
			// Remove a random process
			idx := m.rng.Intn(len(m.processes))
			m.processes = append(m.processes[:idx], m.processes[idx+1:]...)
		} else if len(m.processes) < 15 {
			// Add a new process
			processTypes := []string{"tree", "nim", "mana", "harvest"}
			landIdx := m.rng.Intn(len(m.lands))
			land := m.lands[landIdx]

			m.processes = append(m.processes, Process{
				ID:       generateProcessID(m.rng.Int()),
				LandID:   land.ID,
				Type:     processTypes[m.rng.Intn(len(processTypes))],
				Progress: 0.0,
				X:        land.X + (m.rng.Float64()-0.5)*0.5,
				Y:        land.Y + (m.rng.Float64()-0.5)*0.5,
			})
		}
	}
}

// Helper functions for generating IDs and names
func generateID(n int) string {
	return string(rune('A'+n%26)) + string(rune('0'+n/26%10))
}

func generateName(row, col int) string {
	return string(rune('A'+row)) + string(rune('1'+col))
}

func generateProcessID(n int) string {
	return "P" + string(rune('0'+n%10)) + string(rune('A'+n/10%26))
}
