package router

import "github.com/rabbitmq/amqp091-go"

type RabbitAcknowledger struct {
	Delivery amqp091.Delivery
}

func (r *RabbitAcknowledger) Ack() error {
	return r.Delivery.Ack(false)
}

func (r *RabbitAcknowledger) Reject(requeue bool) error {
	return r.Delivery.Reject(requeue)
}
