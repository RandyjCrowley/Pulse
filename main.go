package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type model struct {
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

type StackStats struct {
	Running     int
	Stopped     int
	Other       int
	TotalMemory string
	TotalCPU    string
}

func initialModel(cli *client.Client, debug bool) model {
	stacks, err := listStacks(context.Background(), cli)
	if err != nil {
		log.Fatalf("Error listing stacks: %v", err)
	}

	// Get initial stack statistics
	stackStats := make(map[string]StackStats)
	var activeServices, totalServices int

	for _, stack := range stacks {
		containers, err := listContainers(context.Background(), cli, stack)
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

	return model{
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

func listStacks(ctx context.Context, cli *client.Client) ([]string, error) {
	services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	stackMap := make(map[string]bool)
	for _, service := range services {
		if stackName, ok := service.Spec.Labels["com.docker.stack.namespace"]; ok {
			stackMap[stackName] = true
		}
	}

	stacks := make([]string, 0, len(stackMap))
	for stackName := range stackMap {
		stacks = append(stacks, stackName)
	}

	return stacks, nil
}

func listContainers(ctx context.Context, cli *client.Client, stackName string) ([]types.Container, error) {
	containerFilter := filters.NewArgs()
	containerFilter.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: containerFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing containers for stack %s: %v", stackName, err)
	}
	return containers, nil
}

func killStack(ctx context.Context, cli *client.Client, stackName string) error {
	serviceFilter := filters.NewArgs()
	serviceFilter.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))

	services, err := cli.ServiceList(ctx, types.ServiceListOptions{
		Filters: serviceFilter,
	})
	if err != nil {
		return fmt.Errorf("error listing services for stack %s: %v", stackName, err)
	}

	for _, service := range services {
		if err := cli.ServiceRemove(ctx, service.ID); err != nil {
			return fmt.Errorf("error removing service %s: %v", service.Spec.Name, err)
		}
	}

	return nil
}

func restartStack(ctx context.Context, cli *client.Client, stackName string) error {
	if err := killStack(ctx, cli, stackName); err != nil {
		return fmt.Errorf("error killing stack: %v", err)
	}

	return fmt.Errorf("full stack restart requires external deployment mechanism")
}

func viewStackLogs(ctx context.Context, cli *client.Client, stackName string) (string, error) {
	serviceFilter := filters.NewArgs()
	serviceFilter.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))

	services, err := cli.ServiceList(ctx, types.ServiceListOptions{
		Filters: serviceFilter,
	})
	if err != nil {
		return "", fmt.Errorf("error listing services for stack %s: %v", stackName, err)
	}

	var logBuilder strings.Builder

	for _, service := range services {
		logs, err := cli.ServiceLogs(ctx, service.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "50",
		})
		if err != nil {
			logBuilder.WriteString(fmt.Sprintf("Error getting logs for service %s: %v\n", service.Spec.Name, err))
			continue
		}
		defer func(logs io.ReadCloser) {
			_ = logs.Close()
		}(logs)

		logBuilder.WriteString(fmt.Sprintf("Logs for service: %s\n", service.Spec.Name))
		logBytes, err := io.ReadAll(logs)
		if err != nil {
			logBuilder.WriteString(fmt.Sprintf("Error reading logs: %v\n", err))
		} else {
			logBuilder.Write(logBytes)
		}
		logBuilder.WriteString("\n---\n")
	}

	return logBuilder.String(), nil
}

