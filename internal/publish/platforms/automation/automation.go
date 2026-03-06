package automation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type Config struct {
	ChromePath string
	Headless   bool
	StealthJS  string
}

type UploadInput struct {
	Title          string
	Description    string
	Tags           []string
	VideoPath      string
	AccountCookies string
	ScheduledAt    *time.Time
	Config         Config
}

type BrowserSession struct {
	Playwright *playwright.Playwright
	Browser    playwright.Browser
	Context    playwright.BrowserContext
	Page       playwright.Page
	Ctx        context.Context
	Cancel     context.CancelFunc
}

func (s *BrowserSession) Close() {
	if s.Cancel != nil {
		s.Cancel()
	}
	if s.Context != nil {
		_ = s.Context.Close()
	}
	if s.Browser != nil {
		_ = s.Browser.Close()
	}
	if s.Playwright != nil {
		_ = s.Playwright.Stop()
	}
}

func ParseConfig(config map[string]interface{}) Config {
	return Config{
		ChromePath: getStringValue(config, "chrome_path"),
		Headless:   getBoolValue(config, "headless"),
		StealthJS:  getStringValue(config, "stealth_js_path"),
	}
}

func ValidateConfig(config map[string]interface{}) error {
	cfg := ParseConfig(config)
	if cfg.ChromePath != "" {
		if _, err := os.Stat(cfg.ChromePath); err != nil {
			return fmt.Errorf("chrome path not found: %v", err)
		}
	}
	if cfg.StealthJS != "" {
		if _, err := os.Stat(cfg.StealthJS); err != nil {
			return fmt.Errorf("stealth js not found: %v", err)
		}
	}
	return nil
}

func StartBrowserSession(ctx context.Context, input UploadInput) (*BrowserSession, error) {
	// Create a new context that can be cancelled when the browser is closed
	ctx, cancel := context.WithCancel(ctx)

	if err := playwright.Install(); err != nil {
		cancel()
		return nil, err
	}
	pw, err := playwright.Run()
	if err != nil {
		cancel()
		return nil, err
	}
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(input.Config.Headless),
	}
	if strings.TrimSpace(input.Config.ChromePath) != "" {
		launchOptions.ExecutablePath = playwright.String(input.Config.ChromePath)
	}
	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		_ = pw.Stop()
		cancel()
		return nil, err
	}

	var storageStatePath *string
	if strings.TrimSpace(input.AccountCookies) != "" {
		storageStatePath = playwright.String(input.AccountCookies)
	}

	contextOptions := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1600,
			Height: 900,
		},
		StorageStatePath: storageStatePath,
	}
	log.Printf("[StartBrowserSession] using cookies: %v", storageStatePath)

	context, err := browser.NewContext(contextOptions)
	if err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		cancel()
		return nil, err
	}

	// Listen for context close event
	context.OnClose(func(playwright.BrowserContext) {
		log.Println("[StartBrowserSession] Context closed by user")
		cancel()
	})

	if strings.TrimSpace(input.Config.StealthJS) != "" {
		if err := context.AddInitScript(playwright.Script{Path: playwright.String(input.Config.StealthJS)}); err != nil {
			_ = context.Close()
			_ = browser.Close()
			_ = pw.Stop()
			cancel()
			return nil, err
		}
	}
	page, err := context.NewPage()
	if err != nil {
		_ = context.Close()
		_ = browser.Close()
		_ = pw.Stop()
		cancel()
		return nil, err
	}

	// Listen for page close event
	page.OnClose(func(playwright.Page) {
		log.Println("[StartBrowserSession] Page closed by user")
		cancel()
	})

	return &BrowserSession{
		Playwright: pw,
		Browser:    browser,
		Context:    context,
		Page:       page,
		Ctx:        ctx,
		Cancel:     cancel,
	}, nil
}

func SaveCookies(context playwright.BrowserContext, cookiePath string) error {
	if context == nil {
		return nil
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cookiePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for cookies: %v", err)
	}
	_, err := context.StorageState(cookiePath)
	return err
}

func EnsureLoggedIn(page playwright.Page) error {
	if page == nil {
		return fmt.Errorf("page not initialized")
	}
	loginPrompt, err := page.Locator("text=手机号登录").Count()
	if err == nil && loginPrompt > 0 {
		return fmt.Errorf("cookie invalid: 手机号登录")
	}
	qrPrompt, err := page.Locator("text=扫码登录").Count()
	if err == nil && qrPrompt > 0 {
		return fmt.Errorf("cookie invalid: 扫码登录")
	}
	return nil
}

func SetInputFile(page playwright.Page, selector, filePath string) error {
	locator := page.Locator(selector)
	return locator.SetInputFiles(filePath)
}

func NormalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		normalized = append(normalized, tag)
	}
	return normalized
}

func TrimTitle(title string, max int) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return title
	}
	runes := []rune(title)
	if len(runes) <= max {
		return title
	}
	return string(runes[:max])
}

func getStringValue(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	value, exists := config[key]
	if !exists || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func getBoolValue(config map[string]interface{}, key string) bool {
	if config == nil {
		return false
	}
	value, exists := config[key]
	if !exists || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}
