package rss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/db/dal"
	"Kairo/internal/db/schema"
	"Kairo/internal/utils"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

type Manager struct {
	ctx            context.Context
	db             *gorm.DB
	rssDAL         *dal.RSSDAL
	mu             sync.Mutex
	OnAutoDownload func(item schema.FeedItem, feed schema.Feed)
}

func NewManager(ctx context.Context, db *gorm.DB) *Manager {
	m := &Manager{
		ctx: ctx,
	}
	m.db = db
	if db != nil {
		m.rssDAL = dal.NewRSSDAL(db)
	}
	return m
}

func (m *Manager) getParser() *gofeed.Parser {
	fp := gofeed.NewParser()
	fp.UserAgent = "Kairo/1.0"
	// Set client with proxy if needed
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	if proxy := config.GetProxyUrl(); proxy != "" {
		if u, err := url.Parse(proxy); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(u),
			}
		}
	}
	fp.Client = client
	return fp
}

func (m *Manager) AddFeed(input schema.AddRSSFeedInput) (*schema.Feed, error) {
	if m.db == nil || m.rssDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	fp := m.getParser()
	feed, err := fp.ParseURL(input.URL)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	rssFeed := &schema.Feed{
		ID:               uuid.New().String(),
		URL:              input.URL,
		Title:            feed.Title,
		Description:      feed.Description,
		LastUpdated:      now,
		UnreadCount:      len(feed.Items),
		CustomDir:        input.CustomDir,
		DownloadLatest:   input.DownloadLatest,
		Filters:          input.Filters,
		Tags:             input.Tags,
		FilenameTemplate: input.FilenameTemplate,
		CategoryID:       input.CategoryID,
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if feed.Image != nil {
		rssFeed.Thumbnail = utils.EnsureHTTPS(feed.Image.URL)
	}
	feedItems := make([]schema.FeedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		// Filter out items without title or link
		if item.Title == "" || item.Link == "" {
			continue
		}

		pubDate := time.Now().Unix()
		if item.PublishedParsed != nil {
			pubDate = item.PublishedParsed.Unix()
		}
		itemID := uuid.New().String()

		thumbnail := ""
		if len(item.Enclosures) > 0 && (item.Enclosures[0].Type == "image/jpeg" || item.Enclosures[0].Type == "image/png") {
			thumbnail = utils.EnsureHTTPS(item.Enclosures[0].URL)
		} else if item.Image != nil {
			thumbnail = utils.EnsureHTTPS(item.Image.URL)
		}

		feedItems = append(feedItems, schema.FeedItem{
			ID:          itemID,
			FeedID:      rssFeed.ID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     pubDate,
			Status:      schema.RSSItemStatusNew,
			Thumbnail:   thumbnail,
			CreatedAt:   pubDate,
			UpdatedAt:   pubDate,
		})
	}

	err = m.db.Transaction(func(tx *gorm.DB) error {
		txDal := dal.NewRSSDAL(tx)
		if err := txDal.CreateFeed(m.ctx, rssFeed); err != nil {
			return err
		}
		return txDal.CreateFeedItems(m.ctx, feedItems)
	})
	if err != nil {
		return nil, err
	}

	go m.processAutoDownload(rssFeed.ID)

	return rssFeed, nil
}

func (m *Manager) GetFeeds() ([]schema.Feed, error) {
	if m.rssDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.rssDAL.ListFeeds(m.ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *Manager) DeleteFeed(id string) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	return m.rssDAL.DeleteFeed(m.ctx, id)
}

func (m *Manager) SetFeedEnabled(feedID string, enabled bool) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	return m.rssDAL.UpdateFeed(m.ctx, feedID, map[string]interface{}{
		"enabled":    enabled,
		"updated_at": time.Now().Unix(),
	})
}

func (m *Manager) GetFeedItems(feedID string) ([]schema.FeedItem, error) {
	if m.rssDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.rssDAL.ListFeedItems(m.ctx, feedID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *Manager) UpdateFeed(feed schema.Feed) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	return m.rssDAL.UpdateFeed(m.ctx, feed.ID, map[string]interface{}{
		"custom_dir":        feed.CustomDir,
		"download_latest":   feed.DownloadLatest,
		"filters":           feed.Filters,
		"tags":              feed.Tags,
		"filename_template": feed.FilenameTemplate,
		"category_id":       feed.CategoryID,
		"updated_at":        time.Now().Unix(),
	})
}

func (m *Manager) updateUnreadCount(feedID string) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	count, err := m.rssDAL.CountUnread(m.ctx, feedID, int(schema.RSSItemStatusNew))
	if err != nil {
		return err
	}
	return m.rssDAL.UpdateFeed(m.ctx, feedID, map[string]interface{}{
		"unread_count": count,
		"updated_at":   time.Now().Unix(),
	})
}

