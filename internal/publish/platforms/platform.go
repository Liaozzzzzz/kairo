package platforms

import (
	"context"
	"fmt"

	"Kairo/internal/db/schema"
	"Kairo/internal/publish/platforms/douyin"
	"Kairo/internal/publish/platforms/openclaw"
	"Kairo/internal/publish/platforms/xiaohongshu"
)

type PlatformAPI interface {
	UploadVideo(ctx context.Context, title, description string, tags []string, videoPath, accountCookiePath string) (string, error)
	ValidateAccount(ctx context.Context, cookiePath string) error
	GetPlatformName() string
	ValidateConfig() error
}

type PlatformManager struct {
	platforms map[string]PlatformAPI
}

func NewPlatformManager() *PlatformManager {
	return &PlatformManager{
		platforms: make(map[string]PlatformAPI),
	}
}

func (pm *PlatformManager) GetPlatform(name string) (PlatformAPI, error) {
	api, exists := pm.platforms[name]
	if !exists {
		return nil, fmt.Errorf("platform %s not found", name)
	}
	return api, nil
}

func (pm *PlatformManager) GetPlatforms() []string {
	platforms := make([]string, 0, len(pm.platforms))
	for name := range pm.platforms {
		platforms = append(platforms, name)
	}
	return platforms
}

// 创建平台API实例
func (pm *PlatformManager) CreatePlatformAPI(platform schema.PublishPlatform) (PlatformAPI, error) {
	if platform.Type == schema.PublishPlatformTypeOpenClaw {
		return openclaw.New(platform), nil
	}

	defaultConfig := "{}"
	switch platform.Name {
	case string(schema.PlatformXiaohongshu):
		return xiaohongshu.New(defaultConfig)
	case string(schema.PlatformDouyin):
		return douyin.New(defaultConfig)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform.Name)
	}
}
