package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	sts "github.com/drutigliano19/suse-observability-mcp/internal/stackstate"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var opts = &slog.HandlerOptions{Level: slog.LevelDebug}
var handler = slog.NewJSONHandler(os.Stdout, opts)
var logger = slog.New(handler)

func init() { slog.SetDefault(logger) }

func loadRespFile(w http.ResponseWriter, path string) {
	path = fmt.Sprintf("../../testdata/%s", path)
	_, err := os.Stat(path)
	if err == nil {
		file, err := os.ReadFile(path)
		if err == nil {
			_, err := w.Write(file)
			if err == nil {
				return
			}
		}
	}
	slog.Info("file not found", "path", path)
	w.WriteHeader(http.StatusNotFound)
}

func getMockServer(conf *sts.StackState, hf http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(hf)
	conf.ApiUrl = server.URL
	return server
}

func getClient(t *testing.T, hf http.HandlerFunc) (*Client, *httptest.Server) {
	conf := getConfig(t)
	server := getMockServer(conf, hf)
	client, _ := NewClient(conf)
	return client, server
}

func TestTraceQuery(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/traces/query", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("start"))
		assert.NotEmpty(t, r.URL.Query().Get("end"))
		assert.NotEmpty(t, r.URL.Query().Get("page"))
		assert.NotEmpty(t, r.URL.Query().Get("pageSize"))
		loadRespFile(w, "api/traces/query/response.json")
	})
	defer server.Close()
	req := &TraceQueryRequest{
		TraceQuery: TraceQuery{
			SpanFilter: SpanFilter{
				Attributes: map[string][]string{
					"service.name": {"PirateJoker"},
				},
			},
			SortBy: []SortBy{
				{
					Field:     SpanSortSpanParentType,
					Direction: SortDirectionAscending,
				},
			},
		},
		Start:    time.Now().Add(-5 * time.Minute),
		End:      time.Now(),
		Page:     0,
		PageSize: 10,
	}
	response, err := client.QueryTraces(req)
	require.NoError(t, err)
	assert.Equal(t, 2, len(response.Traces))
}

func TestGetTrace(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/traces/xxx", r.URL.Path)
		loadRespFile(w, "api/traces/response.json")
	})
	defer server.Close()
	response, err := client.GetTrace("xxx")
	require.NoError(t, err)
	assert.Equal(t, 21, len(response.Spans))
}

func TestViewSnapshot(t *testing.T) {
	query := "type = 'pod' and label = 'namespace:virtual-cluster'"
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/snapshot", r.URL.Path)
		queryReq := ViewSnapshotRequest{}
		err := json.NewDecoder(r.Body).Decode(&queryReq)
		require.NoError(t, err)
		assert.Equal(t, query, queryReq.Query)
		loadRespFile(w, "api/snapshot/response.json")
	})
	defer server.Close()
	// Comment out mock above and uncomment below to test on a live server and debug requests.
	//DumpHttpRequest = true
	//client := NewClient(getConfig(t))
	response, err := client.SnapShotTopologyQuery(query)
	require.NoError(t, err)
	assert.Equal(t, 3, len(response))
}

func TestQuery(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/metrics/query", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("time"))
		assert.NotEmpty(t, r.URL.Query().Get("query"))
		assert.Equal(t, r.URL.Query().Get("timeout"), DefaultTimeout)
		loadRespFile(w, "api/metrics/query/response.json")
	})
	defer server.Close()
	query := `round(sum by (cluster_name, namespace, pod_name)(container_cpu_usage / 1000000000) / sum by (cluster_name, namespace, pod_name) (kubernetes_cpu_requests), 0.001)`
	now := time.Now()
	response, err := client.QueryMetric(query, now, DefaultTimeout)
	require.NoError(t, err)
	assert.Equal(t, "success", response.Status)
}

