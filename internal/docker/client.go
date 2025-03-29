package docker

import (
	"github.com/docker/docker/client"
)

// NewClient creates a new Docker client
func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}
