package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"Kairo/internal/db/schema"
	"Kairo/internal/utils"
)

type API struct {
	platform schema.PublishPlatform
}

func New(platform schema.PublishPlatform) *API {
	return &API{platform: platform}
}

func (o *API) GetPlatformName() string {
	return "OpenClaw"
}

func (o *API) ValidateConfig() error {
	if _, err := exec.LookPath("openclaw"); err != nil {
		return fmt.Errorf("openclaw executable not found in PATH. Please install it: npm install -g openclaw")
	}
	return nil
}

func (o *API) ValidateAccount(ctx context.Context, cookiePath string) error {
	input := map[string]interface{}{
		"command":     "validate",
		"cookie_path": cookiePath,
	}

	inputJson, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %v", err)
	}

	cmd := exec.CommandContext(ctx, "openclaw", "skill", "douyin-uploader", "--input", string(inputJson))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openclaw validate failed: %v, output: %s", err, string(output))
	}

	return nil
}

func (o *API) UploadVideo(ctx context.Context, title, description string, tags []string, videoPath, accountCookiePath string) (string, error) {
	// Construct the natural language message for the agent
	var msgBuilder strings.Builder
	// Use skill name to determine platform, do not hardcode platform name in prompt
	msgBuilder.WriteString(fmt.Sprintf("Use douyin-uploader to upload video %s", videoPath))

	if title != "" {
		msgBuilder.WriteString(fmt.Sprintf(", title: %s", title))
	}
	if description != "" {
		msgBuilder.WriteString(fmt.Sprintf(", description: %s", description))
	}
	if len(tags) > 0 {
		msgBuilder.WriteString(fmt.Sprintf(", tags: %s", strings.Join(tags, ",")))
	}
	if accountCookiePath != "" {
		msgBuilder.WriteString(fmt.Sprintf(", cookie: %s", accountCookiePath))
	}

	message := msgBuilder.String()

	// Execute the openclaw agent command
	cmd := utils.CreateCommandContext(ctx, "openclaw", "agent", "--agent", "main", "--message", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("openclaw upload failed: %v, output: %s", err, string(output))
	}

	// Assuming the output contains the video ID or just success.
	// We can return a generic ID or parse the output if the skill returns JSON.
	// For now, return a timestamp-based ID.
	return fmt.Sprintf("openclaw_%d", time.Now().Unix()), nil
}
