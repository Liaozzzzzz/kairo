package publish

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"Kairo/internal/db/dal"
	"Kairo/internal/db/schema"
	"Kairo/internal/publish/platforms"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type PublishManager struct {
	ctx                  context.Context
	db                   *gorm.DB
	publishTaskDAL       *dal.PublishTaskDAL
	publishRecordDAL     *dal.PublishRecordDAL
	publishPlatformDAL   *dal.PublishPlatformDAL
	publishAccountDAL    *dal.PublishAccountDAL
	publishAutomationDAL *dal.PublishAutomationDAL
	videoDAL             *dal.VideoDAL
	platformManager      *platforms.PlatformManager
	automationCron       *cron.Cron
	automationEntryIDs   map[string]cron.EntryID
	automationCronMu     sync.Mutex
}

func NewPublishManager(ctx context.Context, db *gorm.DB) *PublishManager {
	return &PublishManager{
		ctx:                  ctx,
		db:                   db,
		publishTaskDAL:       dal.NewPublishTaskDAL(db),
		publishRecordDAL:     dal.NewPublishRecordDAL(db),
		publishPlatformDAL:   dal.NewPublishPlatformDAL(db),
		publishAccountDAL:    dal.NewPublishAccountDAL(db),
		publishAutomationDAL: dal.NewPublishAutomationDAL(db),
		videoDAL:             dal.NewVideoDAL(db),
		platformManager:      platforms.NewPlatformManager(),
		automationCron:       cron.New(),
		automationEntryIDs:   make(map[string]cron.EntryID),
	}
}

func (p *PublishManager) StartAutoPublish() {
	p.reloadAutomationCronJobs()
	p.automationCron.Start()
	once := sync.Once{}
	once.Do(func() {
		go p.autoPublishWorker()
	})
}

// 自动发布工作协程
func (p *PublishManager) autoPublishWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.processAutoPublishTasks()
		}
	}
}

func (p *PublishManager) processAutoPublishTasks() error {
	// 获取所有待发布的任务
	tasks, err := p.publishTaskDAL.ListPendingTasks(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to list pending tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Type != schema.PublishTypeAuto {
			continue
		}
		_ = p.publishTask(&task, "auto")
	}

	return nil
}

func (p *PublishManager) reloadAutomationCronJobs() {
	if p.automationCron == nil {
		return
	}

	p.automationCronMu.Lock()
	defer p.automationCronMu.Unlock()

	for _, entryID := range p.automationEntryIDs {
		p.automationCron.Remove(entryID)
	}
	p.automationEntryIDs = make(map[string]cron.EntryID)

	automations, err := p.publishAutomationDAL.ListAutomations(p.ctx, "all", "all")
	if err != nil {
		fmt.Printf("Failed to load automations for cron: %v\n", err)
		return
	}

	for _, auto := range automations {
		if !auto.IsEnabled || strings.TrimSpace(auto.Cron) == "" {
			continue
		}
		autoID := auto.ID
		spec := strings.TrimSpace(auto.Cron)
		entryID, err := p.automationCron.AddFunc(spec, func() {
			p.runAutomationDetection(autoID)
		})
		if err != nil {
			fmt.Printf("Failed to register cron for automation %s (%s): %v\n", autoID, spec, err)
			continue
		}
		p.automationEntryIDs[autoID] = entryID
	}
}

func (p *PublishManager) runAutomationDetection(automationID string) {
	auto, err := p.publishAutomationDAL.GetAutomationById(p.ctx, automationID)
	if err != nil {
		return
	}
	if !auto.IsEnabled || strings.TrimSpace(auto.Cron) == "" {
		return
	}

	account, err := p.publishAccountDAL.GetAccountById(p.ctx, auto.AccountID)
	if err != nil || account == nil || !account.IsEnabled {
		return
	}

	interval, err := time.ParseDuration(account.PublishInterval)
	if err != nil {
		log.Printf("Failed to parse interval for automation %s: %v, defaulting to 1 hour\n", automationID, err)
		interval = time.Hour
	}

	// Calculate base time for scheduling
	latestTask, _ := p.publishTaskDAL.GetLatestScheduledTask(p.ctx, auto.AccountID)
	var baseTime time.Time
	if latestTask != nil {
		baseTime = time.UnixMilli(latestTask.ScheduledAt)
	} else {
		baseTime = time.Now()
	}
	if baseTime.Before(time.Now()) {
		baseTime = time.Now()
	}

	highlights, err := p.videoDAL.ListHighlightsByCategoryID(p.ctx, auto.CategoryID)
	if err != nil {
		return
	}

	for _, highlight := range highlights {
		title := replaceTemplate(auto.TitleTemplate, &highlight)
		description := replaceTemplate(auto.DescriptionTemplate, &highlight)
		baseTime = baseTime.Add(interval)
		scheduledAt := baseTime.UnixMilli()

		_, _ = p.CreateTask(schema.CreatePublishTaskRequest{
			HighlightID: highlight.ID,
			AccountID:   auto.AccountID,
			PublishType: schema.PublishTypeAuto,
			ScheduledAt: scheduledAt,
			Title:       title,
			Description: description,
			Tags:        auto.Tags,
		})
	}
}

func replaceTemplate(template string, highlight *schema.VideoHighlight) string {
	s := template
	s = strings.ReplaceAll(s, "{{title}}", highlight.Title)
	s = strings.ReplaceAll(s, "{{description}}", highlight.Description)
	s = strings.ReplaceAll(s, "{{date}}", time.Unix(highlight.CreatedAt, 0).Format("2006-01-02"))
	return s
}