func (m *Manager) SetItemQueued(itemID string, queued bool) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	s := schema.RSSItemStatusRead // Default to Read if un-queued
	if queued {
		s = schema.RSSItemStatusQueued
	}
	// Don't overwrite Downloaded (4) status unless explicitly re-queuing?
	// For now, simple update.
	if err := m.rssDAL.UpdateFeedItemStatusByID(m.ctx, itemID, int(s)); err != nil {
		return err
	}
	if feedID, err := m.rssDAL.GetFeedIDByItemID(m.ctx, itemID); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) SetItemDownloadedByLink(link string, downloaded bool) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	s := schema.RSSItemStatusRead // Default to Read if not downloaded
	if downloaded {
		s = schema.RSSItemStatusDownloaded
	}
	if err := m.rssDAL.UpdateFeedItemStatusByLink(m.ctx, link, int(s)); err != nil {
		return err
	}
	if feedID, err := m.rssDAL.GetFeedIDByItemLink(m.ctx, link); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) SetItemFailedByLink(link string) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := m.rssDAL.UpdateFeedItemStatusByLink(m.ctx, link, int(schema.RSSItemStatusFailed)); err != nil {
		return err
	}
	if feedID, err := m.rssDAL.GetFeedIDByItemLink(m.ctx, link); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) RefreshFeed(feedID string) error {
	if m.db == nil || m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	url, err := m.rssDAL.GetFeedURLByID(m.ctx, feedID)
	if err != nil {
		return err
	}

	fp := m.getParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	newItems := make([]schema.FeedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		// Filter out items without title or link
		if item.Title == "" || item.Link == "" {
			continue
		}

		pubDate := time.Now().Unix()
		if item.PublishedParsed != nil {
			pubDate = item.PublishedParsed.Unix()
		}
		itemID := uuid.New().String()

		var thumbnail string
		if len(item.Enclosures) > 0 && (item.Enclosures[0].Type == "image/jpeg" || item.Enclosures[0].Type == "image/png") {
			thumbnail = utils.EnsureHTTPS(item.Enclosures[0].URL)
		} else if item.Image != nil {
			thumbnail = utils.EnsureHTTPS(item.Image.URL)
		}
		newItems = append(newItems, schema.FeedItem{
			ID:          itemID,
			FeedID:      feedID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     pubDate,
			Status:      schema.RSSItemStatusNew,
			Thumbnail:   thumbnail,
			CreatedAt:   pubDate,
			UpdatedAt:   pubDate,
		})
	}
	updates := map[string]interface{}{
		"title":        feed.Title,
		"description":  feed.Description,
		"last_updated": now,
		"updated_at":   now,
	}
	if feed.Image != nil {
		updates["thumbnail"] = utils.EnsureHTTPS(feed.Image.URL)
	}
	err = m.db.Transaction(func(tx *gorm.DB) error {
		txDal := dal.NewRSSDAL(tx)
		if err := txDal.UpdateFeed(m.ctx, feedID, updates); err != nil {
			return err
		}
		if err := txDal.CreateFeedItems(m.ctx, newItems); err != nil {
			return err
		}
		count, err := txDal.CountUnread(m.ctx, feedID, int(schema.RSSItemStatusNew))
		if err != nil {
			return err
		}
		return txDal.UpdateFeed(m.ctx, feedID, map[string]interface{}{
			"unread_count": count,
			"updated_at":   time.Now().Unix(),
		})
	})
	if err != nil {
		return err
	}

	go m.processAutoDownload(feedID)

	return nil
}

func (m *Manager) Start() {
	go func() {
		// Initial check after 1 minute (give app time to start up)
		time.Sleep(1 * time.Minute)
		m.checkFeeds()

		for {
			interval := config.GetSettings().RSSCheckInterval
			if interval <= 0 {
				interval = 60
			}
			time.Sleep(time.Duration(interval) * time.Minute)
			m.checkFeeds()
		}
	}()
}

func (m *Manager) checkFeeds() {
	feeds, err := m.GetFeeds()
	if err != nil {
		fmt.Printf("Failed to get feeds for auto-refresh: %v\n", err)
		return
	}

	for _, feed := range feeds {
		if feed.Enabled {
			// Refresh sequentially to avoid network spikes
			if err := m.RefreshFeed(feed.ID); err != nil {
				fmt.Printf("Failed to auto-refresh feed %s: %v\n", feed.Title, err)
			}
		}
	}
}

func (m *Manager) MarkItemRead(itemID string) error {
	if m.rssDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := m.rssDAL.UpdateFeedItemStatusByIDIf(m.ctx, itemID, int(schema.RSSItemStatusNew), int(schema.RSSItemStatusRead)); err != nil {
		return err
	}
	if feedID, err := m.rssDAL.GetFeedIDByItemID(m.ctx, itemID); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) processAutoDownload(feedID string) {
	if m.OnAutoDownload == nil {
		return
	}

	if m.rssDAL == nil {
		return
	}
	feed, err := m.rssDAL.GetFeedByID(m.ctx, feedID)
	if err != nil {
		return
	}

	if !feed.DownloadLatest {
		return
	}

	items, err := m.rssDAL.ListFeedItemsByStatuses(m.ctx, feedID, []int{int(schema.RSSItemStatusNew), int(schema.RSSItemStatusRead)})
	if err != nil {
		return
	}
	for _, item := range items {
		m.OnAutoDownload(item, *feed)
		_ = m.rssDAL.UpdateFeedItemStatusByID(m.ctx, item.ID, int(schema.RSSItemStatusQueued))
	}
	_ = m.updateUnreadCount(feedID)
}
