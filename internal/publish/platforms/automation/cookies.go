package automation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type StorageState struct {
	Cookies []playwright.OptionalCookie `json:"cookies"`
	Origins []interface{}               `json:"origins"`
}

// ConvertToStorageState converts the cookie file content to Playwright StorageState JSON format.
// It supports JSON (pass-through), Netscape, and Header formats.
func ConvertToStorageState(content, targetDomain string) ([]byte, error) {
	strContent := string(content)
	trimmed := strings.TrimSpace(strContent)

	// 1. Check if it's already JSON StorageState
	if strings.HasPrefix(trimmed, "{") {
		// Verify if it's a valid JSON
		var state StorageState
		if err := json.Unmarshal([]byte(strContent), &state); err == nil && len(state.Cookies) > 0 {
			// It's already a valid StorageState, return as is (minified or formatted)
			return []byte(strContent), nil
		}
		// If unmarshal fails or cookies empty, it might be something else, but let's assume valid JSON for now if starts with {
		// Or it could be a simple JSON object that is NOT StorageState.
		// Let's return as is, assuming user knows what they are doing if they provide JSON.
		return []byte(strContent), nil
	}

	// 2. Check if it's a JSON Array (e.g. Exported from some plugins as [{}, {}])
	if strings.HasPrefix(trimmed, "[") {
		var cookies []playwright.OptionalCookie
		if err := json.Unmarshal([]byte(strContent), &cookies); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array cookies: %v", err)
		}
		return createStorageState(cookies)
	}

	var cookies []playwright.OptionalCookie
	var err error

	// 3. Check if it's Netscape format
	if strings.Contains(strContent, "# Netscape") || strings.Contains(strContent, "\t") {
		cookies, err = ParseNetscapeCookies(strContent)
	} else {
		// 4. Assume Header format
		cookies, err = ParseHeaderCookies(strContent, targetDomain)
	}

	if err != nil {
		return nil, err
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found in content")
	}

	return createStorageState(cookies)
}

func createStorageState(cookies []playwright.OptionalCookie) ([]byte, error) {
	state := StorageState{
		Cookies: cookies,
		Origins: []interface{}{},
	}
	return json.MarshalIndent(state, "", "  ")
}

func ParseHeaderCookies(content, domain string) ([]playwright.OptionalCookie, error) {
	var cookies []playwright.OptionalCookie

	// Normalize content: replace newlines with semicolons if it looks like line-separated k=v
	if !strings.Contains(content, ";") && strings.Contains(content, "\n") {
		content = strings.ReplaceAll(content, "\n", ";")
	}

	parts := strings.Split(content, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		name := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		cookie := playwright.OptionalCookie{
			Name:   name,
			Value:  value,
			Path:   playwright.String("/"),
			Secure: playwright.Bool(true),
		}

		if domain != "" {
			cookie.Domain = playwright.String(domain)
		}

		cookies = append(cookies, cookie)
	}

	return cookies, nil
}

func ParseNetscapeCookies(content string) ([]playwright.OptionalCookie, error) {
	var cookies []playwright.OptionalCookie
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}

		expires, _ := strconv.ParseInt(parts[4], 10, 64)
		secure := strings.ToUpper(parts[3]) == "TRUE"

		// Netscape format: domain, flag, path, secure, expiration, name, value
		// Playwright OptionalCookie: Name, Value, Domain, Path, Expires, HttpOnly, Secure, SameSite

		cookie := playwright.OptionalCookie{
			Name:    parts[5],
			Value:   parts[6],
			Domain:  playwright.String(parts[0]),
			Path:    playwright.String(parts[2]),
			Expires: playwright.Float(float64(expires)),
			Secure:  playwright.Bool(secure),
		}
		cookies = append(cookies, cookie)
	}

	return cookies, scanner.Err()
}
