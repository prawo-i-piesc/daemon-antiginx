package types

import "github.com/rabbitmq/amqp091-go"

type TaskWrapper struct {
	QueueTask amqp091.Delivery
	ScanTask  ScanTask
}
type ScanTask struct {
	Target     string              `json:"Target"`
	Parameters []*CommandParameter `json:"Parameters"`
}

type CommandParameter struct {
	Name      string   `json:"Name"`
	Arguments []string `json:"Arguments"`
}
