package tools

import (
	"github.com/drutigliano19/suse-observability-mcp/internal/stackstate"
	"github.com/drutigliano19/suse-observability-mcp/internal/stackstate/api"
)

// Tools contains all tools for the MCP server
type Tools struct {
	client api.Client
}

// NewTools creates and returns a new Tools instance.
func NewTools(conf *stackstate.StackState) *Tools {
	stsClient, err := api.NewClient(conf)
	if err != nil {
		return nil
	}
	return &Tools{
		client: *stsClient,
	}
}
