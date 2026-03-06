package publish

import (
	"Kairo/internal/config"
	"Kairo/internal/db/schema"
	"Kairo/internal/publish/platforms/automation"
	"Kairo/internal/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (p *PublishManager) ListAccounts(platformID string) ([]schema.PublishAccount, error) {
	return p.publishAccountDAL.ListAccounts(p.ctx, platformID)
}

func (p *PublishManager) CreateAccount(platformID, name string, isEnabled bool, cookieFilePath string, publishInterval string) (*schema.PublishAccount, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("account name is required")
	}

	_, err := p.publishPlatformDAL.GetPlatformById(p.ctx, platformID)
	if err != nil {
		return nil, fmt.Errorf("platform not found: %v", err)
	}

	if publishInterval == "" {
		publishInterval = "1h"
	}

	account := &schema.PublishAccount{
		ID:              uuid.New().String(),
		PlatformID:      platformID,
		Name:            name,
		IsEnabled:       isEnabled,
		PublishInterval: publishInterval,
		Status:          schema.PublishAccountStatusUnknown, // Default to unknown
		LastChecked:     time.Now().UnixMilli(),
	}

	destPath, err := p.getAccountCookiePath(account.ID)
	if err != nil {
		return nil, err
	}
	account.CookiePath = destPath

	// Handle cookie file if provided, otherwise just set the path
	if strings.TrimSpace(cookieFilePath) != "" {
		if err := p.saveAccountCookie(cookieFilePath, destPath, platformID); err != nil {
			return nil, err
		}
		account.Status = schema.PublishAccountStatusActive
	} else {
		// Write empty string to destination
		if err := os.WriteFile(destPath, []byte(""), 0644); err != nil {
			return nil, fmt.Errorf("failed to write cookie file: %v", err)
		}
	}

	if err := p.publishAccountDAL.SaveAccount(p.ctx, account); err != nil {
		return nil, fmt.Errorf("failed to save account: %v", err)
	}
	return account, nil
}

func (p *PublishManager) UpdateAccount(id, name string, isEnabled bool, cookieFilePath string, publishInterval string) (*schema.PublishAccount, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("account name is required")
	}

	account, err := p.publishAccountDAL.GetAccountById(p.ctx, id)
	if err != nil {
		return nil, err
	}

	account.Name = name
	account.IsEnabled = isEnabled
	if publishInterval != "" {
		account.PublishInterval = publishInterval
	}

	if strings.TrimSpace(cookieFilePath) != "" {
		if err := p.saveAccountCookie(cookieFilePath, account.CookiePath, account.PlatformID); err != nil {
			return nil, err
		}
		account.Status = schema.PublishAccountStatusActive
	} else {
		// Write empty string to destination
		if err := os.WriteFile(account.CookiePath, []byte(""), 0644); err != nil {
			return nil, fmt.Errorf("failed to write cookie file: %v", err)
		}
	}

	account.LastChecked = time.Now().UnixMilli()
	if err := p.publishAccountDAL.SaveAccount(p.ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update account: %v", err)
	}
	return account, nil
}

func (p *PublishManager) ValidateAccount(id string) (*schema.PublishAccount, error) {
	account, err := p.publishAccountDAL.GetAccountById(p.ctx, id)
	if err != nil {
		return nil, err
	}

	status := schema.PublishAccountStatusInvalid
	var validationErr error

	// Platform-specific validation
	platform, err := p.publishPlatformDAL.GetPlatformById(p.ctx, account.PlatformID)
	if err != nil {
		log.Printf("[ValidateAccount] failed to get platform: %v", err)
		return nil, fmt.Errorf("failed to get platform: %v", err)
	}

	api, err := p.platformManager.CreatePlatformAPI(*platform)
	if err != nil {
		log.Printf("[ValidateAccount] failed to create platform api: %v", err)
		return nil, fmt.Errorf("failed to create platform api: %v", err)
	}

	// API ValidateAccount should handle missing/invalid cookies by triggering login if supported
	if err := api.ValidateAccount(p.ctx, account.CookiePath); err != nil {
		status = schema.PublishAccountStatusInvalid
		validationErr = err
	} else {
		status = schema.PublishAccountStatusActive
	}

	account.Status = status
	account.LastChecked = time.Now().UnixMilli()

	if err := p.publishAccountDAL.SaveAccount(p.ctx, account); err != nil {
		log.Printf("[ValidateAccount] failed to update account status: %v", err)
		return nil, fmt.Errorf("failed to update account status: %v", err)
	}

	if validationErr != nil {
		log.Printf("[ValidateAccount] failed to validate account: %v", validationErr)
	}

	return account, validationErr
}

func (p *PublishManager) DeleteAccount(id string) error {
	account, err := p.publishAccountDAL.GetAccountById(p.ctx, id)
	if err != nil {
		return err
	}

	if err := utils.DeleteFile(account.CookiePath); err != nil {
		return err
	}
	return p.publishAccountDAL.DeleteAccount(p.ctx, id)
}

func (p *PublishManager) getAccountCookiePath(accountID string) (string, error) {
	baseDir, err := config.GetAppConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(baseDir, "accounts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, accountID), nil
}

func (p *PublishManager) saveAccountCookie(sourcePath, destPath, platformID string) error {
	// Read source content
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read cookie file: %v", err)
	}

	// Get platform domain for header cookie parsing
	var targetDomain string
	platform, err := p.publishPlatformDAL.GetPlatformById(p.ctx, platformID)
	if err == nil {
		// Map platform name to domain
		// This is a simple mapping, could be moved to platform definition
		switch platform.Name {
		case string(schema.PlatformXiaohongshu):
			targetDomain = ".xiaohongshu.com"
		case string(schema.PlatformDouyin):
			targetDomain = ".douyin.com"
		}
	}

	// Convert content to StorageState JSON
	jsonBytes, err := automation.ConvertToStorageState(string(content), targetDomain)
	if err != nil {
		return fmt.Errorf("failed to convert cookie format: %v", err)
	}

	// Write JSON to destination
	if err := os.WriteFile(destPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write cookie file: %v", err)
	}

	return nil
}
