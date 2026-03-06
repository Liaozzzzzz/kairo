package publish

import (
	"Kairo/internal/db/schema"

	"github.com/google/uuid"
)

func (p *PublishManager) ListAutomations(categoryID, platformID string) ([]schema.PublishAutomation, error) {
	return p.publishAutomationDAL.ListAutomations(p.ctx, categoryID, platformID)
}

func (p *PublishManager) CreateAutomation(req schema.CreatePublishAutomationRequest) (*schema.PublishAutomation, error) {
	auto := &schema.PublishAutomation{
		ID:                  uuid.New().String(),
		CategoryID:          req.CategoryID,
		AccountID:           req.AccountID,
		TitleTemplate:       req.TitleTemplate,
		DescriptionTemplate: req.DescriptionTemplate,
		Tags:                req.Tags,
		IsEnabled:           req.IsEnabled,
		Cron:                req.Cron,
	}

	if err := p.publishAutomationDAL.CreateAutomation(p.ctx, auto); err != nil {
		return nil, err
	}
	p.reloadAutomationCronJobs()
	return auto, nil
}

func (p *PublishManager) UpdateAutomation(req schema.UpdatePublishAutomationRequest) (*schema.PublishAutomation, error) {
	auto, err := p.publishAutomationDAL.GetAutomationById(p.ctx, req.ID)
	if err != nil {
		return nil, err
	}
	auto.TitleTemplate = req.TitleTemplate
	auto.DescriptionTemplate = req.DescriptionTemplate
	auto.Tags = req.Tags
	auto.IsEnabled = req.IsEnabled
	auto.Cron = req.Cron

	if err := p.publishAutomationDAL.UpdateAutomation(p.ctx, auto); err != nil {
		return nil, err
	}
	p.reloadAutomationCronJobs()
	return auto, nil
}

func (p *PublishManager) DeleteAutomation(id string) error {
	err := p.publishAutomationDAL.DeleteAutomation(p.ctx, id)
	if err == nil {
		p.reloadAutomationCronJobs()
	}
	return err
}
