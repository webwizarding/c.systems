package models

import "time"

type Task struct {
    ID       string    `json:"id"`
    Data     string    `json:"data" validate:"required"`
    Status   string    `json:"status"`
    Created  time.Time `json:"created"`
    Retries  int       `json:"retries"`
    Priority int       `json:"priority" validate:"required,min=1,max=3"`
}
