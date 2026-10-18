package types

import "github.com/rabbitmq/amqp091-go"

type QueueAcknowledger interface {
	Ack() error
	Reject(requeue bool) error
}

type TaskWrapper struct {
	QueueTask amqp091.Delivery
	ScanTask  ScanTask
	Ack       QueueAcknowledger
}
type ScanTask struct {
	Target     string              `json:"Target"`
	Parameters []*CommandParameter `json:"Parameters"`
	TaskId     string              `json:"TaskId"`
}

type CommandParameter struct {
	Name      string   `json:"Name"`
	Arguments []string `json:"Arguments"`
}
