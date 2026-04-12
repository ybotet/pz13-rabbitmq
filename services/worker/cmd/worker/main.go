// services/worker/cmd/worker/main.go
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/ybotet/pz12-REST_vs_GraphQL/services/worker/internal/consumer"
)

func main() {
	// Configurar logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.InfoLevel)

	// Variables de entorno
	rabbitURL := getEnv("RABBIT_URL", "amqp://guest:guest@localhost:5672/")
	queueName := getEnv("QUEUE_NAME", "task_events")

	logger.WithFields(logrus.Fields{
		"rabbit_url": rabbitURL,
		"queue_name": queueName,
	}).Info("Starting Task Event Worker")

	// Crear consumidor
	consumer, err := consumer.NewTaskEventConsumer(rabbitURL, queueName, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create consumer")
	}
	defer consumer.Close()

	// Manejar señales para cierre graceful
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar consumidor en goroutine
	go func() {
		if err := consumer.Start(); err != nil {
			logger.WithError(err).Fatal("Consumer error")
		}
	}()

	logger.Info("Worker is running. Press Ctrl+C to stop.")
	<-sigChan
	logger.Info("Shutting down worker...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}