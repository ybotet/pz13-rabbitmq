// services/worker/internal/consumer/consumer.go
package consumer

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"

	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/events"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/rabbit"
)

type TaskEventConsumer struct {
	client    *rabbit.RabbitClient
	queueName string
	logger    *logrus.Logger
}

func NewTaskEventConsumer(rabbitURL, queueName string, logger *logrus.Logger) (*TaskEventConsumer, error) {
	// Conectar a RabbitMQ
	client, err := rabbit.NewRabbitClient(rabbitURL)
	if err != nil {
		return nil, err
	}

	// Declarar la cola (durable)
	if err := client.DeclareQueue(queueName, true); err != nil {
		client.Close()
		return nil, err
	}

	// Configurar prefetch (1 mensaje por vez)
	if err := client.Channel.Qos(1, 0, false); err != nil {
		client.Close()
		return nil, err
	}

	return &TaskEventConsumer{
		client:    client,
		queueName: queueName,
		logger:    logger,
	}, nil
}

func (c *TaskEventConsumer) Start() error {
	// Suscribirse a la cola
	messages, err := c.client.Channel.Consume(
		c.queueName, // queue
		"worker",    // consumer
		false,       // auto-ack (false para control manual)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	c.logger.WithField("queue", c.queueName).Info("Worker started, waiting for messages...")

	// Procesar mensajes
	for msg := range messages {
		c.processMessage(msg)
	}

	return nil
}

func (c *TaskEventConsumer) processMessage(msg amqp.Delivery) {
	c.logger.WithField("body", string(msg.Body)).Info(" Received message")

	// Parsear el mensaje
	var event events.TaskCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.WithError(err).Error("Failed to parse message")
		msg.Nack(false, false) // Rechazar, no reencolar
		return
	}

	// Procesar el evento
	c.logger.WithFields(logrus.Fields{
		"event":   event.Event,
		"task_id": event.TaskID,
		"ts":      event.Ts,
	}).Info(" Processing task.created event")

	// TODO: Aquí iría la lógica de negocio (ej: enviar email, actualizar estadísticas, etc.)

	// Confirmar procesamiento exitoso
	if err := msg.Ack(false); err != nil {
		c.logger.WithError(err).Error("Failed to ack message")
	} else {
		c.logger.WithField("task_id", event.TaskID).Info(" Message acknowledged")
	}
}

func (c *TaskEventConsumer) Close() {
	c.client.Close()
}