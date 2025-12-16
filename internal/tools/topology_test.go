package tools

import (
	"context"
	"errors"
	"testing"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetComponents(t *testing.T) {
	mockClient := new(MockSuseObservabilityClient)
	tools := NewBaseTool(mockClient)
	ctx := context.Background()

	t.Run("success with simple filters", func(t *testing.T) {
		params := GetComponentsParams{
			Names: "service-a,service-b",
			Types: "service",
		}

		expectedResponse := []suseobservability.ViewComponent{
			{ID: 1, Name: "service-a"},
			{ID: 2, Name: "service-b"},
		}

		// Expected STQL query
		expectedQuery := "name IN (\"service-a\", \"service-b\") AND type IN (\"service\")"

		mockClient.On("SnapShotTopologyQuery", ctx, expectedQuery).
			Return(expectedResponse, nil).Once()

		result, _, err := tools.GetComponents(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		output := result.Content[0].(*mcp.TextContent).Text
		assert.Contains(t, output, "service-a")
		assert.Contains(t, output, "service-b")
	})

	t.Run("success with withNeighborsOf", func(t *testing.T) {
		params := GetComponentsParams{
			Names:                  "db-master",
			WithNeighbors:          true,
			WithNeighborsLevels:    "2",
			WithNeighborsDirection: "down",
		}

		// Expected STQL query
		expectedQuery := "name IN (\"db-master\") OR withNeighborsOf(components = (name IN (\"db-master\")), levels = \"2\", direction = \"down\")"

		mockClient.On("SnapShotTopologyQuery", ctx, expectedQuery).
			Return([]suseobservability.ViewComponent{{ID: 1, Name: "db-master"}}, nil).Once()

		result, _, err := tools.GetComponents(ctx, nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("error missing filters", func(t *testing.T) {
		params := GetComponentsParams{}

		result, _, err := tools.GetComponents(ctx, nil, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "at least one filter")
	})

	t.Run("client error", func(t *testing.T) {
		params := GetComponentsParams{Names: "foo"}

		mockClient.On("SnapShotTopologyQuery", ctx, mock.AnythingOfType("string")).
			Return(nil, errors.New("client error")).Once()

		result, _, err := tools.GetComponents(ctx, nil, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "client error")
	})
}
