package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI based on current state
func (m Model) View() string {
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
		return m.renderStackView(header)
	} else if m.state == "actionMenu" {
		return m.renderActionMenu(header)
	} else if m.state == "containerList" {
		return m.renderContainerList(header)
	} else if m.state == "containerLogs" {
		return m.renderContainerLogs(header)
	}

	return "Unknown state"
}

// renderStackView renders the stack selection view
func (m Model) renderStackView(header string) string {
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
}

// renderActionMenu renders the action menu for a stack
func (m Model) renderActionMenu(header string) string {
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
}

// renderContainerList renders the container list view
func (m Model) renderContainerList(header string) string {
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
}

// renderContainerLogs renders the container logs view
func (m Model) renderContainerLogs(header string) string {
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
