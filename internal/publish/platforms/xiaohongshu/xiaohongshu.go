package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"Kairo/internal/publish/platforms/automation"

	"github.com/playwright-community/playwright-go"
)

type API struct {
	config map[string]interface{}
}

func New(config string) (*API, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, fmt.Errorf("invalid config format: %v", err)
	}
	return &API{config: cfg}, nil
}

func (x *API) GetPlatformName() string {
	return "xiaohongshu"
}

func (x *API) ValidateConfig() error {
	return automation.ValidateConfig(x.config)
}

func (x *API) ValidateAccount(ctx context.Context, cookiePath string) error {
	// 1. Try to validate existing cookie if it exists
	if info, err := os.Stat(cookiePath); err == nil && info.Size() > 0 {
		if err := x.checkLoginStatus(ctx, cookiePath); err == nil {
			return nil
		}
		// If validation fails, log it and proceed to login flow
		fmt.Printf("Cookie validation failed, starting login flow...\n")
	}

	// 2. Start login flow
	return x.performLogin(ctx, cookiePath)
}

func (x *API) checkLoginStatus(ctx context.Context, cookiePath string) error {
	config := automation.ParseConfig(x.config)
	// Use headless mode for validation to be less intrusive
	config.Headless = true

	session, err := automation.StartBrowserSession(ctx, automation.UploadInput{
		Config:         config,
		AccountCookies: cookiePath,
	})
	if err != nil {
		return fmt.Errorf("failed to start browser session: %v", err)
	}
	defer session.Close()

	// Use session context to detect browser close
	ctx = session.Ctx
	page := session.Page
	// Navigate to publish page which requires authentication
	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video"); err != nil {
		return fmt.Errorf("failed to navigate to publish page: %v", err)
	}

	// Wait for potential redirects
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	// Check if redirected to login page
	if strings.Contains(page.URL(), "login") {
		return fmt.Errorf("cookie expired or invalid (redirected to login)")
	}

	// Check for login prompts on the page
	if err := automation.EnsureLoggedIn(page); err != nil {
		return err
	}

	return nil
}

func (x *API) performLogin(ctx context.Context, cookiePath string) error {
	config := automation.ParseConfig(x.config)
	// Must be headed for manual login
	config.Headless = false

	// Start session without cookies initially
	session, err := automation.StartBrowserSession(ctx, automation.UploadInput{
		Config: config,
	})
	if err != nil {
		return fmt.Errorf("failed to start browser session for login: %v", err)
	}
	defer session.Close()

	// Use session context to detect browser close
	ctx = session.Ctx
	page := session.Page

	// Go to creator home page
	if _, err := page.Goto("https://creator.xiaohongshu.com/"); err != nil {
		return fmt.Errorf("failed to open login page: %v", err)
	}

	// Wait for login success
	// We wait for the URL to change to the creator backend or for the login buttons to disappear
	// Set a reasonable timeout for user to scan QR code (e.g. 5 minutes)
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("login timeout")
		case <-ticker.C:
			// Check if we are logged in
			// 1. Check URL - Xiaohongshu creator center usually has /creator/ or /publish/ in path
			// The login page is usually https://creator.xiaohongshu.com/login
			// After login it redirects to https://creator.xiaohongshu.com/publish/publish or similar
			currentURL := page.URL()
			if !strings.Contains(currentURL, "login") && (strings.Contains(currentURL, "publish") || strings.Contains(currentURL, "creator")) {
				// Double check ensuring login buttons are gone
				if err := automation.EnsureLoggedIn(page); err == nil {
					// Success! Save cookies
					if err := automation.SaveCookies(session.Context, cookiePath); err != nil {
						return fmt.Errorf("failed to save cookies: %v", err)
					}
					return nil
				}
			}
		}
	}
}

func (x *API) UploadVideo(ctx context.Context, title, description string, tags []string, videoPath, accountCookiePath string, scheduledAt *time.Time) (string, error) {
	if strings.TrimSpace(accountCookiePath) == "" {
		return "", fmt.Errorf("account cookie is required")
	}
	if _, err := os.Stat(accountCookiePath); err != nil {
		return "", fmt.Errorf("account cookie not found: %v", err)
	}
	config := automation.ParseConfig(x.config)
	if err := uploadToXiaohongshu(ctx, automation.UploadInput{
		Title:          title,
		Description:    description,
		Tags:           tags,
		VideoPath:      videoPath,
		AccountCookies: accountCookiePath,
		ScheduledAt:    scheduledAt,
		Config:         config,
	}); err != nil {
		return "", err
	}
	return fmt.Sprintf("xhs_%d", time.Now().Unix()), nil
}

