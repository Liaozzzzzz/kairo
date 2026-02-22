package deps

import (
	"context"
	"errors"
)

type AssetProvider func(name string) ([]byte, error)

type Manager struct {
	Ctx           context.Context
	AssetProvider AssetProvider
	YtDlpPath     string
	FFmpegPath    string
}

func NewManager(ctx context.Context, ap AssetProvider) *Manager {
	return &Manager{
		Ctx:           ctx,
		AssetProvider: ap,
	}
}

func (m *Manager) GetYtDlpPath() (string, error) {
	m.EnsureYtDlp()
	if m.YtDlpPath == "" {
		return "", errors.New("yt-dlp not found")
	}
	return m.YtDlpPath, nil
}

func (m *Manager) GetFFmpegPath() (string, error) {
	m.EnsureFFmpeg()
	if m.FFmpegPath == "" {
		return "", errors.New("ffmpeg not found")
	}
	return m.FFmpegPath, nil
}
