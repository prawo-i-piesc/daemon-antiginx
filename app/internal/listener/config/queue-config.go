package config

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type AMQPConsumer interface {
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Close() error
}

type RabbitConfig struct {
	ConnCh          *amqp.Connection
	ConsumerChannel AMQPConsumer
	ErrMidConnCh    chan *amqp.Error
	QueueName       string
}
