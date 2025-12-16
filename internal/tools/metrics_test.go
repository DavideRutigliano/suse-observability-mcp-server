package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListMetrics(t *testing.T) {
	mockClient := new(MockSuseObservabilityClient)
	tools := NewBaseTool(mockClient)
	ctx := context.Background()

	t.Run("success with metrics", func(t *testing.T) {
		componentID := int64(123)
		params := ListMetricsParams{ComponentID: componentID}

		expectedResponse := &suseobservability.BoundMetricsResponse{
			BoundMetrics: []suseobservability.BoundMetric{
				{
					Name: "cpu_usage",
					Unit: "percent",
					BoundQueries: []suseobservability.BoundQuery{
						{Expression: "avg(cpu_usage)"},
					},
				},
			},
		}

		mockClient.On("GetBoundMetricsWithData", ctx, componentID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return(expectedResponse, nil).Once()

		result, _, err := tools.ListMetrics(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "cpu_usage")
		assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "percent")
		assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "avg(cpu_usage)")
	})

	t.Run("success no metrics", func(t *testing.T) {
		componentID := int64(456)
		params := ListMetricsParams{ComponentID: componentID}

		expectedResponse := &suseobservability.BoundMetricsResponse{
			BoundMetrics: []suseobservability.BoundMetric{},
		}

		mockClient.On("GetBoundMetricsWithData", ctx, componentID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return(expectedResponse, nil).Once()

		result, _, err := tools.ListMetrics(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "No bound metrics found")
	})

	t.Run("client error", func(t *testing.T) {
		componentID := int64(789)
		params := ListMetricsParams{ComponentID: componentID}

		mockClient.On("GetBoundMetricsWithData", ctx, componentID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return(nil, errors.New("client error")).Once()

		result, _, err := tools.ListMetrics(ctx, nil, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "client error")
	})
}

func TestQueryMetric(t *testing.T) {
	mockClient := new(MockSuseObservabilityClient)
	tools := NewBaseTool(mockClient)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		query := "up"
		params := QueryMetricParams{
			Query: query,
			Start: "1h",
			End:   "now",
		}

		timestamp := time.Now().Unix()
		expectedResponse := &suseobservability.MetricQueryResponse{
			Data: suseobservability.MetricData{
				Result: []suseobservability.MetricResult{
					{
						Labels: map[string]string{
							"job": "node_exporter",
						},
						Points: []suseobservability.MetricPoint{
							{Timestamp: timestamp, Value: 1.0},
						},
					},
				},
			},
		}

		mockClient.On("QueryRangeMetric", ctx, query, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), "1m", "30s").
			Return(expectedResponse, nil).Once()

		result, _, err := tools.QueryMetric(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		output := result.Content[0].(*mcp.TextContent).Text
		assert.Contains(t, output, "node_exporter")
		assert.Contains(t, output, "1.0000")
	})

	t.Run("parsing error", func(t *testing.T) {
		params := QueryMetricParams{
			Query: "up",
			Start: "invalid",
		}

		result, _, err := tools.QueryMetric(ctx, nil, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to parse start time")
	})
}