func uploadToXiaohongshu(ctx context.Context, input automation.UploadInput) error {
	session, err := automation.StartBrowserSession(ctx, input)
	if err != nil {
		return err
	}
	defer session.Close()

	// Use session context to detect browser close
	ctx = session.Ctx
	page := session.Page

	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video"); err != nil {
		return err
	}
	if err := page.WaitForURL("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video"); err != nil {
		return err
	}
	if err := automation.EnsureLoggedIn(page); err != nil {
		return err
	}
	if err := automation.SetInputFile(page, "div[class^='upload-content'] input.upload-input", input.VideoPath); err != nil {
		return err
	}
	if err := waitForXiaohongshuUpload(ctx, page); err != nil {
		return err
	}
	if err := fillXiaohongshuTitleAndTags(page, input.Title, automation.NormalizeTags(input.Tags)); err != nil {
		return err
	}
	if input.ScheduledAt != nil {
		if err := setXiaohongshuSchedule(page, input.ScheduledAt); err != nil {
			return err
		}
	}
	if err := publishXiaohongshu(ctx, page, input.ScheduledAt != nil); err != nil {
		return err
	}
	return automation.SaveCookies(session.Context, input.AccountCookies)
}

func waitForXiaohongshuUpload(ctx context.Context, page playwright.Page) error {
	deadline := time.Now().Add(15 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		result, err := page.Evaluate(`() => {
			const uploadInput = document.querySelector('input.upload-input');
			if (!uploadInput) return false;
			const preview = uploadInput.parentElement?.querySelector('div.preview-new');
			if (!preview) return false;
			const stages = preview.querySelectorAll('div.stage');
			return Array.from(stages).some(stage => (stage.textContent || '').includes('上传成功'));
		}`)
		if err == nil {
			if ok, castOk := result.(bool); castOk && ok {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("upload timeout")
}

func fillXiaohongshuTitleAndTags(page playwright.Page, title string, tags []string) error {
	titleLocator := page.Locator("div.plugin.title-container input.d-text")
	if count, err := titleLocator.Count(); err == nil && count > 0 {
		if err := titleLocator.Fill(automation.TrimTitle(title, 30)); err != nil {
			return err
		}
	} else {
		alt := page.Locator(".notranslate")
		if err := alt.Click(); err != nil {
			return err
		}
		if err := page.Keyboard().Press("Control+KeyA"); err != nil {
			return err
		}
		if err := page.Keyboard().Press("Delete"); err != nil {
			return err
		}
		if err := page.Keyboard().Type(title); err != nil {
			return err
		}
		if err := page.Keyboard().Press("Enter"); err != nil {
			return err
		}
	}
	tagSelector := ".ql-editor"
	for _, tag := range tags {
		if err := page.Type(tagSelector, "#"+tag); err != nil {
			return err
		}
		if err := page.Press(tagSelector, "Space"); err != nil {
			return err
		}
	}
	return nil
}

func setXiaohongshuSchedule(page playwright.Page, scheduledAt *time.Time) error {
	if scheduledAt == nil {
		return nil
	}
	label := page.Locator("label:has-text('定时发布')")
	if err := label.Click(); err != nil {
		return err
	}
	dateInput := page.Locator(`.el-input__inner[placeholder="选择日期和时间"]`)
	dateValue := scheduledAt.Format("2006-01-02 15:04")
	if err := dateInput.Fill(dateValue); err != nil {
		return err
	}
	return page.Keyboard().Press("Enter")
}

func publishXiaohongshu(ctx context.Context, page playwright.Page, scheduled bool) error {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		buttonText := "发布"
		if scheduled {
			buttonText = "定时发布"
		}
		if err := page.Locator("button:has-text(\"" + buttonText + "\")").Click(); err == nil {
			if err := page.WaitForURL("https://creator.xiaohongshu.com/publish/success?**"); err == nil {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("publish timeout")
}
