package config

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConfig struct {
	ConnCh       *amqp.Connection
	TaskCh       *amqp.Channel
	ErrMidConnCh chan *amqp.Error
}
