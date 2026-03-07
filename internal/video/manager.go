package video

import (
	"context"
	"fmt"
	"log"
	"time"

	"Kairo/internal/ai"
	"Kairo/internal/db/dal"
	"Kairo/internal/db/schema"
	"Kairo/internal/deps"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	ctx           context.Context
	db            *gorm.DB
	aiService     *ai.Manager
	deps          *deps.Manager
	subtitleQueue chan SubtitleTask
	analysisQueue chan string
	videoDAL      *dal.VideoDAL
	subtitleDAL   *dal.VideoSubtitleDAL
	categoryDAL   *dal.CategoryDAL
	highlightDAL  *dal.VideoHighlightDAL
}

func NewManager(ctx context.Context, db *gorm.DB, d *deps.Manager) *Manager {
	m := &Manager{
		ctx:       ctx,
		db:        db,
		aiService: ai.NewManager(ctx),
		deps:      d,
	}
	if db != nil {
		m.videoDAL = dal.NewVideoDAL(db)
		m.subtitleDAL = dal.NewVideoSubtitleDAL(db)
		m.categoryDAL = dal.NewCategoryDAL(db)
		m.highlightDAL = dal.NewVideoHighlightDAL(db)
	}
	m.InitSubtitleQueue()
	m.InitAnalyzeQueue()
	return m
}

func (m *Manager) CreateFromTask(t *schema.Task) error {
	if t.FilePath == "" {
		return fmt.Errorf("task has no file path")
	}
	exists, err := m.HasVideoForTask(t.ID, t.FilePath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	v := &schema.Video{
		ID:         uuid.New().String(),
		TaskID:     t.ID,
		Title:      t.Title,
		URL:        t.URL,
		FilePath:   t.FilePath,
		Thumbnail:  t.Thumbnail,
		Size:       t.TotalBytes,
		Format:     t.Format,
		Resolution: t.Quality,
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
		CategoryID: t.CategoryID,
		Status:     "none",
	}

	if v.Duration <= 0 && v.FilePath != "" {
		log.Printf("[CreateFromTask] get duration from file: %s", v.FilePath)
		if duration, err := m.getDurationFromFile(v.FilePath); err == nil {
			v.Duration = duration
		}
		log.Printf("[CreateFromTask] duration: %f", v.Duration)
	}

	err = m.videoDAL.Save(m.ctx, v)
	if err == nil {
		go m.FetchSubtitles(v.ID)
	}
	return err
}