// Add function to view logs from a specific container
func viewContainerLogs(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	logs, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
		Timestamps: true,
	})
	if err != nil {
		return "", fmt.Errorf("error getting logs for container %s: %v", containerID, err)
	}
	defer logs.Close()

	// Read container logs
	logBytes, err := io.ReadAll(logs)
	if err != nil {
		return "", fmt.Errorf("error reading container logs: %v", err)
	}

	return string(logBytes), nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

					containers, err := listContainers(context.Background(), m.cli, m.stacks[m.selectedStack])
					if err != nil {
						m.logOutput = fmt.Sprintf("Error listing containers: %v", err)
					} else {
						m.containers = containers
					}
				}
			} else if m.state == "containerList" && len(m.containers) > 0 {
				// View logs for the selected container
				m.state = "containerLogs"
				logs, err := viewContainerLogs(context.Background(), m.cli, m.containers[m.selectedContainer].ID)
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
				err := restartStack(context.Background(), m.cli, selectedStack)
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
				err := killStack(context.Background(), m.cli, selectedStack)
				if err != nil {
					m.logOutput = fmt.Sprintf("Error killing stack: %v", err)
				} else {
					m.logOutput = fmt.Sprintf("Stack %s killed successfully", selectedStack)
				}
				m.state = "stack"

				// Update stats after kill operation
				stacks, _ := listStacks(context.Background(), m.cli)
				m.stacks = stacks
				m.updateStackStats()
			}
		case "l":
			if m.state == "actionMenu" {
				selectedStack := m.stacks[m.selectedStack]
				logs, err := viewStackLogs(context.Background(), m.cli, selectedStack)
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

// Helper method to update stack statistics
func (m *model) updateStackStats() {
	m.stackStats = make(map[string]StackStats)
	m.activeServices = 0
	m.totalServices = 0

	for _, stack := range m.stacks {
		containers, err := listContainers(context.Background(), m.cli, stack)
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

var (
	// Vibrant color palette
	colorPrimary    = lipgloss.Color("#FF5F87") // Vibrant pink
	colorSecondary  = lipgloss.Color("#5FAFFF") // Bright blue
	colorAccent     = lipgloss.Color("#FFAF00") // Bold orange
	colorSuccess    = lipgloss.Color("#50FA7B") // Neon green
	colorDanger     = lipgloss.Color("#FF5555") // Bright red
	colorWarning    = lipgloss.Color("#F1FA8C") // Vibrant yellow
	colorBackground = lipgloss.Color("#282A36") // Dark background
	colorText       = lipgloss.Color("#F8F8F2") // Light text
	colorSubtext    = lipgloss.Color("#BFBFBF") // Grey text
	colorHighlight  = lipgloss.Color("#BD93F9") // Purple highlight

	// Updated styles with more vibrant colors
	titleStyle       = lipgloss.NewStyle().Foreground(colorHighlight).Bold(true).Padding(1, 2)
	selectedStyle    = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).PaddingLeft(2)
	unselectedStyle  = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	logStyle         = lipgloss.NewStyle().Padding(1, 2).Background(colorBackground).Foreground(colorText)
	instructionStyle = lipgloss.NewStyle().Foreground(colorSubtext).Padding(1, 2)
	debugStyle       = lipgloss.NewStyle().Foreground(colorDanger)

	// Redesigned UI components with vibrant borders and backgrounds
	headerStyle     = lipgloss.NewStyle().Foreground(colorText).Background(colorPrimary).Bold(true).Padding(0, 1).Width(100)
	stackPanelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSecondary).Padding(1, 2).Background(colorBackground)
	containerStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSuccess).Padding(1, 2).Background(colorBackground)
	logPanelStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Padding(1, 2).Background(colorBackground)
	helpPanelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorHighlight).Padding(1, 2).Background(colorBackground)
	actionMenuStyle = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(colorPrimary).Background(colorBackground).Foreground(colorText)

	// Status indicators
	statusRunning = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	statusStopped = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	statusOther   = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
)

