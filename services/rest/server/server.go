package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/ybotet/pz12-REST_vs_GraphQL/services/rest/handlers"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/repository"
)

type RESTServer struct {
	port    string
	handler *handlers.TaskHandler
	logger  *logrus.Logger
}

func NewRESTServer(port string, repo *repository.PostgresTaskRepository, logger *logrus.Logger) *RESTServer {
	return &RESTServer{
		port:    port,
		handler: handlers.NewTaskHandler(repo, logger),
		logger:  logger,
	}
}

func (s *RESTServer) setupRoutes() *mux.Router {
	r := mux.NewRouter()

	// API v1 routes
	api := r.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/tasks", s.handler.ListTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", s.handler.GetTask).Methods("GET")
	api.HandleFunc("/tasks", s.handler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", s.handler.UpdateTask).Methods("PATCH")
	api.HandleFunc("/tasks/{id}", s.handler.DeleteTask).Methods("DELETE")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	return r
}

func (s *RESTServer) Start() error {
	router := s.setupRoutes()
	addr := fmt.Sprintf(":%s", s.port)
	s.logger.WithField("port", s.port).Info("Starting REST server on http://localhost" + addr)
	return http.ListenAndServe(addr, router)
}