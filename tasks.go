package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

// type marshalling
func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Type) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Convert string back to Type
	switch s {
	case "TTI":
		*t = TTI
	case "LLM":
		*t = LLM
	case "RECON":
		*t = Recon
	default:
		return fmt.Errorf("unknown type: %s", s)
	}
	return nil
}

type TaskStatus int

const (
	StatusPending TaskStatus = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
)

func (ts TaskStatus) String() string {
	switch ts {
	case StatusPending:
		return "PENDING"
	case StatusProcessing:
		return "PROCESSING"
	case StatusCompleted:
		return "COMPLETED"
	case StatusFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

func (ts TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(ts.String())
}

func (ts *TaskStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Convert string back to Type
	switch s {
	case "PENDING":
		*ts = StatusPending
	case "PROCESSING":
		*ts = StatusProcessing
	case "COMPLETED":
		*ts = StatusCompleted
	case "FAILED":
		*ts = StatusFailed
	default:
		return fmt.Errorf("unknown task status: %s", s)
	}
	return nil
}

type Tasukete struct {
	UUID      uuid.UUID      `json:"uuid"`
	Type      Type           `json:"type"`
	Prompt    string         `json:"prompt"`
	Model     int            `json:"model"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	Status    TaskStatus     `json:"status"`
}

// constructor
func NewTasukete(taskType Type, prompt string, model int) *Tasukete {
	return &Tasukete{
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
func (t *Tasukete) UpdateStatus(status TaskStatus) {
	t.Status = status
}

func (t *Tasukete) AddMetadata(key string, value any) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]any)
	}
	t.Metadata[key] = value
}

func (t *Tasukete) GetMetadata(key string) (any, bool) {
	if t.Metadata == nil {
		return nil, false
	}
	val, exists := t.Metadata[key]
	return val, exists
}

func (t *Tasukete) Validate() error {
	if t.UUID == uuid.Nil {
		return errors.New("invalid UUID")
	}
	return nil
}

func (t *Tasukete) MarshalJSON() ([]byte, error) {
	type Alias Tasukete // avoid recursive JSON marshaling
	return json.Marshal(&struct {
		*Alias
		UUID string `json:"uuid"`
	}{
		Alias: (*Alias)(t),
		UUID:  t.UUID.String(),
	})
}

func (t *Tasukete) UnmarshalJSON(data []byte) error {
	type Alias Tasukete
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
