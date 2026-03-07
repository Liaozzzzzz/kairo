package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func (d *API) GetPlatformName() string {
	return "douyin"
}

func (d *API) ValidateConfig() error {
	return automation.ValidateConfig(d.config)
}

func (d *API) ValidateAccount(ctx context.Context, cookiePath string) error {
	// 1. Try to validate existing cookie if it exists
	if info, err := os.Stat(cookiePath); err == nil && info.Size() > 0 {
		if err := d.checkLoginStatus(ctx, cookiePath); err == nil {
			log.Printf("Cookie validation passed for account %s\n", cookiePath)
			return nil
		}
		// If validation fails, log it and proceed to login flow
		log.Printf("Cookie validation failed for account %s, starting login flow...\n", cookiePath)
	}

	// 2. Start login flow
	return d.performLogin(ctx, cookiePath)
}

func (d *API) checkLoginStatus(ctx context.Context, cookiePath string) error {
	config := automation.ParseConfig(d.config)
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
	if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload"); err != nil {
		return fmt.Errorf("failed to navigate to publish page: %v", err)
	}

	// Wait for potential redirects
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	// Check if redirected to login page (Douyin usually redirects to sso or login)
	if strings.Contains(page.URL(), "login") || strings.Contains(page.URL(), "sso") {
		return fmt.Errorf("cookie expired or invalid (redirected to login)")
	}

	// Check for login prompts on the page
	if err := automation.EnsureLoggedIn(page); err != nil {
		return err
	}

	return nil
}

func (d *API) performLogin(ctx context.Context, cookiePath string) error {
	config := automation.ParseConfig(d.config)
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
	if _, err := page.Goto("https://creator.douyin.com/"); err != nil {
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
			// 1. Check URL
			if strings.Contains(page.URL(), "creator-micro") {
				// Double check ensuring login buttons are gone
				if err := automation.EnsureLoggedIn(page); err == nil {
					// Success! Save cookies
					if err := automation.SaveCookies(session.Context, cookiePath); err != nil {
						return fmt.Errorf("failed to save cookies: %v", err)
					}
					return nil
				}
			}

			// Also check if we are on the home page but logged in (sometimes URL might not change immediately if we are on root)
			// But creator.douyin.com usually redirects to creator-micro
		}
	}
}

