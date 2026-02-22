package ai

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var ErrAIDisabled = errors.New("ai is disabled")
var ErrWhisperDisabled = errors.New("whisper is disabled")

type Manager struct {
	client *http.Client
	ctx    context.Context
}

func NewManager(ctx context.Context) *Manager {
	return &Manager{
		ctx: ctx,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}
