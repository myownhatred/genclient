package main

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Type int

const (
	TTI   Type = iota // Text to Image
	LLM               // Language Model
	Recon             // Recognition
)

func (t Type) String() string {
	switch t {
	case TTI:
		return "TTI"
	case LLM:
		return "LLM"
	case Recon:
		return "RECON"
	default:
		return "UNKNOWN"
	}
}

type TaskStatus int

const (
	StatusPending TaskStatus = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
)

type Taskete struct {
	UUID      uuid.UUID      `json:"uuid"`
	Type      Type           `json:"type"`
	Prompt    string         `json:"prompt"`
	Model     int            `json:"model"`
	Metadata  map[string]any `json:"meta"`
	CreatedAt time.Time      `json:"created_at"`
	Status    TaskStatus     `json:"status"`
}

// constructor
func NewTaskete(taskType Type, prompt string, model int) *Taskete {
	return &Taskete{
		UUID:      uuid.New(),
		Type:      taskType,
		Prompt:    prompt,
		Model:     model,
		Metadata:  make(map[string]any),
		CreatedAt: time.Now(),
		Status:    StatusPending,
	}
}

// Methods for task management
func (t *Taskete) UpdateStatus(status TaskStatus) {
	t.Status = status
}

func (t *Taskete) AddMetadata(key string, value any) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]any)
	}
	t.Metadata[key] = value
}

func (t *Taskete) GetMetadata(key string) (any, bool) {
	if t.Metadata == nil {
		return nil, false
	}
	val, exists := t.Metadata[key]
	return val, exists
}

func (t *Taskete) Validate() error {
	if t.UUID == uuid.Nil {
		return errors.New("invalid UUID")
	}
	return nil
}

func (t *Taskete) MarshalJSON() ([]byte, error) {
	type Alias Taskete // avoid recursive JSON marshaling
	return json.Marshal(&struct {
		*Alias
		UUID string `json:"uuid"`
	}{
		Alias: (*Alias)(t),
		UUID:  t.UUID.String(),
	})
}

func (t *Taskete) UnmarshalJSON(data []byte) error {
	type Alias Taskete
	aux := &struct {
		*Alias
		UUID string `json:"uuid"`
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if parsedUUID, err := uuid.Parse(aux.UUID); err != nil {
		return err
	} else {
		t.UUID = parsedUUID
	}
	return nil
}
