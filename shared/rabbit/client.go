// shared/rabbit/client.go
package rabbit

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitClient struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	URL     string
}

// NewRabbitClient crea una nueva conexión a RabbitMQ
func NewRabbitClient(url string) (*RabbitClient, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitClient{
		Conn:    conn,
		Channel: ch,
		URL:     url,
	}, nil
}

// DeclareQueue declara una cola con los parámetros especificados
func (r *RabbitClient) DeclareQueue(queueName string, durable bool) error {
	_, err := r.Channel.QueueDeclare(
		queueName, // name
		durable,   // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	return nil
}

// PublishJSON publica un mensaje JSON en la cola
func (r *RabbitClient) PublishJSON(queueName string, body []byte) error {
	err := r.Channel.Publish(
		"",       // exchange
		queueName, // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // mensaje persistente
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// Close cierra la conexión y el canal
func (r *RabbitClient) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
	log.Println("RabbitMQ connection closed")
}