func TestQueryRange(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/metrics/query_range", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("start"))
		assert.NotEmpty(t, r.URL.Query().Get("end"))
		assert.NotEmpty(t, r.URL.Query().Get("query"))
		assert.NotEmpty(t, r.URL.Query().Get("step"))
		assert.Equal(t, r.URL.Query().Get("timeout"), DefaultTimeout)
		loadRespFile(w, "api/metrics/query_range/response.json")
	})
	defer server.Close()
	query := `sum by (cluster_name) (max_over_time(kubernetes_state_node_count{cluster_name="susecon-frb-cluster-0"}[${__interval}]))`
	now := time.Now()
	begin := now.Add(-5 * time.Minute)
	response, err := client.QueryRangeMetric(query, begin, now, "1m", DefaultTimeout)
	require.NoError(t, err)
	assert.Equal(t, "success", response.Status)
}

func TestClientConnection(t *testing.T) {
	conf := getConfig(t)
	client, _ := NewClient(conf)
	status, err := client.Status()
	require.NoError(t, err, `Not expecting err %v`, err)
	assert.Equal(t, status.Version.Major, 6)
}

func TestTopologyQuery(t *testing.T) {
	conf := getConfig(t)
	client, _ := NewClient(conf)
	res, err := client.TopologyQuery("type = 'service' and label in ('namespace:kube-system')", "", false)
	require.NoError(t, err)
	require.True(t, res.Success, `Expected to be successful but was %s`, toJson(res))
	assert.True(t, len(res.Data) > 0)
	fmt.Println(toJson(res))
}

func TestTopologyStreamQuery(t *testing.T) {
	conf := getConfig(t)
	client, _ := NewClient(conf)
	res, err := client.TopologyStreamQuery("type = 'service' and label in ('namespace:kube-system')", "", true)
	require.NoError(t, err)
	require.True(t, res.Success, `Expected to be successful but was %s`, toJson(res))
	assert.True(t, len(res.Data) > 0)
	fmt.Println(toJson(res))
}

func TestGetEvents(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/events", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		var eventReq EventListRequest
		err := json.NewDecoder(r.Body).Decode(&eventReq)
		require.NoError(t, err)
		assert.Equal(t, "type = 'pod'", eventReq.TopologyQuery)
		assert.Equal(t, 100, eventReq.Limit)

		// Return mock response
		resp := EventItemsWithTotal{
			Items: []TopologyEvent{
				{
					Identifier:         "event-1",
					Name:               "Test Event",
					Category:           EventCategoryAlerts,
					EventType:          "test.event",
					EventTime:          time.Now().UnixMilli(),
					ProcessedTime:      time.Now().UnixMilli(),
					Source:             "test-source",
					ElementIdentifiers: []string{"elem-1"},
					Elements:           []interface{}{},
					SourceLinks:        []SourceLink{},
					Tags:               []EventTag{},
					Data:               map[string]interface{}{},
				},
			},
			Total: 1,
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	now := time.Now()
	req := &EventListRequest{
		StartTimestampMs: now.Add(-1 * time.Hour).UnixMilli(),
		EndTimestampMs:   now.UnixMilli(),
		TopologyQuery:    "type = 'pod'",
		Limit:            100,
	}
	response, err := client.GetEvents(req)
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.Total)
	assert.Equal(t, 1, len(response.Items))
	assert.Equal(t, "event-1", response.Items[0].Identifier)
	assert.Equal(t, "Test Event", response.Items[0].Name)
}

