package ui

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"pulse/internal/docker"
)

// Model represents the application state
type Model struct {
	stacks        []string
	selectedStack int
	cli           *client.Client
	state         string
	logOutput     string
	containers    []types.Container
	debug         bool

	// New fields for enhanced information
	stackStats     map[string]StackStats
	viewportWidth  int
	viewportHeight int
	activeServices int
	totalServices  int

	// Add selected container tracking
	selectedContainer int
}

// StackStats holds statistics for a stack
type StackStats struct {
	Running     int
	Stopped     int
	Other       int
	TotalMemory string
	TotalCPU    string
}

// NewModel creates and initializes a new model
func NewModel(cli *client.Client, debug bool) Model {
	stacks, err := docker.ListStacks(context.Background(), cli)
	if err != nil {
		log.Fatalf("Error listing stacks: %v", err)
	}

	// Get initial stack statistics
	stackStats := make(map[string]StackStats)
	var activeServices, totalServices int

	for _, stack := range stacks {
		containers, err := docker.ListContainers(context.Background(), cli, stack)
		if err != nil {
			log.Printf("Error getting containers for stack %s: %v", stack, err)
			continue
		}

		stats := StackStats{}
		for _, c := range containers {
			totalServices++
			switch c.State {
			case "running":
				stats.Running++
				activeServices++
			case "exited", "stopped":
				stats.Stopped++
			default:
				stats.Other++
			}
		}
		stackStats[stack] = stats
	}

	return Model{
		stacks:            stacks,
		selectedStack:     0,
		cli:               cli,
		state:             "stack",
		debug:             debug,
		stackStats:        stackStats,
		activeServices:    activeServices,
		totalServices:     totalServices,
		viewportWidth:     100, // Default, will be updated
		viewportHeight:    30,  // Default, will be updated
		selectedContainer: 0,   // Initialize selected container
	}
}

// Update handles UI state updates based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if m.state == "stack" {
				m.state = "containerList"
				m.selectedContainer = 0 // Reset selected container when entering container list
				fmt.Println("len(m.stacks)", len(m.stacks))
				if len(m.stacks) > 0 {
					containers, err := docker.ListContainers(context.Background(), m.cli, m.stacks[m.selectedStack])
					if err != nil {
						m.logOutput = fmt.Sprintf("Error listing containers: %v", err)
					} else {
						m.containers = containers
					}
				}
			} else if m.state == "containerList" && len(m.containers) > 0 {
				// View logs for the selected container
				m.state = "containerLogs"
				logs, err := docker.ViewContainerLogs(context.Background(), m.cli, m.containers[m.selectedContainer].ID)
				if err != nil {
					m.logOutput = fmt.Sprintf("Error retrieving container logs: %v", err)
					m.state = "containerList" // Return to container list on error
				} else {
					m.logOutput = logs
				}
			}
		case "a":
			if m.state == "stack" {
				m.state = "actionMenu"
			}
		case "up":
			if m.state == "stack" && m.selectedStack > 0 {
				m.selectedStack--
			} else if m.state == "containerList" && m.selectedContainer > 0 {
				m.selectedContainer--
			}
		case "down":
			if m.state == "stack" && m.selectedStack < len(m.stacks)-1 {
				m.selectedStack++
			} else if m.state == "containerList" && m.selectedContainer < len(m.containers)-1 {
				m.selectedContainer++
			}
		case "r":
			if m.state == "actionMenu" {
				selectedStack := m.stacks[m.selectedStack]
				err := docker.RestartStack(context.Background(), m.cli, selectedStack)
				if err != nil {
					m.logOutput = fmt.Sprintf("Error restarting stack: %v", err)
				} else {
					m.logOutput = fmt.Sprintf("Stack %s restarted successfully", selectedStack)
				}
				m.state = "stack"
			}
		case "k":
			if m.state == "actionMenu" {
				selectedStack := m.stacks[m.selectedStack]
				err := docker.KillStack(context.Background(), m.cli, selectedStack)
				if err != nil {
					m.logOutput = fmt.Sprintf("Error killing stack: %v", err)
				} else {
					m.logOutput = fmt.Sprintf("Stack %s killed successfully", selectedStack)
				}
				m.state = "stack"

				// Update stats after kill operation
				stacks, _ := docker.ListStacks(context.Background(), m.cli)
				m.stacks = stacks
				m.updateStackStats()
			}
		case "l":
			if m.state == "actionMenu" {
				selectedStack := m.stacks[m.selectedStack]
				logs, err := docker.ViewStackLogs(context.Background(), m.cli, selectedStack)
				if err != nil {
					m.logOutput = fmt.Sprintf("Error retrieving logs: %v", err)
				} else {
					m.logOutput = logs
				}
				m.state = "stack"
			}
		case "escape", "backspace", "b":
			// Multiple keys for going back for better UX
			switch m.state {
			case "containerLogs":
				m.state = "containerList"
				m.logOutput = "" // Clear log output when going back
			case "containerList":
				m.state = "stack"
				// Refresh stack stats when returning to stack view
				m.updateStackStats()
			case "actionMenu":
				m.state = "stack"
			}
		}
	case tea.WindowSizeMsg:
		// Save window dimensions for responsive layout
		m.viewportWidth = msg.Width
		m.viewportHeight = msg.Height

		// Update header width to match viewport width
		headerStyle = headerStyle.Width(msg.Width)
	}
	return m, nil
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Helper method to update stack statistics
func (m *Model) updateStackStats() {
	m.stackStats = make(map[string]StackStats)
	m.activeServices = 0
	m.totalServices = 0

	for _, stack := range m.stacks {
		containers, err := docker.ListContainers(context.Background(), m.cli, stack)
		if err != nil {
			continue
		}

		stats := StackStats{}
		for _, c := range containers {
			m.totalServices++
			switch c.State {
			case "running":
				stats.Running++
				m.activeServices++
			case "exited", "stopped":
				stats.Stopped++
			default:
				stats.Other++
			}
		}
		m.stackStats[stack] = stats
	}
}
