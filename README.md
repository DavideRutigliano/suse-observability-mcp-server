# SUSE Observability MCP Server

An [Model Context Protocol](https://modelcontextprotocol.io/) server that provides AI agents with structured access to SUSE Observability (StackState) data for intelligent troubleshooting and root cause analysis.

## Overview

This MCP server bridges AI language models with SUSE Observability's rich operational data. It exposes topology, metrics, and monitor information through a standardized interface that AI agents can use to:

- Query infrastructure topology and component relationships
- Retrieve real-time metrics and time-series data
- Access monitor states and health checks
- Perform root cause analysis during incidents

The server normalizes StackState's technical identifiers (URNs, metric queries, health states) into LLM-friendly formats, enabling AI agents to provide accurate, data-grounded diagnoses without hallucinations.

## Available Tools

The server currently exposes the following tools for AI agents:

### Metrics Tools

-   **`listMetrics`**: Lists bound metrics for a specific component.
    -   Arguments: `component_id` (integer, required): The ID of the component to list bound metrics for (from topology queries)
    -   Returns: A markdown table showing the bound metrics with their names, units, and query expressions

-   **`getMetrics`**: Query metrics from SUSE Observability over a range of time.
    -   Arguments: 
        - `query` (string, required): The PromQL query to execute
        - `start` (string, required): Start time for the query (e.g., 'now', '1h')
        - `end` (string, required): End time for the query (e.g., 'now', '1h')
        - `step` (string, optional): Query resolution step width (e.g., '15s', '1m', defaults to '1m')
    -   Returns: A markdown table with the visual representation of the query result

### Monitors Tools

-   **`listMonitors`**: Lists monitors for a specific component.
    -   Arguments: `component_id` (integer, required): The ID of the component to list monitors for (from topology queries)
    -   Returns: A markdown table showing monitors associated with the specified component and their current states

### Topology Tools

-   **`getComponents`**: Searches for topology components using STQL filters.
    -   Arguments (all support comma-separated values for multiple items):
        - `names` (string, optional): Component names to match exactly (comma-separated, e.g., 'checkout-service,redis-master')
        - `types` (string, optional): Component types (comma-separated, e.g., 'pod,service,deployment')
        - `healthstates` (string, optional): Health states (comma-separated, e.g., 'CRITICAL,DEVIATING'). Particularly useful to query multiple states at once
        - `domains` (string, optional): Cluster names to filter (comma-separated, e.g., 'prod-cluster,staging-cluster'). Domain represents the cluster name
        - `namespace` (string, optional): Kubernetes namespace to filter (e.g., 'default', 'kube-system')
        - `with_neighbors` (boolean, optional): Include connected components using withNeighborsOf
        - `with_neighbors_levels` (string, optional): Number of levels (1-14) or 'all' (default: 1)
        - `with_neighbors_direction` (string, optional): 'up', 'down', or 'both' (default: 'both')
    -   Note: At least one filter must be provided. All filters use STQL IN operator for efficient multi-value queries
    -   Returns: A markdown table of matching components with their IDs and identifiers

## Build and Run

### Prerequisites
-   Go 1.23 or later

### Build
To build the server, run:
```bash
go build -o suse-observability-mcp-server cmd/server/main.go
```

### Run
To run the server, you need to provide the SUSE Observability API details. You can run it using stdio (default) or HTTP.

**Using Stdio (for MCP clients):**
```bash
./suse-observability-mcp-server \
  -url "https://your-instance.suse.observability.com" \
  -token "YOUR_API_TOKEN" \
  -apitoken
```

**Using HTTP:**
```bash
./suse-observability-mcp-server \
  -http ":8080" \
  -url "https://your-instance.suse.observability.com" \
  -token "YOUR_API_TOKEN" \
  -apitoken
```

### Configuration Flags
-   `-http`: Address for HTTP transport (e.g., ":8080"). If empty, defaults to stdio.
-   `-url`: SUSE Observability API URL
-   `-token`: SUSE Observability API Token
-   `-apitoken`: Use SUSE Observability API Token instead of a Service Token (boolean)

## Resources
*   [Honeycomb: End of Observability](https://www.honeycomb.io/blog/its-the-end-of-observability-as-we-know-it-and-i-feel-fine)
*   [Datadog Remote MCP Server](https://www.datadoghq.com/blog/datadog-remote-mcp-server)
*   [Model Context Protocol Specification](https://modelcontextprotocol.io/specification/2025-06-18/index)
