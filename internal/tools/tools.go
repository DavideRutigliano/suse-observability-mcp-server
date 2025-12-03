package tools

import (
	"suse-observability-mcp/client/suseobservability"
)

type tool struct {
	client *suseobservability.Client
}

// NewFactory returns a tool factory
func NewBaseTool(c *suseobservability.Client) (t *tool) {
	t = new(tool)
	t.client = c
	return
}
