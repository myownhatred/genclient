package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTasukete_MarshallJSON(t *testing.T) {
	fixedUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	fixedTime := time.Date(2024, 2, 14, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		task     Tasukete
		expected string
		wantErr  bool
	}{
		{
			name: "basic task",
			task: Tasukete{
				UUID:      fixedUUID,
				Type:      TTI,
				Prompt:    "generate a cat",
				Model:     1,
				Metadata:  map[string]any{"seed": 12345},
				CreatedAt: fixedTime,
				Status:    StatusPending,
			},
			expected: `{"uuid":"550e8400-e29b-41d4-a716-446655440000","type":"TTI","prompt":"generate a cat","model":1,"metadata":{"seed":12345},"created_at":"2024-02-14T12:00:00Z","status":"PENDING"}`,
			wantErr:  false,
		},
		{
			name: "empty metadata",
			task: Tasukete{
				UUID:      fixedUUID,
				Type:      LLM,
				Prompt:    "test prompt",
				Model:     2,
				Metadata:  nil,
				CreatedAt: fixedTime,
				Status:    StatusProcessing,
			},
			expected: `{"uuid":"550e8400-e29b-41d4-a716-446655440000","type":"LLM","prompt":"test prompt","model":2,"metadata":null,"created_at":"2024-02-14T12:00:00Z","status":"PROCESSING"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(&tt.task)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestTasukete_UnmarshallJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    Tasukete
		wantErr bool
	}{
		{
			name: "valid json",
			json: `{"uuid":"550e8400-e29b-41d4-a716-446655440000","type":"TTI","prompt":"generate a cat","model":1,"metadata":{"seed":12345},"created_at":"2024-02-14T12:00:00Z","status":"PENDING"}`,
			want: Tasukete{
				UUID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Type:      TTI,
				Prompt:    "generate a cat",
				Model:     1,
				Metadata:  map[string]any{"seed": float64(12345)}, // JSON numbers are decoded as float64
				CreatedAt: time.Date(2024, 2, 14, 12, 0, 0, 0, time.UTC),
				Status:    StatusPending,
			},
			wantErr: false,
		},
		{
			name:    "invalid uuid",
			json:    `{"uuid":"invalid-uuid","type":"TTI","prompt":"test","model":1}`,
			wantErr: true,
		},
		{
			name:    "invalid type",
			json:    `{"uuid":"550e8400-e29b-41d4-a716-446655440000","type":"INVALID","prompt":"test","model":1}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var task Tasukete
			err := json.Unmarshal([]byte(tt.json), &task)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, task)
		})
	}
}