func (d *API) UploadVideo(ctx context.Context, title, description string, tags []string, videoPath, accountCookiePath string, scheduledAt *time.Time) (string, error) {
	if strings.TrimSpace(accountCookiePath) == "" {
		return "", fmt.Errorf("account cookie is required")
	}
	if _, err := os.Stat(accountCookiePath); err != nil {
		return "", fmt.Errorf("account cookie not found: %v", err)
	}
	config := automation.ParseConfig(d.config)
	if err := uploadToDouyin(ctx, automation.UploadInput{
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
	return fmt.Sprintf("dy_%d", time.Now().Unix()), nil
}

func uploadToDouyin(ctx context.Context, input automation.UploadInput) error {
	return nil
	// session, err := automation.StartBrowserSession(ctx, input)
	// if err != nil {
	// 	return err
	// }
	// defer session.Close()

	// // Use session context to detect browser close
	// ctx = session.Ctx
	// page := session.Page

	// if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload"); err != nil {
	// 	log.Printf("Failed to navigate to publish page: %v", err)
	// 	return err
	// }
	// log.Printf("[uploadToDouyin] Navigated to publish page")
	// if err := page.WaitForURL("https://creator.douyin.com/creator-micro/content/upload"); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to wait for publish page to load: %v", err)
	// 	return err
	// }
	// log.Printf("[uploadToDouyin] Publish page loaded successfully")
	// if err := automation.EnsureLoggedIn(page); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to ensure login: %v", err)
	// 	return err
	// }

	// log.Printf("[uploadToDouyin] Setting input file: %s", input.VideoPath)
	// if err := automation.SetInputFile(page, "div[class^='container'] input", input.VideoPath); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to set input file: %v", err)
	// 	return err
	// }

	// log.Printf("[uploadToDouyin] Waiting for publish page to load")
	// if err := waitForDouyinPublishPage(ctx, page); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to wait for publish page to load: %v", err)
	// 	return err
	// }

	// log.Printf("[uploadToDouyin] Filling title and tags")
	// if err := fillDouyinTitleAndTags(page, input.Title, automation.NormalizeTags(input.Tags)); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to fill title and tags: %v", err)
	// 	return err
	// }

	// log.Printf("[uploadToDouyin] Waiting for upload to complete")
	// if err := waitForDouyinUploadDone(ctx, page, input.VideoPath); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to wait for upload to complete: %v", err)
	// 	return err
	// }

	// log.Printf("[uploadToDouyin] Publishing video")
	// if err := publishDouyin(ctx, page); err != nil {
	// 	log.Printf("[uploadToDouyin] Failed to publish video: %v", err)
	// 	return err
	// }
	// return automation.SaveCookies(session.Context, input.AccountCookies)
}

func waitForDouyinPublishPage(ctx context.Context, page playwright.Page) error {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		url := page.URL()
		if strings.Contains(url, "creator-micro/content/publish?enter_from=publish_page") ||
			strings.Contains(url, "creator-micro/content/post/video?enter_from=publish_page") {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("publish page not ready")
}

func fillDouyinTitleAndTags(page playwright.Page, title string, tags []string) error {
	titleLocator := page.Locator("text=作品标题").Locator("..").Locator("xpath=following-sibling::div[1]").Locator("input")
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
	for _, tag := range tags {
		if err := page.Type(".zone-container", "#"+tag); err != nil {
			return err
		}
		if err := page.Press(".zone-container", "Space"); err != nil {
			return err
		}
	}
	return nil
}

func waitForDouyinUploadDone(ctx context.Context, page playwright.Page, videoPath string) error {
	deadline := time.Now().Add(15 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		successCount, err := page.Locator(`[class^="long-card"] div:has-text("重新上传")`).Count()
		if err == nil && successCount > 0 {
			return nil
		}

		// Check for failure: "上传失败" text appears
		failureCount, err := page.Locator("div.progress-div > div:has-text('上传失败')").Count()
		if err == nil && failureCount > 0 {
			log.Println("[-] Upload failed detected, retrying...")
			// Retry logic: set input file again
			retryInput := page.Locator(`div.progress-div [class^="upload-btn-input"]`)
			if err := retryInput.SetInputFiles(videoPath); err != nil {
				return fmt.Errorf("failed to retry upload: %w", err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("upload timeout")
}

func publishDouyin(ctx context.Context, page playwright.Page) error {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		publishButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "发布", Exact: playwright.Bool(true)})
		if count, err := publishButton.Count(); err == nil && count > 0 {
			if err := publishButton.Click(); err == nil {
				// Check for success URL
				if err := page.WaitForURL("https://creator.douyin.com/creator-micro/content/manage**", playwright.PageWaitForURLOptions{
					Timeout: playwright.Float(3000),
				}); err == nil {
					return nil
				}
			}
		}

		// Check if we need to handle auto cover (if publish failed or button click didn't redirect)
		if err := handleAutoVideoCover(ctx, page); err != nil {
			log.Printf("handleAutoVideoCover error: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("publish timeout")
}

func handleAutoVideoCover(ctx context.Context, page playwright.Page) error {
	// 1. Check if "请设置封面后再发布" prompt is visible
	prompt := page.GetByText("请设置封面后再发布").First()
	if visible, err := prompt.IsVisible(); err != nil || !visible {
		return nil
	}

	log.Println("[-] Detected cover requirement prompt...")

	// 2. Select the first recommended cover
	// Use class^= prefix matching
	recommendCover := page.Locator(`[class^="recommendCover-"]`).First()
	if count, _ := recommendCover.Count(); count > 0 {
		log.Println("[-] Selecting first recommended cover...")
		if err := recommendCover.Click(); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)

		// 3. Handle confirmation dialog "是否确认应用此封面？"
		confirmText := "是否确认应用此封面？"
		confirmPrompt := page.GetByText(confirmText).First()
		if visible, err := confirmPrompt.IsVisible(); err == nil && visible {
			log.Printf("[-] Detected confirmation dialog: %s", confirmText)
			if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"}).Click(); err != nil {
				return err
			}
			log.Println("[-] Clicked confirm cover")
			time.Sleep(1 * time.Second)
		}
		return nil
	}
	return fmt.Errorf("recommend cover not found")
}
