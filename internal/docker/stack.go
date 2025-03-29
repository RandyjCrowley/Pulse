package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// ListStacks returns all Docker stacks
func ListStacks(ctx context.Context, cli *client.Client) ([]string, error) {
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

// ListContainers returns all containers in a stack
func ListContainers(ctx context.Context, cli *client.Client, stackName string) ([]types.Container, error) {
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

// KillStack kills a Docker stack by removing all its services
func KillStack(ctx context.Context, cli *client.Client, stackName string) error {
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

// RestartStack restarts a Docker stack
func RestartStack(ctx context.Context, cli *client.Client, stackName string) error {
	if err := KillStack(ctx, cli, stackName); err != nil {
		return fmt.Errorf("error killing stack: %v", err)
	}

	return fmt.Errorf("full stack restart requires external deployment mechanism")
}

// ViewStackLogs returns logs for all services in a stack
func ViewStackLogs(ctx context.Context, cli *client.Client, stackName string) (string, error) {
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

// ViewContainerLogs returns logs for a specific container
func ViewContainerLogs(ctx context.Context, cli *client.Client, containerID string) (string, error) {
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
