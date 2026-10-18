package listener

import (
	"context"
	"testing"

	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/common/types"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/listener/config"
	"github.com/prawo-i-piesc/daemon-antiginx/app/internal/router"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAMQPChannel struct {
	mock.Mock
}

func (m *MockAMQPChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	argsCall := m.Called(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	return argsCall.Get(0).(<-chan amqp.Delivery), argsCall.Error(1)
}

func (m *MockAMQPChannel) Close() error {
	return m.Called().Error(0)
}

type MockAcknowledger struct {
	mock.Mock
}

func (m *MockAcknowledger) Ack(tag uint64, multiple bool) error           { return nil }
func (m *MockAcknowledger) Nack(tag uint64, multiple, requeue bool) error { return nil }
func (m *MockAcknowledger) Reject(tag uint64, requeue bool) error {
	return m.Called(tag, requeue).Error(0)
}

func TestListen_HappyPath_SuccessfullyRoutesMessageAndDelegatesAck(t *testing.T) {
	taskDeliveryChan := make(chan amqp.Delivery, 1)

	mockCh := new(MockAMQPChannel)
	mockCh.On("Consume", "test-queue", "", false, false, false, false, mock.Anything).
		Return((<-chan amqp.Delivery)(taskDeliveryChan), nil)

	mockAck := new(MockAcknowledger)

	taskDeliveryChan <- amqp.Delivery{
		DeliveryTag:  1,
		MessageId:    "msg-777",
		Body:         []byte(`{"TaskId": "task-777", "Target": "example.com"}`),
		Acknowledger: mockAck,
	}
	close(taskDeliveryChan)

	taskChan := make(chan types.TaskWrapper, 1)
	realRouter := router.TaskRouter{
		ScanTaskChannel: taskChan,
	}

	cfg := &config.RabbitConfig{
		ConsumerChannel: mockCh,
		QueueName:       "test-queue",
		ErrMidConnCh:    make(chan *amqp.Error),
	}

	err := Listen(cfg, context.Background(), realRouter)

	require.NoError(t, err)
	require.Len(t, taskChan, 1)

	wrapper := <-taskChan
	require.Equal(t, "task-777", wrapper.ScanTask.TaskId)

	mockCh.AssertExpectations(t)
	mockAck.AssertExpectations(t)
}

func TestListen_ContextCancel(t *testing.T) {
	mockCh := new(MockAMQPChannel)
	mockCh.On("Consume", "test-queue", "", false, false, false, false, mock.Anything).
		Return(make(<-chan amqp.Delivery), nil)
	mockCh.On("Close").Return(nil).Once()

	cfg := &config.RabbitConfig{
		ConsumerChannel: mockCh,
		QueueName:       "test-queue",
		ErrMidConnCh:    make(chan *amqp.Error),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Listen(cfg, ctx, router.TaskRouter{})
	require.NoError(t, err)

	mockCh.AssertExpectations(t)
}

func TestListen_MidConnectionError(t *testing.T) {
	mockCh := new(MockAMQPChannel)
	mockCh.On("Consume", "test-queue", "", false, false, false, false, mock.Anything).
		Return(make(<-chan amqp.Delivery), nil)
	mockCh.On("Close").Return(nil).Once()

	errChan := make(chan *amqp.Error, 1)
	amqpError := &amqp.Error{Code: 500, Reason: "Connection dropped"}
	errChan <- amqpError

	cfg := &config.RabbitConfig{
		ConsumerChannel: mockCh,
		QueueName:       "test-queue",
		ErrMidConnCh:    errChan,
	}

	err := Listen(cfg, context.Background(), router.TaskRouter{})

	require.Error(t, err)
	var scanErr *types.ScanError
	require.ErrorAs(t, err, &scanErr)
	require.Equal(t, 500, scanErr.Code)
	require.Equal(t, amqpError, scanErr.Err)

	mockCh.AssertExpectations(t)
}

func TestListen_MessageRoutingError_TriggersReject(t *testing.T) {
	taskDeliveryChan := make(chan amqp.Delivery, 1)

	mockCh := new(MockAMQPChannel)
	mockCh.On("Consume", "test-queue", "", false, false, false, false, mock.Anything).
		Return((<-chan amqp.Delivery)(taskDeliveryChan), nil)

	mockAck := new(MockAcknowledger)
	mockAck.On("Reject", uint64(1), false).Return(nil).Once()

	taskDeliveryChan <- amqp.Delivery{
		DeliveryTag:  1,
		MessageId:    "msg-123",
		Body:         []byte(`{invalid-json`),
		Acknowledger: mockAck,
	}
	close(taskDeliveryChan)

	realRouter := router.TaskRouter{
		ScanTaskChannel: make(chan types.TaskWrapper, 1),
	}

	cfg := &config.RabbitConfig{
		ConsumerChannel: mockCh,
		QueueName:       "test-queue",
		ErrMidConnCh:    make(chan *amqp.Error),
	}

	err := Listen(cfg, context.Background(), realRouter)

	require.NoError(t, err)
	mockAck.AssertExpectations(t)
}
