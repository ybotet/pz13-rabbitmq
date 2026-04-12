package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/events"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/models"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/rabbit"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/repository"
)

// CreateTaskRequest para el endpoint REST
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// UpdateTaskRequest para el endpoint REST
type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	rabbit    interface{}
	Done        *bool   `json:"done,omitempty"`
}

type TaskHandler struct {
	repo      *repository.PostgresTaskRepository
	logger    *logrus.Logger
	rabbit    interface{} // Temporal hasta crear el tipo RabbitClient
	queueName string
}

func NewTaskHandler(repo *repository.PostgresTaskRepository, logger *logrus.Logger, rabbit interface{}, queueName string,) *TaskHandler {
	return &TaskHandler{
		repo:   repo,
		logger: logger,
		rabbit:    rabbit,
		queueName: queueName,

	}
}

// ListTasks GET /v1/tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	h.logger.WithField("path", r.URL.Path).Info("REST request: list tasks")

	tasks, err := h.repo.GetAll(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tasks")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

// GetTask GET /v1/tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	h.logger.WithFields(logrus.Fields{
		"path": r.URL.Path,
		"id":   id,
	}).Info("REST request: get task by ID")

	task, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("id", id).Warn("Task not found")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// CreateTask POST /v1/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	h.logger.WithField("path", r.URL.Path).Info("REST request: create task")

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)
	task := &models.Task{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Done:        false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.Create(r.Context(), task); err != nil {
		h.logger.WithError(err).Error("Failed to create task")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	//  Publicar evento de forma asíncrona (best effort)
	go h.publishTaskCreatedEvent(task.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// Añadir este método al TaskHandler
func (h *TaskHandler) SetRabbitClient(rabbit interface{}, queueName string) {
	h.rabbit = rabbit
	h.queueName = queueName
	h.logger.Info("RabbitMQ client set in handler")
}

// Actualizar publishTaskCreatedEvent para usar el cliente real
func (h *TaskHandler) publishTaskCreatedEvent(taskID string) {
	if h.rabbit == nil {
		h.logger.WithField("task_id", taskID).Warn("RabbitMQ client not available, event not published")
		return
	}
	
	event := events.TaskCreatedEvent{
		Event:   "task.created",
		TaskID:  taskID,
		Ts:      time.Now().UTC(),
	}
	
	body, err := json.Marshal(event)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal event")
		return
	}
	
	// Type assertion para usar el cliente real
	if client, ok := h.rabbit.(*rabbit.RabbitClient); ok {
		if err := client.PublishJSON(h.queueName, body); err != nil {
			h.logger.WithError(err).Error("Failed to publish event")
		} else {
			h.logger.WithFields(logrus.Fields{
				"task_id": taskID,
				"queue":   h.queueName,
			}).Info("Event published successfully")
		}
	} else {
		h.logger.Error("Invalid RabbitMQ client type")
	}
}

// UpdateTask PATCH /v1/tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	h.logger.WithFields(logrus.Fields{
		"path": r.URL.Path,
		"id":   id,
	}).Info("REST request: update task")

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Obtener tarea existente
	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("id", id).Warn("Task not found for update")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	// Actualizar solo los campos proporcionados
	updated := false
	if req.Title != nil {
		existing.Title = *req.Title
		updated = true
	}
	if req.Description != nil {
		existing.Description = *req.Description
		updated = true
	}
	if req.Done != nil {
		existing.Done = *req.Done
		updated = true
	}

	if updated {
		existing.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	if err := h.repo.Update(r.Context(), existing); err != nil {
		h.logger.WithError(err).Error("Failed to update task")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(existing)
}

// DeleteTask DELETE /v1/tasks/{id}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	h.logger.WithFields(logrus.Fields{
		"path": r.URL.Path,
		"id":   id,
	}).Info("REST request: delete task")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.logger.WithError(err).WithField("id", id).Warn("Failed to delete task")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
}