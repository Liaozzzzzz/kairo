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

	// 1. Navigate to Publish Page
	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video"); err != nil {
		return fmt.Errorf("failed to navigate to publish page: %v", err)
	}
	if err := page.WaitForURL("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video"); err != nil {
		return fmt.Errorf("failed to wait for publish page URL: %v", err)
	}

	// 2. Ensure Logged In
	if err := automation.EnsureLoggedIn(page); err != nil {
		return fmt.Errorf("not logged in: %v", err)
	}

	// 3. Upload Video
	// Wait for upload input to be available
	uploadInput := page.Locator("input.upload-input")
	if err := uploadInput.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateAttached}); err != nil {
		return fmt.Errorf("upload input not found: %v", err)
	}
	if err := uploadInput.SetInputFiles(input.VideoPath); err != nil {
		return fmt.Errorf("failed to set input file: %v", err)
	}

	// 4. Wait for Upload Completion
	if err := waitForXiaohongshuUpload(ctx, page); err != nil {
		return fmt.Errorf("upload failed or timed out: %v", err)
	}

	// 5. Fill Title
	// XHS Title limit is 20 chars, but we can try to fill more and let user/UI handle it,
	// or truncate as per skill recommendation.
	if err := fillXiaohongshuTitle(page, input.Title); err != nil {
		return fmt.Errorf("failed to fill title: %v", err)
	}

	// 6. Fill Description and Tags
	if err := fillXiaohongshuDescription(page, input.Description, automation.NormalizeTags(input.Tags)); err != nil {
		return fmt.Errorf("failed to fill description: %v", err)
	}

	// 7. Set Schedule (Optional)
	if input.ScheduledAt != nil {
		if err := setXiaohongshuSchedule(page, input.ScheduledAt); err != nil {
			return fmt.Errorf("failed to set schedule: %v", err)
		}
	}

	// 8. Publish
	if err := publishXiaohongshu(ctx, page, input.ScheduledAt != nil); err != nil {
		return fmt.Errorf("failed to publish: %v", err)
	}

	return automation.SaveCookies(session.Context, input.AccountCookies)
}

func waitForXiaohongshuUpload(ctx context.Context, page playwright.Page) error {
	// Wait for "上传成功" text to appear in the upload area
	// The structure usually changes after upload starts.
	// We can look for the success indicator or the absence of progress bar.

	// Retry loop for 15 minutes
	deadline := time.Now().Add(15 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("upload timeout")
			}

			// Check for success message
			// Common selector for success state
			success, err := page.Locator(":text('上传成功')").IsVisible()
			if err == nil && success {
				return nil
			}

			// Also check if "re-upload" button is visible which implies success
			reupload, err := page.Locator(":text('重新上传')").IsVisible()
			if err == nil && reupload {
				return nil
			}
		}
	}
}

func fillXiaohongshuTitle(page playwright.Page, title string) error {
	// Try standard input selector first
	// Usually: input[placeholder*="标题"]
	titleLocator := page.Locator("input[placeholder*='标题']")
	if count, _ := titleLocator.Count(); count == 0 {
		// Fallback to class based
		titleLocator = page.Locator(".title-input input")
	}

	// Clear existing
	if err := titleLocator.Click(); err != nil {
		return err
	}
	if err := titleLocator.Clear(); err != nil {
		// If clear fails, try manual delete
		page.Keyboard().Press("Control+A")
		page.Keyboard().Press("Backspace")
	}

	// Trim to 20 chars if strictly following skill, but let's be flexible
	// safeTitle := automation.TrimTitle(title, 20)
	return titleLocator.Fill(title)
}

func fillXiaohongshuDescription(page playwright.Page, description string, tags []string) error {
	// Description is usually a contenteditable div
	// Locator: .post-content or #post-textarea or similar
	descLocator := page.Locator(".post-content")
	if count, _ := descLocator.Count(); count == 0 {
		descLocator = page.Locator("div[contenteditable='true']") // Generic fallback
	}

	if err := descLocator.Click(); err != nil {
		return err
	}

	// Fill description
	if err := descLocator.Fill(description); err != nil {
		return err
	}

	// Append tags
	if len(tags) > 0 {
		// Move to end
		page.Keyboard().Press("End")
		page.Keyboard().Press("Enter")

		for _, tag := range tags {
			// Type #tag then space to trigger tag creation
			if err := page.Keyboard().Type("#" + tag + " "); err != nil {
				return err
			}
			time.Sleep(200 * time.Millisecond) // Wait for tag UI
		}
	}
	return nil
}

func setXiaohongshuSchedule(page playwright.Page, scheduledAt *time.Time) error {
	// Locate "定时发布" radio/button
	// Usually text="定时发布"
	if err := page.Locator(":text('定时发布')").Click(); err != nil {
		return err
	}

	// Format time: 2024-05-20 12:00
	timeStr := scheduledAt.Format("2006-01-02 15:04")

	// Find the time input
	// This is tricky as it might be a complex picker.
	// Often clicking the input and typing works.
	timeInput := page.Locator("input[placeholder*='日期']")
	if err := timeInput.Click(); err != nil {
		return err
	}

	// Select all and type
	if err := page.Keyboard().Press("Control+A"); err != nil {
		return err
	}
	if err := page.Keyboard().Type(timeStr); err != nil {
		return err
	}

	// Close picker (click outside)
	return page.Locator("body").Click()
}

func publishXiaohongshu(ctx context.Context, page playwright.Page, isScheduled bool) error {
	btnText := "发布"
	if isScheduled {
		btnText = "定时发布"
	}

	// Click the button
	btn := page.Locator(fmt.Sprintf("button:has-text('%s')", btnText))
	if err := btn.Click(); err != nil {
		return err
	}

	// Wait for success confirmation
	// Usually a toast or redirect
	// Wait for "发布成功" toast
	_, err := page.WaitForSelector(":text('发布成功')", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000), // 10 seconds
	})
	return err
}
