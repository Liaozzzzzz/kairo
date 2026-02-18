package rss

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/models"
	"Kairo/internal/utils"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	_ "modernc.org/sqlite"
)

type Manager struct {
	ctx            context.Context
	db             *sql.DB
	mu             sync.Mutex
	OnAutoDownload func(item models.RSSItem, feed models.RSSFeed)
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		ctx: ctx,
	}
	m.initDB()
	return m
}

func (m *Manager) initDB() {
	appDir, err := config.GetAppConfigDir()
	if err != nil {
		fmt.Printf("Failed to get app config dir: %v\n", err)
		return
	}
	dbPath := filepath.Join(appDir, "rss.db")
	// Enable WAL mode and set busy timeout to handle concurrent writes better
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		fmt.Printf("Failed to open RSS database: %v\n", err)
		return
	}
	m.db = db

	// Create tables if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS feeds (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			thumbnail TEXT,
			last_updated INTEGER,
			unread_count INTEGER DEFAULT 0,
			custom_dir TEXT DEFAULT '',
			download_latest INTEGER DEFAULT 0,
			filters TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			filename_template TEXT DEFAULT '',
			enabled INTEGER DEFAULT 1
		);
		CREATE TABLE IF NOT EXISTS items (
			id TEXT PRIMARY KEY,
			feed_id TEXT NOT NULL,
			title TEXT NOT NULL,
			link TEXT NOT NULL,
			description TEXT,
			pub_date INTEGER,
			status INTEGER DEFAULT 0,
			thumbnail TEXT,
			FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_items_feed_id ON items(feed_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_items_link ON items(feed_id, link);
	`)
	if err != nil {
		fmt.Printf("Failed to create tables: %v\n", err)
	}
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

func (m *Manager) AddFeed(input models.AddRSSFeedInput) (*models.RSSFeed, error) {
	fp := m.getParser()
	feed, err := fp.ParseURL(input.URL)
	if err != nil {
		return nil, err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rssFeed := &models.RSSFeed{
		ID:               uuid.New().String(),
		URL:              input.URL,
		Title:            feed.Title,
		Description:      feed.Description,
		LastUpdated:      time.Now().Unix(),
		UnreadCount:      len(feed.Items),
		CustomDir:        input.CustomDir,
		DownloadLatest:   input.DownloadLatest,
		Filters:          input.Filters,
		Tags:             input.Tags,
		FilenameTemplate: input.FilenameTemplate,
		Enabled:          true,
	}

	if feed.Image != nil {
		rssFeed.Thumbnail = utils.EnsureHTTPS(feed.Image.URL)
	}

	_, err = tx.Exec(`INSERT INTO feeds (id, url, title, description, thumbnail, last_updated, unread_count, custom_dir, download_latest, filters, tags, filename_template, enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rssFeed.ID, rssFeed.URL, rssFeed.Title, rssFeed.Description, rssFeed.Thumbnail, rssFeed.LastUpdated, rssFeed.UnreadCount,
		rssFeed.CustomDir, rssFeed.DownloadLatest, rssFeed.Filters, rssFeed.Tags, rssFeed.FilenameTemplate, rssFeed.Enabled)
	if err != nil {
		return nil, err
	}

	for _, item := range feed.Items {
		pubDate := time.Now().Unix()
		if item.PublishedParsed != nil {
			pubDate = item.PublishedParsed.Unix()
		}
		itemID := uuid.New().String()

		rssItem := models.RSSItem{
			ID:          itemID,
			FeedID:      rssFeed.ID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     pubDate,
			Status:      models.RSSItemStatusNew,
		}
		// Extract thumbnail if possible (from extensions or enclosure)
		if len(item.Enclosures) > 0 && (item.Enclosures[0].Type == "image/jpeg" || item.Enclosures[0].Type == "image/png") {
			rssItem.Thumbnail = utils.EnsureHTTPS(item.Enclosures[0].URL)
		} else if item.Image != nil {
			rssItem.Thumbnail = utils.EnsureHTTPS(item.Image.URL)
		}

		_, err = tx.Exec(`INSERT OR IGNORE INTO items (id, feed_id, title, link, description, pub_date, status, thumbnail) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			rssItem.ID, rssItem.FeedID, rssItem.Title, rssItem.Link, rssItem.Description, rssItem.PubDate, models.RSSItemStatusNew, rssItem.Thumbnail)
		if err != nil {
			// Log error but continue
			fmt.Printf("Failed to insert item %s: %v\n", rssItem.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	go m.processAutoDownload(rssFeed.ID)

	return rssFeed, nil
}

func (m *Manager) GetFeeds() ([]models.RSSFeed, error) {
	rows, err := m.db.Query(`SELECT id, url, title, description, thumbnail, last_updated, unread_count, custom_dir, download_latest, filters, tags, filename_template, enabled FROM feeds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.RSSFeed
	for rows.Next() {
		var feed models.RSSFeed
		var customDir, filters, tags, filenameTemplate sql.NullString
		var downloadLatest, enabled sql.NullInt32
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.Thumbnail, &feed.LastUpdated, &feed.UnreadCount, &customDir, &downloadLatest, &filters, &tags, &filenameTemplate, &enabled); err != nil {
			continue
		}
		feed.CustomDir = customDir.String
		feed.Filters = filters.String
		feed.Tags = tags.String
		feed.FilenameTemplate = filenameTemplate.String
		feed.DownloadLatest = downloadLatest.Int32 == 1
		// Default to true if null (for old records if migration fails silently or default not applied)
		if enabled.Valid {
			feed.Enabled = enabled.Int32 == 1
		} else {
			feed.Enabled = true
		}
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

func (m *Manager) DeleteFeed(id string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Explicitly delete items first to ensure they are removed regardless of foreign key settings
	if _, err := tx.Exec(`DELETE FROM items WHERE feed_id = ?`, id); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM feeds WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *Manager) SetFeedEnabled(feedID string, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := m.db.Exec(`UPDATE feeds SET enabled = ? WHERE id = ?`, val, feedID)
	return err
}

func (m *Manager) GetFeedItems(feedID string) ([]models.RSSItem, error) {
	rows, err := m.db.Query(`SELECT id, feed_id, title, link, description, pub_date, status, thumbnail FROM items WHERE feed_id = ? ORDER BY pub_date DESC`, feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.RSSItem
	for rows.Next() {
		var item models.RSSItem
		if err := rows.Scan(&item.ID, &item.FeedID, &item.Title, &item.Link, &item.Description, &item.PubDate, &item.Status, &item.Thumbnail); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (m *Manager) UpdateFeed(feed models.RSSFeed) error {
	_, err := m.db.Exec(`UPDATE feeds SET custom_dir = ?, download_latest = ?, filters = ?, tags = ?, filename_template = ? WHERE id = ?`,
		feed.CustomDir, feed.DownloadLatest, feed.Filters, feed.Tags, feed.FilenameTemplate, feed.ID)
	return err
}

func (m *Manager) updateUnreadCount(feedID string) error {
	var unreadCount int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM items WHERE feed_id = ? AND status = ?`, feedID, models.RSSItemStatusNew).Scan(&unreadCount)
	if err == nil {
		_, err = m.db.Exec(`UPDATE feeds SET unread_count = ? WHERE id = ?`, unreadCount, feedID)
	}
	return err
}

func (m *Manager) SetItemQueued(itemID string, queued bool) error {
	s := models.RSSItemStatusRead // Default to Read if un-queued
	if queued {
		s = models.RSSItemStatusQueued
	}
	// Don't overwrite Downloaded (4) status unless explicitly re-queuing?
	// For now, simple update.
	_, err := m.db.Exec(`UPDATE items SET status = ? WHERE id = ?`, s, itemID)
	if err != nil {
		return err
	}

	var feedID string
	if err := m.db.QueryRow("SELECT feed_id FROM items WHERE id = ?", itemID).Scan(&feedID); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) SetItemDownloadedByLink(link string, downloaded bool) error {
	s := models.RSSItemStatusRead // Default to Read if not downloaded
	if downloaded {
		s = models.RSSItemStatusDownloaded
	}
	_, err := m.db.Exec(`UPDATE items SET status = ? WHERE link = ?`, s, link)
	if err != nil {
		return err
	}

	var feedID string
	if err := m.db.QueryRow("SELECT feed_id FROM items WHERE link = ?", link).Scan(&feedID); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) SetItemFailedByLink(link string) error {
	_, err := m.db.Exec(`UPDATE items SET status = ? WHERE link = ?`, models.RSSItemStatusFailed, link)
	if err != nil {
		return err
	}

	var feedID string
	if err := m.db.QueryRow("SELECT feed_id FROM items WHERE link = ?", link).Scan(&feedID); err == nil {
		_ = m.updateUnreadCount(feedID)
	}
	return nil
}

func (m *Manager) RefreshFeed(feedID string) error {
	var url string
	err := m.db.QueryRow("SELECT url FROM feeds WHERE id = ?", feedID).Scan(&url)
	if err != nil {
		return err
	}

	fp := m.getParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update feed info
	_, err = tx.Exec(`UPDATE feeds SET title = ?, description = ?, last_updated = ? WHERE id = ?`,
		feed.Title, feed.Description, time.Now().Unix(), feedID)
	if err != nil {
		return err
	}
	if feed.Image != nil {
		_, _ = tx.Exec(`UPDATE feeds SET thumbnail = ? WHERE id = ?`, utils.EnsureHTTPS(feed.Image.URL), feedID)
	}

	// Insert new items
	for _, item := range feed.Items {
		pubDate := time.Now().Unix()
		if item.PublishedParsed != nil {
			pubDate = item.PublishedParsed.Unix()
		}
		// Check if item already exists by link
		var exists int
		err = tx.QueryRow(`SELECT 1 FROM items WHERE feed_id = ? AND link = ?`, feedID, item.Link).Scan(&exists)
		if err == nil && exists == 1 {
			continue
		}

		itemID := uuid.New().String()

		var thumbnail string
		if len(item.Enclosures) > 0 && (item.Enclosures[0].Type == "image/jpeg" || item.Enclosures[0].Type == "image/png") {
			thumbnail = utils.EnsureHTTPS(item.Enclosures[0].URL)
		} else if item.Image != nil {
			thumbnail = utils.EnsureHTTPS(item.Image.URL)
		}

		_, err = tx.Exec(`INSERT INTO items (id, feed_id, title, link, description, pub_date, status, thumbnail) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			itemID, feedID, item.Title, item.Link, item.Description, pubDate, models.RSSItemStatusNew, thumbnail)
		if err != nil {
			continue
		}
	}

	// Update unread count
	// This is a bit expensive but accurate. Or we could just increment for new items if we tracked them.
	// For simplicity, let's recount.
	var unreadCount int
	err = tx.QueryRow(`SELECT COUNT(*) FROM items WHERE feed_id = ? AND status = ?`, feedID, models.RSSItemStatusNew).Scan(&unreadCount)
	if err == nil {
		_, _ = tx.Exec(`UPDATE feeds SET unread_count = ? WHERE id = ?`, unreadCount, feedID)
	}

	if err := tx.Commit(); err != nil {
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
	// Only mark as read if it's currently New (0)
	_, err := m.db.Exec(`UPDATE items SET status = ? WHERE id = ? AND status = ?`, models.RSSItemStatusRead, itemID, models.RSSItemStatusNew)
	// Update feed unread count
	var feedID string
	if err := m.db.QueryRow("SELECT feed_id FROM items WHERE id = ?", itemID).Scan(&feedID); err == nil {
		var unreadCount int
		m.db.QueryRow(`SELECT COUNT(*) FROM items WHERE feed_id = ? AND status = ?`, feedID, models.RSSItemStatusNew).Scan(&unreadCount)
		m.db.Exec(`UPDATE feeds SET unread_count = ? WHERE id = ?`, unreadCount, feedID)
	}
	return err
}

func (m *Manager) processAutoDownload(feedID string) {
	if m.OnAutoDownload == nil {
		return
	}

	var feed models.RSSFeed
	var downloadLatest int
	err := m.db.QueryRow("SELECT id, url, title, thumbnail, custom_dir, download_latest, filters, tags, filename_template FROM feeds WHERE id = ?", feedID).
		Scan(&feed.ID, &feed.URL, &feed.Title, &feed.Thumbnail, &feed.CustomDir, &downloadLatest, &feed.Filters, &feed.Tags, &feed.FilenameTemplate)
	if err != nil {
		return
	}
	feed.DownloadLatest = downloadLatest == 1

	if !feed.DownloadLatest {
		return
	}

	rows, err := m.db.Query(`SELECT id, feed_id, title, link, description, pub_date, status, thumbnail FROM items WHERE feed_id = ? AND (status = ? OR status = ?)`,
		feedID, models.RSSItemStatusNew, models.RSSItemStatusRead)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.RSSItem
		if err := rows.Scan(&item.ID, &item.FeedID, &item.Title, &item.Link, &item.Description, &item.PubDate, &item.Status, &item.Thumbnail); err != nil {
			continue
		}

		m.OnAutoDownload(item, feed)

		_, _ = m.db.Exec(`UPDATE items SET status = ? WHERE id = ?`, models.RSSItemStatusQueued, item.ID)
	}

	_ = m.updateUnreadCount(feedID)
}
