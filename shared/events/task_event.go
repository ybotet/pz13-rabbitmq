package events

import "time"

type TaskCreatedEvent struct {
    Event   string    `json:"event"`   // "task.created"
    TaskID  string    `json:"task_id"`
    Ts      time.Time `json:"ts"`
}