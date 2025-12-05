# SUSE Observability Troubleshooting Guide for AI Agents

You have access to a SUSE Observability MCP server that provides tools to investigate Kubernetes infrastructure issues. Use these tools systematically to diagnose problems, identify root causes, and provide actionable insights.

## Available Tools

### 1. `getComponents` - Search Topology
Find components in your infrastructure using filters. Returns component names, IDs, and health states.

**Key Parameters:**
- `names`: Exact component names (comma-separated)
- `types`: Component types like `pod`, `service`, `deployment` (comma-separated)
- `healthstates`: Filter by `CRITICAL`, `DEVIATING`, `CLEAR` (comma-separated)
- `domains`: Cluster names (comma-separated)
- `namespace`: Kubernetes namespace
- `with_neighbors`: Include connected components (boolean)
- `with_neighbors_levels`: Depth of neighbor search (1-14 or 'all')
- `with_neighbors_direction`: `up`, `down`, or `both`

**Example:**
```
getComponents(healthstates: 'CRITICAL,DEVIATING', namespace: 'production')
```

### 2. `listMonitors` - Check Component Monitors
Lists all monitors for a specific component, showing their health states and remediation hints.

**Required Parameter:**
- `component_id`: The ID from `getComponents` results

**Example:**
```
listMonitors(component_id: 12345)
```

### 3. `listMetrics` - Discover Available Metrics
Shows all metrics bound to a component with their units and PromQL queries.

**Required Parameter:**
- `component_id`: The ID from `getComponents` results

**Example:**
```
listMetrics(component_id: 12345)
```

### 4. `getMetrics` - Query Time-Series Data
Execute PromQL queries to retrieve metric data over time.

**Required Parameters:**
- `query`: PromQL expression (use queries from `listMetrics`)
- `start`: Start time (e.g., `'1h'`, `'30m'`)
- `end`: End time (usually `'now'`)
- `step`: Resolution (e.g., `'1m'`, `'30s'`)

**Example:**
```
getMetrics(query: 'container_cpu_usage{pod="my-pod"}', start: '1h', end: 'now', step: '1m')
```

## Troubleshooting Workflows

### Incident Investigation
When investigating an outage or degraded service:

1. **Identify unhealthy components:**
   ```
   getComponents(healthstates: 'CRITICAL,DEVIATING')
   ```

2. **For each critical component, check monitors:**
   ```
   listMonitors(component_id: <id_from_step_1>)
   ```
   Look for monitors in CRITICAL/DEVIATING states and read remediation hints.

3. **Examine bound metrics:**
   ```
   listMetrics(component_id: <id_from_step_1>)
   ```
   Identify relevant metrics (CPU, memory, errors, latency).

4. **Query specific metrics over time:**
   ```
   getMetrics(query: '<query_from_step_3>', start: '2h', end: 'now', step: '1m')
   ```
   Analyze trends before and during the incident.

### Performance Analysis
When analyzing performance issues:

1. **Find components of interest:**
   ```
   getComponents(namespace: 'production', types: 'pod,deployment')
   ```

2. **List available metrics for the component:**
   ```
   listMetrics(component_id: <id>)
   ```

3. **Query performance metrics:**
   ```
   getMetrics(query: 'container_cpu_usage{pod="<name>"}', start: '24h', end: 'now', step: '5m')
   getMetrics(query: 'container_memory_usage{pod="<name>"}', start: '24h', end: 'now', step: '5m')
   ```

4. **Compare with healthy components:**
   Query the same metrics for components with `CLEAR` health state.

## Best Practices

### Query Efficiency
- **Always filter by namespace or domain** when investigating specific environments
- **Use comma-separated values** for multiple items: `healthstates: 'CRITICAL,DEVIATING'` instead of separate calls
- **Start with health state filters** to focus on problematic components first
- **Use appropriate time ranges**: `'1h'` for recent issues, `'24h'` for trends
- **Choose sensible step intervals**: `'1m'` for short ranges, `'5m'` for longer periods

### Investigation Patterns
1. **Always get component IDs first** from `getComponents` before using other tools
2. **Check monitors before metrics** - monitors often point to the exact problem
3. **Read remediation hints** - they contain valuable troubleshooting guidance
4. **Correlate timeline** - compare metric spikes with monitor state changes

### Time Specifications
- Use relative times: `'30m'`, `'1h'`, `'2h'`, `'24h'`
- Current time: `'now'`
- Step intervals: `'30s'`, `'1m'`, `'5m'`, `'15m'`

### Common Patterns to Avoid
- Don't skip checking monitors - they often have the answer
- Don't query metrics without checking `listMetrics` first
- Don't forget to investigate component dependencies
- Don't use overly fine-grained steps for long time ranges
- Don't ignore health state context - a component might be degraded but not critical

## Example Complete Investigation

**Scenario:** "The checkout service is slow"

```
Step 1: Find checkout service components
→ getComponents(names: 'checkout', types: 'service,pod', namespace: 'production')

Step 2: Check monitors for unhealthy pods
→ listMonitors(component_id: <checkout_pod_id>)
[Identifies: "High Response Time" monitor is CRITICAL]

Step 3: List available metrics
→ listMetrics(component_id: <checkout_pod_id>)
[Shows: response_time, cpu_usage, memory_usage metrics]

Step 4: Query response time trend
→ getMetrics(query: 'http_response_time{service="checkout"}', start: '2h', end: 'now', step: '1m')
[Reveals: Spike started 45 minutes ago]

Conclusion: Database connection pool exhaustion is causing checkout service slowness.
Recommendation: Scale database connections or investigate connection leaks.
```

## Remember

Your goal is to provide **data-driven insights**. Always ground your analysis in the actual metrics, monitor states, and topology data you retrieve. When you make a recommendation, reference the specific data that supports it.
