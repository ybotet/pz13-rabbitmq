package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/ybotet/pz12-REST_vs_GraphQL/services/task/server"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/repository"
)

func main() {
	// Configuración desde variables de entorno
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "tasksuser")
	dbPass := getEnv("DB_PASSWORD", "taskspass")
	dbName := getEnv("DB_NAME", "tasksdb")
	restPort := getEnv("REST_PORT", "8082")

	rabbitURL := getEnv("RABBIT_URL", "amqp://guest:guest@localhost:5672/")
	queueName := getEnv("QUEUE_NAME", "task_events")

	// Configurar logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Conectar a PostgreSQL
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Verificar conexión
	if err := db.Ping(); err != nil {
		logger.WithError(err).Fatal("Database ping failed")
	}
	logger.Info("Connected to PostgreSQL")

	// Crear repositorio
	repo := repository.NewPostgresTaskRepository(db, logger)

	// Crear y iniciar servidor REST
	srv := server.NewRESTServer(restPort, repo, logger, rabbitURL, queueName)

	logger.WithFields(logrus.Fields{
		"rest_port":  restPort,
		"rabbit_url": rabbitURL,
		"queue_name": queueName,
	}).Info("Starting services")

	if err := srv.Start(repo); err != nil {
		logger.WithError(err).Fatal("Failed to start REST server")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}