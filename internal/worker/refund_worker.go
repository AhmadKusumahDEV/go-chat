package worker

import (
	"net/http"

	"github.com/rabbitmq/amqp091-go"
)

type RefundWorker struct {
	httpClient *http.Client
	rabbitmq   *amqp091.Channel
}

func (r *RefundWorker) Start() error {
	return nil
}
