package publish

import (
	"fmt"

	"Kairo/internal/db/schema"

	"github.com/google/uuid"
)

func (p *PublishManager) ListPlatforms() ([]schema.PublishPlatform, error) {
	return p.publishPlatformDAL.ListPlatforms(p.ctx)
}

func (p *PublishManager) GetPlatformById(cid string) (*schema.PublishPlatform, error) {
	return p.publishPlatformDAL.GetPlatformById(p.ctx, cid)
}

// Platform management
func (p *PublishManager) CreatePlatform(name, displayName string, platformType schema.PublishPlatformType) (*schema.PublishPlatform, error) {
	platform := &schema.PublishPlatform{
		ID:          uuid.NewString(),
		Name:        name,
		DisplayName: displayName,
		Type:        platformType,
		Status:      schema.PublishPlatformStatusEnabled,
	}

	if err := p.publishPlatformDAL.SavePlatform(p.ctx, platform); err != nil {
		return nil, fmt.Errorf("failed to save platform: %v", err)
	}

	return platform, nil
}

func (p *PublishManager) UpdatePlatform(id, displayName string) (*schema.PublishPlatform, error) {
	platform, err := p.publishPlatformDAL.GetPlatformById(p.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("platform not found: %v", err)
	}

	platform.DisplayName = displayName

	if err := p.publishPlatformDAL.SavePlatform(p.ctx, platform); err != nil {
		return nil, fmt.Errorf("failed to update platform: %v", err)
	}

	return platform, nil
}

func (p *PublishManager) DeletePlatform(id string) error {
	// Check if there are any tasks associated with this platform
	tasks, err := p.publishTaskDAL.ListAllTasks(p.ctx, "", id)
	if err != nil {
		return fmt.Errorf("failed to check platform tasks: %v", err)
	}

	if len(tasks) > 0 {
		return fmt.Errorf("cannot delete platform with existing tasks")
	}

	return p.publishPlatformDAL.DeletePlatform(p.ctx, id)
}
