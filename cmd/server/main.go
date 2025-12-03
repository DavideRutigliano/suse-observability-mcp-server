package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"suse-observability-mcp/client/suseobservability"
	"suse-observability-mcp/internal/tools"
)

func main() {
	// SUSE Observability flags
	url := flag.String("url", "", "SUSE Observability API URL")
	token := flag.String("token", "", "SUSE Observability API Token")
	useAPIToken := flag.Bool("apitoken", false, "Indicates if the token is an API token, instead of a service token")

	// MCP server flags
	listenAddr := flag.String("http", "", "address for http transport, defaults to stdio")
	flag.Parse()

	client, err := suseobservability.NewClient(*url, *token, *useAPIToken)
	if err != nil {
		return
	}

	mcpTools := tools.NewBaseTool(client)

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "SUSE Observability MCP server", Version: "v0.0.1"}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "listMetrics",
		Description: `Searches for metrics in SUSE Observability by pattern and shows their available label keys.
		Arguments:
		- search_pattern (required): A regex pattern to search for metrics (e.g., 'cpu', 'memory', 'redis.*').
		Returns:
		A markdown table showing matching metric names and their available label keys (dimensions)`},
		mcpTools.ListMetrics,
	)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getMetrics",
		Description: `Query metrics from SUSE Observability over a range of time.
		Arguments:
		- query (required): The PromQL query to execute.
		- start (required): Start time for the query (e.g., 'now', '1h', '24h').
		- end (required): End time for the query (e.g., 'now', '1h').
		- step (optional): Query resolution step width (e.g., '15s', '1m', '5m'). Default: '1m'.
		Returns:
		A markdown table showing the time series data with timestamps, values, and labels.`},
		mcpTools.QueryRangeMetric,
	)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getMonitors",
		Description: `Lists active monitors filtered by health state with component details.
		Arguments:
		- state (optional): Filter by state - 'CRITICAL', 'DEVIATING', or 'UNKNOWN' (default: CRITICAL).
		Returns:
		Monitors in the specified state with affected component names and URNs`},
		mcpTools.GetMonitors,
	)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name: "getComponents",
		Description: `Searches for topology components using STQL filters.
		Arguments:
		- query (optional): Raw STQL query for advanced filtering (overrides other filters).
		  Example: 'layer = "Containers" AND (healthstate = "CRITICAL" OR healthstate = "DEVIATING")'
		- name_pattern (optional): Component name with wildcard support (e.g., 'checkout*', 'redis*').
		- type (optional): Component type filter (e.g., 'pod', 'service', 'deployment').
		- layer (optional): Layer filter (e.g., 'Containers', 'Services').
		- domain (optional): Domain filter (e.g., 'cluster.example.com').
		- healthstate (optional): Health state filter (e.g., 'CRITICAL', 'DEVIATING', 'CLEAR').
		- with_neighbors (optional): Include connected components using withNeighborsOf (simple filters only).
		- with_neighbors_levels (optional): Number of levels (1-14) or 'all' (default: 1).
		- with_neighbors_direction (optional): 'up', 'down', or 'both' (default: both).
		Either 'query' or at least one simple filter must be provided.
		Returns:
		A markdown table of matching components with their IDs and identifiers`},
		mcpTools.GetComponents,
	)

	if *listenAddr == "" {
		// Run the server on the stdio transport.
		if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			slog.Error("Server failed", "error", err)
		}
	} else {
		// Create a streamable HTTP handler.
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return mcpServer
		}, nil)

		// Run the server on the HTTP transport.
		slog.Info("Server listening", "address", *listenAddr)
		if err := http.ListenAndServe(*listenAddr, handler); err != nil {
			slog.Error("Server failed", "error", err)
		}
	}
}