func TestGetEvent(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/events/event-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.NotEmpty(t, r.URL.Query().Get("startTimestampMs"))
		assert.NotEmpty(t, r.URL.Query().Get("endTimestampMs"))

		// Return mock response
		resp := TopologyEvent{
			Identifier:         "event-123",
			Name:               "Specific Event",
			Category:           EventCategoryDeployments,
			EventType:          "deployment.created",
			EventTime:          time.Now().UnixMilli(),
			ProcessedTime:      time.Now().UnixMilli(),
			Source:             "kubernetes",
			ElementIdentifiers: []string{"elem-1", "elem-2"},
			Elements:           []interface{}{},
			SourceLinks: []SourceLink{
				{Title: "View in K8s", URL: "https://example.com"},
			},
			Tags: []EventTag{
				{Key: "env", Value: "production"},
			},
			Data: map[string]interface{}{
				"deployment": "my-app",
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	now := time.Now()
	response, err := client.GetEvent("event-123", now.Add(-1*time.Hour).UnixMilli(), now.UnixMilli())
	require.NoError(t, err)
	assert.Equal(t, "event-123", response.Identifier)
	assert.Equal(t, "Specific Event", response.Name)
	assert.Equal(t, EventCategoryDeployments, response.Category)
	assert.Equal(t, 1, len(response.SourceLinks))
	assert.Equal(t, 1, len(response.Tags))
	assert.Equal(t, "production", response.Tags[0].Value)
}

func toJson(a any) string {
	marshal, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Failed to marshall json. %v", err)
	}
	return string(marshal)
}

func TestGetMonitors(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/monitors", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Return mock response
		resp := MonitorList{
			Monitors: []Monitor{
				{
					Id:                  1,
					Name:                "CPU Monitor",
					Identifier:          "urn:monitor:cpu",
					Description:         "Monitors CPU usage",
					FunctionId:          100,
					Arguments:           []map[string]interface{}{},
					IntervalSeconds:     60,
					Tags:                []string{"cpu", "infrastructure"},
					Source:              "stackstate",
					CanEdit:             true,
					CanClone:            true,
					Status:              MonitorStatusEnabled,
					RuntimeStatus:       MonitorRuntimeStatusEnabled,
					LastUpdateTimestamp: time.Now().UnixMilli(),
				},
				{
					Id:                  2,
					Name:                "Memory Monitor",
					Identifier:          "urn:monitor:memory",
					FunctionId:          101,
					Arguments:           []map[string]interface{}{},
					IntervalSeconds:     120,
					Tags:                []string{"memory", "infrastructure"},
					Source:              "stackstate",
					CanEdit:             true,
					CanClone:            true,
					Status:              MonitorStatusEnabled,
					RuntimeStatus:       MonitorRuntimeStatusEnabled,
					LastUpdateTimestamp: time.Now().UnixMilli(),
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	response, err := client.GetMonitors()
	require.NoError(t, err)
	assert.Equal(t, 2, len(response.Monitors))
	assert.Equal(t, "CPU Monitor", response.Monitors[0].Name)
	assert.Equal(t, "Memory Monitor", response.Monitors[1].Name)
}

func TestGetMonitor(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/monitors/123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Return mock response
		resp := Monitor{
			Id:                  123,
			Name:                "Disk Monitor",
			Identifier:          "urn:monitor:disk",
			Description:         "Monitors disk usage",
			FunctionId:          102,
			Arguments:           []map[string]interface{}{{"threshold": 80}},
			RemediationHint:     "Check disk space and clean up logs",
			IntervalSeconds:     300,
			Tags:                []string{"disk", "storage"},
			Source:              "stackstate",
			SourceDetails:       "Auto-generated",
			CanEdit:             true,
			CanClone:            true,
			Status:              MonitorStatusEnabled,
			RuntimeStatus:       MonitorRuntimeStatusEnabled,
			Dummy:               false,
			LastUpdateTimestamp: time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	response, err := client.GetMonitor("123")
	require.NoError(t, err)
	assert.Equal(t, int64(123), response.Id)
	assert.Equal(t, "Disk Monitor", response.Name)
	assert.Equal(t, "urn:monitor:disk", response.Identifier)
	assert.Equal(t, "Check disk space and clean up logs", response.RemediationHint)
	assert.Equal(t, 1, len(response.Arguments))
}

func TestGetMonitorsOverview(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/monitors/overview", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Return mock response
		resp := MonitorOverviewList{
			Monitors: []MonitorOverview{
				{
					Monitor: Monitor{
						Id:                  1,
						Name:                "API Monitor",
						FunctionId:          200,
						Arguments:           []map[string]interface{}{},
						IntervalSeconds:     60,
						Tags:                []string{"api"},
						Source:              "stackstate",
						CanEdit:             true,
						CanClone:            true,
						Status:              MonitorStatusEnabled,
						RuntimeStatus:       MonitorRuntimeStatusEnabled,
						LastUpdateTimestamp: time.Now().UnixMilli(),
					},
					Function: MonitorFunction{
						Id:                  200,
						Name:                "Metric Health Check",
						Identifier:          "urn:function:metric-health",
						Description:         "Check metric thresholds",
						LastUpdateTimestamp: time.Now().UnixMilli(),
					},
					Errors: []MonitorError{
						{
							Error: "Failed to query metrics",
							Count: 5,
							Level: "WARNING",
						},
					},
					RuntimeMetrics: MonitorRuntimeMetrics{
						GroupCount: 3,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	response, err := client.GetMonitorsOverview()
	require.NoError(t, err)
	assert.Equal(t, 1, len(response.Monitors))
	assert.Equal(t, "API Monitor", response.Monitors[0].Monitor.Name)
	assert.Equal(t, "Metric Health Check", response.Monitors[0].Function.Name)
	assert.Equal(t, 1, len(response.Monitors[0].Errors))
	assert.Equal(t, 3, response.Monitors[0].RuntimeMetrics.GroupCount)
}

func TestGetMonitorCheckStates(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/monitors/456/checkStates", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "CRITICAL", r.URL.Query().Get("healthState"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		// Return mock response
		resp := MonitorCheckStates{
			States: []ViewCheckState{
				{
					CheckStateId:          "check-1",
					TopologyElementId:     1001,
					TopologyElementIdType: "id",
					Name:                  "pod-xyz CPU check",
					Health:                "CRITICAL",
					Message:               "CPU usage above 90%",
				},
				{
					CheckStateId:          "check-2",
					TopologyElementId:     1002,
					TopologyElementIdType: "id",
					Name:                  "pod-abc CPU check",
					Health:                "CRITICAL",
					Message:               "CPU usage above 95%",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	response, err := client.GetMonitorCheckStates("456", "CRITICAL", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, len(response.States))
	assert.Equal(t, "check-1", response.States[0].CheckStateId)
	assert.Equal(t, "CRITICAL", response.States[0].Health)
	assert.Equal(t, "CPU usage above 90%", response.States[0].Message)
}

func TestGetMonitorCheckStatus(t *testing.T) {
	client, server := getClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/monitor/checkStatus/789", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.NotEmpty(t, r.URL.Query().Get("topologyTime"))

		// Return mock response
		now := time.Now().UnixMilli()
		resp := MonitorCheckStatus{
			Id:                 789,
			CheckStateId:       "check-789",
			Message:            "High CPU usage detected",
			Reason:             "Threshold exceeded",
			Health:             "CRITICAL",
			TriggeredTimestamp: now,
			Metrics: []MonitorCheckStatusMetric{
				{
					Type:        "MetricHealthCheck",
					Name:        "CPU Usage",
					Description: "Container CPU usage percentage",
					Unit:        "percent",
					Queries: []MonitorCheckStatusQuery{
						{
							Query:                       "cpu_usage{pod='xyz'}",
							Alias:                       "CPU",
							ComponentIdentifierTemplate: "urn:kubernetes:pod:{{pod}}",
						},
					},
				},
			},
			Component: MonitorCheckStatusComponent{
				Id:         1001,
				Identifier: "urn:kubernetes:pod:xyz",
				Name:       "pod-xyz",
				Type:       "pod",
				IconBase64: "base64data",
			},
			MonitorId:            123,
			MonitorName:          "Pod CPU Monitor",
			MonitorDescription:   "Monitors CPU usage for pods",
			TroubleshootingSteps: "1. Check pod logs\n2. Review resource limits",
			TopologyTime:         now,
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	topologyTime := time.Now().UnixMilli()
	response, err := client.GetMonitorCheckStatus(789, topologyTime)
	require.NoError(t, err)
	assert.Equal(t, int64(789), response.Id)
	assert.Equal(t, "check-789", response.CheckStateId)
	assert.Equal(t, "CRITICAL", response.Health)
	assert.Equal(t, "High CPU usage detected", response.Message)
	assert.Equal(t, "Pod CPU Monitor", response.MonitorName)
	assert.Equal(t, 1, len(response.Metrics))
	assert.Equal(t, "CPU Usage", response.Metrics[0].Name)
	assert.Equal(t, "pod-xyz", response.Component.Name)
	assert.Contains(t, response.TroubleshootingSteps, "Check pod logs")
}

func getConfig(t *testing.T) *sts.StackState {
	require.NoError(t, godotenv.Load("../../.env"))
	return &sts.StackState{
		ApiUrl:   os.Getenv("STS_URL"),
		ApiKey:   os.Getenv("STS_API_KEY"),
		ApiToken: os.Getenv("STS_TOKEN"),
	}
}
