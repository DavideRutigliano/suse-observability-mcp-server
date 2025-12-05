package suseobservability

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	rq "github.com/carlmjohnson/requests"
)

type Client struct {
	soURL    string
	token    string
	apiToken bool
}

var (
	transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
)

func NewClient(soURL, serviceToken string, apiToken bool) (c *Client, err error) {
	_, err = url.ParseRequestURI(soURL)
	if err != nil {
		return
	}
	c = new(Client)
	c.soURL, _ = strings.CutSuffix(soURL, "/")
	c.token = serviceToken
	c.apiToken = apiToken
	return
}

const (
	GroovyScript   string = "GroovyScript"
	DefaultTimeout string = "10s"
)

func (c Client) Status(ctx context.Context) (*ServerInfo, error) {
	var s ServerInfo
	err := c.apiRequests("server/info").
		ToJSON(&s).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c Client) GetTrace(ctx context.Context, id string) (*Trace, error) {
	var res Trace
	err := c.apiRequests(fmt.Sprintf("traces/%s", id)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c Client) GetTraceSpan(ctx context.Context, traceId string, spanId string) (*Span, error) {
	var res Span
	err := c.apiRequests(fmt.Sprintf("traces/%s/spans/%s", traceId, spanId)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c Client) QueryTraces(ctx context.Context, req *TraceQueryRequest) (*TraceQueryResponse, error) {
	var res TraceQueryResponse
	err := c.apiRequests("traces/query").
		Post().
		Param("end", toMs(req.End)).
		Param("start", toMs(req.Start)).
		Param("page", strconv.Itoa(req.Page)).
		Param("pageSize", strconv.Itoa(req.PageSize)).
		BodyJSON(req.TraceQuery).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func toMs(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

// ListMetrics fetches all available metrics
func (c Client) ListMetrics(ctx context.Context, start, end time.Time) ([]string, error) {
	var res struct {
		Data []string `json:"data"`
	}
	err := c.apiRequests("metrics/label/__name__/values").
		Param("start", toMs(start)).
		Param("end", toMs(end)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// QueryMetric is the instant query at a single point in time.
// The endpoint evaluates an instant query at a single point in time.
// Query is the promql query and Time the single point.
// Timeout is in the form "<number><unit (y|w|d|h|m|s|ms)>". Example 10ms.
func (c Client) QueryMetric(ctx context.Context, query string, at time.Time, timeout string) (*MetricQueryResponse, error) {
	var m MetricQueryResponse
	err := c.apiRequests("metrics/query").
		Param("query", query).
		Param("timeout", timeout).
		Param("time", toMs(at)).
		ToJSON(&m).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// QueryRangeMetric is the query over a range of time
// The endpoint evaluates an expression query over a range of time
// Query is the promql query. Start and End times indicate the range.
// Step is the promstep in the same format as Timeout.
// Timeout is in the form "<number><unit (y|w|d|h|m|s|ms)>". Example 10ms.
func (c Client) QueryRangeMetric(ctx context.Context, query string, start time.Time, end time.Time, step, timeout string) (*MetricQueryResponse, error) {
	var m MetricQueryResponse
	err := c.apiRequests("metrics/query_range").
		Param("query", query).
		Param("timeout", timeout).
		Param("step", step).
		Param("start", toMs(start)).
		Param("end", toMs(end)).
		ToJSON(&m).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (c Client) SnapShotTopologyQuery(ctx context.Context, query string) ([]ViewComponent, error) {
	req := NewViewSnapshotRequest(query)
	res, err := c.ViewSnapshot(ctx, req)
	if err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, errors.New(res.Errors[0].Message)
	}
	return res.Components, nil
}

func (c Client) ViewSnapshot(ctx context.Context, req *ViewSnapshotRequest) (*ViewSnapshotResponse, error) {
	var res querySnapshotResult
	var e ErrorResp
	err := c.apiRequests("snapshot").
		Post().
		BodyJSON(&req).
		ErrorJSON(&e).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		if len(e.Errors) > 0 {
			return &ViewSnapshotResponse{Success: false, Errors: e.Errors}, nil
		}
		return nil, err
	}
	res.ViewSnapshotResponse.Success = true
	return &res.ViewSnapshotResponse, nil
}

func (c Client) Layers() (*map[int64]NodeType, error) {
	return c.getNodesOfType("Layer")
}

func (c Client) ComponentTypes() (*map[int64]NodeType, error) {
	return c.getNodesOfType("ComponentType")
}

func (c Client) RelationTypes() (*map[int64]NodeType, error) {
	return c.getNodesOfType("RelationType")
}

func (c Client) Domains() (*map[int64]NodeType, error) {
	return c.getNodesOfType("Domain")
}

func (c Client) getNodesOfType(t string) (*map[int64]NodeType, error) {
	var res []NodeType
	err := c.apiRequests(fmt.Sprintf("node/%s", t)).
		ToJSON(&res).
		Fetch(context.Background())
	if err != nil {
		return nil, err
	}
	nodes := make(map[int64]NodeType, len(res))
	for _, r := range res {
		nodes[r.ID] = r
	}
	return &nodes, nil
}

func (c Client) TopologyQuery(ctx context.Context, query string, at string, fullLoad bool) (*TopoQueryResponse, error) {
	query, at = sanitizeQuery(query, at)
	method := "components"
	if fullLoad {
		method = "fullComponents"
	}
	body := fmt.Sprintf(`Topology.query('%s')%s.%s()`, query, at, method)
	return c.executeTopoScript(ctx, scriptRequest{
		ReqType: GroovyScript,
		Body:    body,
	})
}

func (c Client) TopologyStreamQuery(ctx context.Context, query string, at string, withSyncData bool) (*TopoQueryResponse, error) {
	query, at = sanitizeQuery(query, at)
	method := ""
	if withSyncData {
		method = ".withSynchronizationData()"
	}
	body := fmt.Sprintf(`TopologyStream.query('%s')%s%s`, query, at, method)
	return c.executeTopoScript(ctx, scriptRequest{
		ReqType: GroovyScript,
		Body:    body,
	})
}

func sanitizeQuery(query string, at string) (string, string) {
	query = strings.ReplaceAll(query, "'", "\"")
	if at != "" {
		at = fmt.Sprintf(".at('%s')", at)
	}
	return query, at
}

func (c Client) executeTopoScript(ctx context.Context, req scriptRequest) (*TopoQueryResponse, error) {
	var r SuccessResp
	var e ErrorResp
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	slog.Debug("request", "body", string(b))
	err = c.apiRequests("script").
		BodyJSON(&req).
		ErrorJSON(&e).
		ToJSON(&r).
		Fetch(ctx)
	if err != nil {
		if e.Errors != nil {
			return &TopoQueryResponse{Success: false, Errors: e.Errors, Data: nil}, nil
		}
		return nil, err
	}
	return &TopoQueryResponse{Success: true, Errors: nil, Data: r.Result}, nil
}

// GetEvents retrieves a list of events based on topology and time selections
func (c Client) GetEvents(ctx context.Context, req *EventListRequest) (*EventItemsWithTotal, error) {
	var res EventItemsWithTotal
	err := c.apiRequests("events").
		Post().
		BodyJSON(req).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetEvent retrieves a specific event by its identifier
func (c Client) GetEvent(ctx context.Context, eventId string, startMs int64, endMs int64) (*TopologyEvent, error) {
	var res TopologyEvent
	err := c.apiRequests(fmt.Sprintf("events/%s", eventId)).
		Param("startTimestampMs", strconv.FormatInt(startMs, 10)).
		Param("endTimestampMs", strconv.FormatInt(endMs, 10)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c Client) apiRequests(endpoint string) *rq.Builder {
	uri := fmt.Sprintf("%s/api/%s", c.soURL, endpoint)
	return request(uri).
		Header(c.GetXHeader(), c.token)
}

func (c Client) GetXHeader() string {
	if c.apiToken {
		return "X-API-Token"
	}
	return "X-API-Key"
}

func request(uri string) *rq.Builder {
	b := rq.URL(uri).
		ContentType("application/json").
		Transport(transport)
	return b
}

// GetMonitors lists all available monitors
func (c Client) GetMonitors(ctx context.Context) (*MonitorList, error) {
	var res MonitorList
	err := c.apiRequests("monitors").
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetMonitor retrieves a specific monitor by its identifier (ID or URN)
func (c Client) GetMonitor(ctx context.Context, monitorIdOrUrn string) (*Monitor, error) {
	var res Monitor
	err := c.apiRequests(fmt.Sprintf("monitors/%s", monitorIdOrUrn)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetMonitorsOverview lists all available monitors with their function and runtime data
func (c Client) GetMonitorsOverview(ctx context.Context) (*MonitorOverviewList, error) {
	var res MonitorOverviewList
	err := c.apiRequests("monitors/overview").
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetMonitorCheckStates returns the check states that a monitor generated
func (c Client) GetMonitorCheckStates(ctx context.Context, monitorIdOrUrn string, healthState string, limit int, timestamp int64) (*MonitorCheckStates, error) {
	var res MonitorCheckStates
	req := c.apiRequests(fmt.Sprintf("monitors/%s/checkStates", monitorIdOrUrn))

	if healthState != "" {
		req.Param("healthState", healthState)
	}
	if limit > 0 {
		req.Param("limit", strconv.Itoa(limit))
	}
	if timestamp > 0 {
		req.Param("timestamp", strconv.FormatInt(timestamp, 10))
	}

	err := req.ToJSON(&res).Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetMonitorCheckStatus returns a monitor check status by check state id
func (c Client) GetMonitorCheckStatus(ctx context.Context, id int64, topologyTime int64) (*MonitorCheckStatus, error) {
	var res MonitorCheckStatus
	req := c.apiRequests(fmt.Sprintf("monitor/checkStatus/%d", id))

	if topologyTime > 0 {
		req.Param("topologyTime", strconv.FormatInt(topologyTime, 10))
	}

	err := req.ToJSON(&res).Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetBoundMetricsWithData retrieves bound metrics for a specific component
func (c Client) GetBoundMetricsWithData(ctx context.Context, componentID int64, start, end time.Time) (*BoundMetricsResponse, error) {
	var res BoundMetricsResponse
	err := c.apiRequests(fmt.Sprintf("components/%d/boundMetricsWithData", componentID)).
		Param("startSeconds", strconv.FormatInt(start.Unix(), 10)).
		Param("endSeconds", strconv.FormatInt(end.Unix(), 10)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetComponent retrieves a component by ID with full details including synced check states
func (c Client) GetComponent(ctx context.Context, componentID int64) (*ComponentResponse, error) {
	var res ComponentResponse
	err := c.apiRequests(fmt.Sprintf("components/%d", componentID)).
		ToJSON(&res).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
