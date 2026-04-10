package graph

import (
	"github.com/sirupsen/logrus"
	"github.com/ybotet/pz12-REST_vs_GraphQL/shared/repository"
)

type Resolver struct {
	TaskRepo *repository.PostgresTaskRepository
	Logger   *logrus.Logger
}

func NewResolver(taskRepo *repository.PostgresTaskRepository, log *logrus.Logger) *Resolver {
	return &Resolver{
		TaskRepo: taskRepo,
		Logger:   log,
	}
}