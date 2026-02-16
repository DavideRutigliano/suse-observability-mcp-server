package tools

import (
	"context"
	"errors"
	"testing"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestListMonitors(t *testing.T) {
	mockClient := new(MockSuseObservabilityClient)
	tools := NewBaseTool(mockClient)
	ctx := context.Background()

	t.Run("success with monitors", func(t *testing.T) {
		componentID := int64(123)
		params := ListMonitorsParams{ComponentID: componentID}

		expectedResponse := &suseobservability.ComponentResponse{
			Node: suseobservability.ComponentNode{
				ID:   componentID,
				Name: "test-component",
				SyncedCheckStates: []map[string]interface{}{
					{
						"name":   "High CPU",
						"health": "CRITICAL",
						"data": map[string]interface{}{
							"remediationHint": "Check logs",
							"displayTimeSeries": []interface{}{
								map[string]interface{}{
									"queries": []interface{}{
										map[string]interface{}{
											"query": "avg(cpu)",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		mockClient.On("GetComponent", ctx, componentID).
			Return(expectedResponse, nil).Once()

		result, _, err := tools.ListMonitors(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		output := result.Content[0].(*mcp.TextContent).Text
		assert.Contains(t, output, "High CPU")
		assert.Contains(t, output, "CRITICAL")
		assert.Contains(t, output, "Check logs")
		assert.Contains(t, output, "avg(cpu)")
	})

	t.Run("success no monitors", func(t *testing.T) {
		componentID := int64(456)
		params := ListMonitorsParams{ComponentID: componentID}

		expectedResponse := &suseobservability.ComponentResponse{
			Node: suseobservability.ComponentNode{
				ID:                componentID,
				Name:              "test-component",
				SyncedCheckStates: []map[string]interface{}{},
			},
		}

		mockClient.On("GetComponent", ctx, componentID).
			Return(expectedResponse, nil).Once()

		result, _, err := tools.ListMonitors(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "No monitors found")
	})

	t.Run("client error", func(t *testing.T) {
		componentID := int64(789)
		params := ListMonitorsParams{ComponentID: componentID}

		mockClient.On("GetComponent", ctx, componentID).
			Return(nil, errors.New("client error")).Once()

		result, _, err := tools.ListMonitors(ctx, nil, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "client error")
	})
}
