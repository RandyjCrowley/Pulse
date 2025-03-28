package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
}

func initialModel(cli *client.Client) model {
	stacks, err := listStacks(context.Background(), cli)
	if err != nil {
		log.Fatalf("Error listing stacks: %v", err)
	}

	return model{
		stacks:        stacks,
		selectedStack: 0,
		cli:           cli,
		state:         "stack",
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
				m.state = "actionMenu"
			}
		case "up":
			if m.state == "stack" && m.selectedStack > 0 {
				m.selectedStack--
			}
		case "down":
			if m.state == "stack" && m.selectedStack < len(m.stacks)-1 {
				m.selectedStack++
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
		case "escape":
			m.state = "stack"
		}
	}
	return m, nil
}

func (m model) View() string {
	var view string

	if m.state == "stack" {
		view = "Docker Stacks:\n"
		for i, stack := range m.stacks {
			if i == m.selectedStack {
				view += fmt.Sprintf("> %s\n", stack)
			} else {
				view += fmt.Sprintf("  %s\n", stack)
			}
		}

		view += "\nUse arrow keys to navigate, 'enter' to select a stack, 'q' to quit."
		
		if m.logOutput != "" {
			view += fmt.Sprintf("\n\nLast Action Output:\n%s", m.logOutput)
		}
	} else if m.state == "actionMenu" {
		view = fmt.Sprintf("\nActions for stack: %s\n", m.stacks[m.selectedStack])
		view += "Press 'r' to restart, 'k' to kill, 'l' to view logs, 'esc' to go back, 'q' to quit.\n"
	}

	return view
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Unable to create Docker client: %v", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	p := tea.NewProgram(initialModel(cli))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting the program: %v\n", err)
		os.Exit(1)
	}
}