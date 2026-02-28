package dal

import (
	"context"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RSSDAL struct {
	db *gorm.DB
}

func NewRSSDAL(db *gorm.DB) *RSSDAL {
	return &RSSDAL{db: db}
}

func (d *RSSDAL) CreateFeed(ctx context.Context, feed *schema.Feed) error {
	return d.db.WithContext(ctx).Create(feed).Error
}

func (d *RSSDAL) CreateFeedItems(ctx context.Context, items []schema.FeedItem) error {
	if len(items) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&items).Error
}

func (d *RSSDAL) ListFeeds(ctx context.Context) ([]schema.Feed, error) {
	var feeds []schema.Feed
	err := d.db.WithContext(ctx).Find(&feeds).Error
	return feeds, err
}

func (d *RSSDAL) GetFeedByID(ctx context.Context, id string) (*schema.Feed, error) {
	var feed schema.Feed
	err := d.db.WithContext(ctx).First(&feed, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

func (d *RSSDAL) GetFeedURLByID(ctx context.Context, id string) (string, error) {
	var feed schema.Feed
	err := d.db.WithContext(ctx).Select("url").First(&feed, "id = ?", id).Error
	if err != nil {
		return "", err
	}
	return feed.URL, nil
}

func (d *RSSDAL) UpdateFeed(ctx context.Context, id string, updates map[string]interface{}) error {
	return d.db.WithContext(ctx).Model(&schema.Feed{}).Where("id = ?", id).Updates(updates).Error
}

func (d *RSSDAL) DeleteFeed(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&schema.FeedItem{}, "feed_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&schema.Feed{}, "id = ?", id).Error
	})
}

func (d *RSSDAL) ListFeedItems(ctx context.Context, feedID string) ([]schema.FeedItem, error) {
	var items []schema.FeedItem
	err := d.db.WithContext(ctx).Where("feed_id = ?", feedID).Order("pub_date desc").Find(&items).Error
	return items, err
}

func (d *RSSDAL) ListFeedItemsByStatuses(ctx context.Context, feedID string, statuses []int) ([]schema.FeedItem, error) {
	var items []schema.FeedItem
	err := d.db.WithContext(ctx).Where("feed_id = ? AND status IN ?", feedID, statuses).Find(&items).Error
	return items, err
}

func (d *RSSDAL) UpdateFeedItemStatusByID(ctx context.Context, id string, status int) error {
	return d.db.WithContext(ctx).Model(&schema.FeedItem{}).Where("id = ?", id).Update("status", status).Error
}

func (d *RSSDAL) UpdateFeedItemStatusByIDIf(ctx context.Context, id string, fromStatus int, toStatus int) error {
	return d.db.WithContext(ctx).Model(&schema.FeedItem{}).Where("id = ? AND status = ?", id, fromStatus).Update("status", toStatus).Error
}

func (d *RSSDAL) UpdateFeedItemStatusByLink(ctx context.Context, link string, status int) error {
	return d.db.WithContext(ctx).Model(&schema.FeedItem{}).Where("link = ?", link).Update("status", status).Error
}

func (d *RSSDAL) GetFeedIDByItemID(ctx context.Context, id string) (string, error) {
	var item schema.FeedItem
	err := d.db.WithContext(ctx).Select("feed_id").First(&item, "id = ?", id).Error
	if err != nil {
		return "", err
	}
	return item.FeedID, nil
}

func (d *RSSDAL) GetFeedIDByItemLink(ctx context.Context, link string) (string, error) {
	var item schema.FeedItem
	err := d.db.WithContext(ctx).Select("feed_id").First(&item, "link = ?", link).Error
	if err != nil {
		return "", err
	}
	return item.FeedID, nil
}

func (d *RSSDAL) CountUnread(ctx context.Context, feedID string, status int) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&schema.FeedItem{}).Where("feed_id = ? AND status = ?", feedID, status).Count(&count).Error
	return count, err
}