func (m model) View() string {
	// Set dynamic widths based on viewport
	stackWidth := m.viewportWidth * 2 / 3
	helpWidth := m.viewportWidth - stackWidth - 4 // Account for borders and padding

	// Ensure minimal widths for readability
	if stackWidth < 40 {
		stackWidth = 40
	}
	if helpWidth < 30 {
		helpWidth = 30
	}

	// Make panels fill available space
	stackPanelStyle = stackPanelStyle.Width(stackWidth)
	helpPanelStyle = helpPanelStyle.Width(helpWidth)
	logPanelStyle = logPanelStyle.Width(m.viewportWidth - 4) // Account for borders and padding
	containerStyle = containerStyle.Width(m.viewportWidth - 4)
	headerStyle = headerStyle.Width(m.viewportWidth)

	// Application header - now full width
	header := headerStyle.Render(fmt.Sprintf("DOCKER STACK MANAGER | Active: %d/%d services", m.activeServices, m.totalServices))

	if m.state == "stack" {
		// Stack selection panel
		stackList := ""
		for i, stack := range m.stacks {
			stats := m.stackStats[stack]
			statusInfo := fmt.Sprintf("[%s %d • %s %d • %s %d]",
				statusRunning.Render("●"), stats.Running,
				statusStopped.Render("●"), stats.Stopped,
				statusOther.Render("●"), stats.Other)

			if i == m.selectedStack {
				stackList += selectedStyle.Render(fmt.Sprintf("❯ %s %s\n", stack, statusInfo))
			} else {
				stackList += unselectedStyle.Render(fmt.Sprintf("  %s %s\n", stack, statusInfo))
			}
		}

		// Help panel with vibrant controls
		helpText := titleStyle.Render("Keyboard Controls") + "\n\n" +
			fmt.Sprintf("%s Navigate stacks\n", selectedStyle.Render("↑/↓")) +
			fmt.Sprintf("%s View containers\n", selectedStyle.Render("Enter")) +
			fmt.Sprintf("%s Action menu\n", selectedStyle.Render("A")) +
			fmt.Sprintf("%s Back/Escape\n", selectedStyle.Render("Esc/B")) +
			fmt.Sprintf("%s Quit application", selectedStyle.Render("Q"))
		helpPanel := helpPanelStyle.Render(helpText)

		// Stack panel with title
		stackPanel := stackPanelStyle.Render(
			titleStyle.Render("Docker Stacks") + "\n" +
				stackList + "\n" +
				instructionStyle.Render("Press 'A' for actions, 'Enter' to view containers"))

		// Log output panel
		logPanel := ""
		if m.logOutput != "" {
			// Calculate max height for log panel to avoid it being too large
			maxLogHeight := m.viewportHeight / 3

			// Truncate log if too long
			logOutputLines := strings.Split(m.logOutput, "\n")
			if len(logOutputLines) > maxLogHeight {
				// Take the last maxLogHeight lines
				logOutputLines = logOutputLines[len(logOutputLines)-maxLogHeight:]
				m.logOutput = strings.Join(logOutputLines, "\n")
			}

			logPanel = logPanelStyle.Render(
				titleStyle.Render("Output Log") + "\n" +
					logStyle.Render(m.logOutput))
		}

		// Combine panels
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, stackPanel, helpPanel)
		view := lipgloss.JoinVertical(lipgloss.Left, header, topRow)

		if m.logOutput != "" {
			view = lipgloss.JoinVertical(lipgloss.Left, view, logPanel)
		}

		if m.debug {
			debugView := debugStyle.Render(fmt.Sprintf("\nDEBUG:\nstate: %s\nselectedStack: %d\nviewport: %dx%d",
				m.state, m.selectedStack, m.viewportWidth, m.viewportHeight))
			view = lipgloss.JoinVertical(lipgloss.Left, view, debugView)
		}

		return view

	} else if m.state == "actionMenu" {
		selectedStack := m.stacks[m.selectedStack]

		// More vibrant action menu
		actionTitle := titleStyle.Render(fmt.Sprintf("Actions for Stack: %s", selectedStack))

		actionOptions := "\n\n" +
			selectedStyle.Render("[R]") + " Restart Stack\n" +
			selectedStyle.Render("[K]") + " Kill Stack\n" +
			selectedStyle.Render("[L]") + " View Logs\n" +
			selectedStyle.Render("[Esc/B]") + " Back to Stack List"

		// Make action menu responsive
		actionMenuStyle = actionMenuStyle.Width(m.viewportWidth / 2).Align(lipgloss.Center)
		actionPanel := actionMenuStyle.Render(actionTitle + actionOptions)

		// Center the action menu in the screen
		centeredPanel := lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight-2, // Account for header
			lipgloss.Center,
			lipgloss.Center,
			actionPanel)

		return lipgloss.JoinVertical(lipgloss.Left, header, centeredPanel)

	} else if m.state == "containerList" {
		selectedStack := m.stacks[m.selectedStack]
		containerList := ""

		if len(m.containers) == 0 {
			containerList = unselectedStyle.Render("No containers found for this stack")
		} else {
			// Header for container list with vibrant styling
			containerList += titleStyle.Render(fmt.Sprintf("%-20s %-15s %-12s %-20s\n", "NAME", "STATUS", "ID", "IMAGE"))
			containerList += fmt.Sprintf("%-20s %-15s %-12s %-20s\n",
				strings.Repeat("━", 18),
				strings.Repeat("━", 12),
				strings.Repeat("━", 10),
				strings.Repeat("━", 18))

			for i, container := range m.containers {
				name := strings.TrimPrefix(container.Names[0], "/")
				if len(name) > 18 {
					name = name[:15] + "..."
				}

				image := container.Image
				if len(image) > 18 {
					image = image[:15] + "..."
				}

				shortID := container.ID[:10]

				status := container.State
				var styledStatus string
				switch status {
				case "running":
					styledStatus = statusRunning.Render(status)
				case "exited", "stopped":
					styledStatus = statusStopped.Render(status)
				default:
					styledStatus = statusOther.Render(status)
				}

				// Show selection indicator for the current container
				prefix := "  "
				if i == m.selectedContainer {
					prefix = "❯ "
					containerList += selectedStyle.Render(fmt.Sprintf("%s%-20s %-15s %-12s %-20s\n",
						prefix, name, styledStatus, shortID, image))
				} else {
					containerList += unselectedStyle.Render(fmt.Sprintf("%s%-20s %-15s %-12s %-20s\n",
						prefix, name, styledStatus, shortID, image))
				}
			}
		}

		containerPanel := containerStyle.Render(
			titleStyle.Render(fmt.Sprintf("Containers in %s", selectedStack)) + "\n" +
				containerList + "\n" +
				instructionStyle.Render("Press Enter to view container logs, Esc/B to go back"))

		return lipgloss.JoinVertical(lipgloss.Left, header, containerPanel)
	} else if m.state == "containerLogs" {
		// New container logs view
		if len(m.containers) == 0 {
			return lipgloss.JoinVertical(lipgloss.Left, header,
				logPanelStyle.Render(unselectedStyle.Render("No container selected")))
		}

		container := m.containers[m.selectedContainer]
		containerName := strings.TrimPrefix(container.Names[0], "/")

		// Make log panel fill available height
		logViewHeight := m.viewportHeight - 8 // Account for borders, header, and instructions
		if logViewHeight < 10 {
			logViewHeight = 10
		}

		// Limit log output height for better display
		logLines := strings.Split(m.logOutput, "\n")
		if len(logLines) > logViewHeight {
			logLines = logLines[len(logLines)-logViewHeight:]
			m.logOutput = strings.Join(logLines, "\n")
		}

		logPanel := logPanelStyle.Height(logViewHeight).Render(
			titleStyle.Render(fmt.Sprintf("Logs: %s (%s)", containerName, container.ID[:10])) + "\n" +
				logStyle.Render(m.logOutput) + "\n" +
				instructionStyle.Render("Press Esc/B to go back to container list"))

		return lipgloss.JoinVertical(lipgloss.Left, header, logPanel)
	}

	return "Unknown state"
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Unable to create Docker client: %v", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	// Use WithAltScreen to enable full-screen mode with proper window size events
	p := tea.NewProgram(
		initialModel(cli, *debug),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Optional: add mouse support for future enhancements
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting the program: %v\n", err)
		os.Exit(1)
	}
}
