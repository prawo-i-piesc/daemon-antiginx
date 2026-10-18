package router

import (
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

var routerTests = []struct {
	name          string
	taskBody      []byte
	expectError   bool
	expectedError *types.ScanError
}{
	{
		name:        "Happy path, successfully parse and route task",
		taskBody:    []byte(`{"TaskId": "task-123", "Target": "example.com"}`),
		expectError: false,
	},
	{
		name:        "Should return error on invalid JSON",
		taskBody:    []byte(`{"TaskId": "task-123", "Target": "example.com"`),
		expectError: true,
		expectedError: &types.ScanError{
			Message: "Scan task parsing error",
			Code:    500,
		},
	},
}

func TestTaskRouter_Route(t *testing.T) {
	for _, tt := range routerTests {
		t.Run(tt.name, func(t *testing.T) {
			taskChan := make(chan types.TaskWrapper, 1)
			router := &TaskRouter{
				ScanTaskChannel: taskChan,
			}

			delivery := amqp091.Delivery{
				Body: tt.taskBody,
			}

			err := router.Route(delivery)

			if tt.expectError {
				require.Error(t, err)

				var scanErr *types.ScanError
				require.ErrorAs(t, err, &scanErr)
				require.Equal(t, tt.expectedError.Message, scanErr.Message)
				require.Equal(t, tt.expectedError.Code, scanErr.Code)
				require.NotNil(t, scanErr.Err)

				require.Len(t, taskChan, 0)
			} else {
				require.NoError(t, err)
				require.Len(t, taskChan, 1)

				wrapper := <-taskChan
				require.Equal(t, "task-123", wrapper.ScanTask.TaskId)
				require.Equal(t, tt.taskBody, wrapper.QueueTask.Body)
			}
		})
	}
}
