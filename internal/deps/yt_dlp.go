package deps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	runtime "runtime"
	"sort"
	"strings"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/models"
	"Kairo/internal/utils"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type thumbnailEntry struct {
	URL string `json:"url"`
}

func (m *Manager) EnsureYtDlp() {
	if m.YtDlpPath != "" {
		if _, err := os.Stat(m.YtDlpPath); err == nil {
			return
		}
	}

	base, err := config.GetBinDir()
	if err != nil {
		return
	}

	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{"yt-dlp_macos", "yt-dlp"}
	case "windows":
		candidates = []string{"yt-dlp.exe"}
	default:
		candidates = []string{"yt-dlp"}
	}
	for _, name := range candidates {
		final := filepath.Join(base, name)
		if _, err := os.Stat(final); err == nil {
			m.YtDlpPath = final
			return
		}
		if m.AssetProvider != nil {
			if data, err := m.AssetProvider(name); err == nil && len(data) > 1_000_000 {
				tmp := final + ".tmp"
				if err := os.WriteFile(tmp, data, 0o755); err == nil {
					_ = os.Chmod(tmp, 0o755)
					if err := os.Rename(tmp, final); err == nil {
						m.YtDlpPath = final
						wailsRuntime.LogInfo(m.Ctx, "已从内置资源安装 yt-dlp")
						return
					}
				}
			}
		}
	}
	for _, name := range candidates {
		url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/" + name
		wailsRuntime.LogInfo(m.Ctx, "正在下载 yt-dlp: "+name)
		req, _ := http.NewRequest("GET", url, nil)
		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			wailsRuntime.LogError(m.Ctx, err.Error())
			continue
		}
		if resp.StatusCode != http.StatusOK {
			wailsRuntime.LogError(m.Ctx, fmt.Sprintf("下载失败: %s", resp.Status))
			resp.Body.Close()
			continue
		}
		tmp := filepath.Join(base, name+".tmp")
		f, err := os.Create(tmp)
		if err != nil {
			wailsRuntime.LogError(m.Ctx, err.Error())
			resp.Body.Close()
			continue
		}
		_, err = io.Copy(f, resp.Body)
		resp.Body.Close()
		_ = f.Close()
		if err != nil {
			wailsRuntime.LogError(m.Ctx, err.Error())
			_ = os.Remove(tmp)
			continue
		}
		final := filepath.Join(base, name)
		_ = os.Chmod(tmp, 0o755)
		if err := os.Rename(tmp, final); err != nil {
			wailsRuntime.LogError(m.Ctx, err.Error())
			_ = os.Remove(tmp)
			continue
		}
		m.YtDlpPath = final
		wailsRuntime.LogInfo(m.Ctx, "yt-dlp 下载完成")
		return
	}
}

func (m *Manager) GetVideoInfo(url string) (*models.VideoInfo, error) {
	if url == "" {
		return nil, errors.New("url is empty")
	}

	m.EnsureYtDlp()
	if m.YtDlpPath == "" {
		return nil, errors.New("yt-dlp not found")
	}

	if m.isLikelyPlaylist(url) {
		playlistInfo, err := m.getPlaylistInfo(url)
		if err != nil {
			return nil, err
		}
		if playlistInfo != nil && playlistInfo.SourceType == models.SourceTypePlaylist {
			return playlistInfo, nil
		}
	} else {
		info, err := m.getSingleVideoInfo(url)
		if err == nil {
			return info, nil
		}

		playlistInfo, pErr := m.getPlaylistInfo(url)
		if pErr == nil && playlistInfo != nil && playlistInfo.SourceType == models.SourceTypePlaylist {
			return playlistInfo, nil
		}
		return nil, err
	}

	return m.getSingleVideoInfo(url)
}

func (m *Manager) isLikelyPlaylist(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "list=") ||
		strings.Contains(lower, "/playlist") ||
		strings.Contains(lower, "/album") ||
		strings.Contains(lower, "/set/") ||
		strings.Contains(lower, "collection") ||
		strings.Contains(lower, "series") ||
		strings.Contains(lower, "season") ||
		strings.Contains(lower, "medialist")
}

func (m *Manager) getSingleVideoInfo(url string) (*models.VideoInfo, error) {
	args := []string{"--dump-json", "--no-playlist"}
	if proxy := config.GetProxyUrl(); proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	if ua := config.GetUserAgent(); ua != "" {
		args = append(args, "--user-agent", ua)
	}
	if ref := config.GetReferer(); ref != "" {
		args = append(args, "--referer", ref)
	}
	if config.GetGeoBypass() {
		args = append(args, "--geo-bypass")
	} else {
		args = append(args, "--no-geo-bypass")
	}

	if cookieArgs := config.GetCookieArgs(); len(cookieArgs) > 0 {
		args = append(args, cookieArgs...)
	}

	args = append(args, url)

	cmd := utils.CreateCommand(m.YtDlpPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	return m.ParseVideoInfo(output)
}

func (m *Manager) getPlaylistInfo(url string) (*models.VideoInfo, error) {
	args := []string{"--dump-single-json"}

	lowerURL := strings.ToLower(url)
	isBilibili := strings.Contains(lowerURL, "bilibili.com") || strings.Contains(lowerURL, "b23.tv")
	if !isBilibili {
		args = append(args, "--flat-playlist")
	}
	if proxy := config.GetProxyUrl(); proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	if ua := config.GetUserAgent(); ua != "" {
		args = append(args, "--user-agent", ua)
	}
	if ref := config.GetReferer(); ref != "" {
		args = append(args, "--referer", ref)
	}
	if config.GetGeoBypass() {
		args = append(args, "--geo-bypass")
	} else {
		args = append(args, "--no-geo-bypass")
	}

	if cookieArgs := config.GetCookieArgs(); len(cookieArgs) > 0 {
		args = append(args, cookieArgs...)
	}
	args = append(args, url)

	cmd := utils.CreateCommand(m.YtDlpPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var rawInfo struct {
		Title      string           `json:"title"`
		Thumbnail  string           `json:"thumbnail"`
		Thumbnails []thumbnailEntry `json:"thumbnails"`
		Type       string           `json:"_type"`
		Entries    []struct {
			Title      string           `json:"title"`
			Duration   float64          `json:"duration"`
			Thumbnail  string           `json:"thumbnail"`
			Thumbnails []thumbnailEntry `json:"thumbnails"`
			URL        string           `json:"url"`
			WebpageURL string           `json:"webpage_url"`
		} `json:"entries"`
	}

	if err := json.NewDecoder(strings.NewReader(string(output))).Decode(&rawInfo); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}
	wailsRuntime.EventsEmit(m.Ctx, "debug:notify", map[string]interface{}{
		"type": "playlist",
		"raw":  rawInfo,
	})

	if rawInfo.Type != "playlist" && len(rawInfo.Entries) == 0 {
		return nil, nil
	}

	var items []models.PlaylistItem
	for i, entry := range rawInfo.Entries {
		itemTitle := entry.Title
		if itemTitle == "" {
			if entry.WebpageURL != "" {
				itemTitle = entry.WebpageURL
			} else {
				itemTitle = entry.URL
			}
			if itemTitle == "" {
				itemTitle = fmt.Sprintf("Item %d", i+1)
			}
		}
		itemThumbnail := pickThumbnail(entry.Thumbnail, entry.Thumbnails)
		itemURL := strings.TrimSpace(entry.WebpageURL)
		if itemURL == "" {
			itemURL = strings.TrimSpace(entry.URL)
		}
		items = append(items, models.PlaylistItem{
			Index:     i + 1,
			Title:     itemTitle,
			Duration:  entry.Duration,
			Thumbnail: itemThumbnail,
			URL:       itemURL,
		})
	}

	playlistThumbnail := pickThumbnail(rawInfo.Thumbnail, rawInfo.Thumbnails)
	info := models.VideoInfo{
		Title:         rawInfo.Title,
		Thumbnail:     playlistThumbnail,
		SourceType:    models.SourceTypePlaylist,
		PlaylistItems: items,
		TotalItems:    len(items),
	}

	return &info, nil
}

func (m *Manager) ParseVideoInfo(output []byte) (*models.VideoInfo, error) {
	var rawInfo struct {
		Title      string                   `json:"title"`
		Thumbnail  string                   `json:"thumbnail"`
		Thumbnails []thumbnailEntry         `json:"thumbnails"`
		Duration   float64                  `json:"duration"`
		Formats    []map[string]interface{} `json:"formats"`
	}

	if err := json.NewDecoder(strings.NewReader(string(output))).Decode(&rawInfo); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}
	wailsRuntime.EventsEmit(m.Ctx, "debug:notify", map[string]interface{}{
		"type": "single",
		"raw":  rawInfo,
	})

	thumbnail := pickThumbnail(rawInfo.Thumbnail, rawInfo.Thumbnails)
	info := models.VideoInfo{
		Title:     rawInfo.Title,
		Thumbnail: thumbnail,
		Duration:  rawInfo.Duration,
	}

	getSize := func(f map[string]interface{}) int64 {
		if s, ok := f["filesize"].(float64); ok {
			return int64(s)
		}
		if s, ok := f["filesize_approx"].(float64); ok {
			return int64(s)
		}
		return 0
	}

	var bestAudioSize int64
	var bestAudioID string
	for _, f := range rawInfo.Formats {
		vcodec, _ := f["vcodec"].(string)
		acodec, _ := f["acodec"].(string)
		if vcodec != "none" || acodec == "none" {
			continue
		}

		if size := getSize(f); size > 0 {
			bestAudioSize = size
			if formatID, ok := f["format_id"].(string); ok && formatID != "" {
				bestAudioID = formatID
			}
		}
	}

	type formatSelection struct {
		Size int64
		ID   string
	}
	videoOnlyBestSize := make(map[int]formatSelection)
	combinedBestSize := make(map[int]formatSelection)

	for _, f := range rawInfo.Formats {
		vcodec, _ := f["vcodec"].(string)
		acodec, _ := f["acodec"].(string)

		isVideo := false
		if vcodec != "" && vcodec != "none" {
			isVideo = true
		} else if _, ok := f["width"]; ok {
			isVideo = true
		}

		if !isVideo {
			continue
		}

		if h, ok := f["height"].(float64); ok && h > 0 {
			height := int(h)
			size := getSize(f)
			formatID, _ := f["format_id"].(string)

			if size == 0 || formatID == "" {
				continue
			}

			if acodec == "none" || acodec == "" {
				videoOnlyBestSize[height] = formatSelection{Size: size, ID: formatID}
			} else {
				combinedBestSize[height] = formatSelection{Size: size, ID: formatID}
			}
		}
	}

	heightMap := make(map[int]bool)
	for h := range videoOnlyBestSize {
		heightMap[h] = true
	}
	for h := range combinedBestSize {
		heightMap[h] = true
	}

	var heights []int
	for h := range heightMap {
		heights = append(heights, h)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))

	var qualities []models.QualityOption

	for _, h := range heights {
		var videoSize, audioSize, totalSize int64
		var videoSizeStr, audioSizeStr string
		var formatID string

		if vSel, ok := videoOnlyBestSize[h]; ok && bestAudioSize > 0 && bestAudioID != "" {
			videoSize = vSel.Size
			audioSize = bestAudioSize
			totalSize = videoSize + audioSize
			videoSizeStr = utils.FormatBytes(videoSize)
			audioSizeStr = utils.FormatBytes(audioSize)
			formatID = vSel.ID + "+" + bestAudioID
		} else if cSel, ok := combinedBestSize[h]; ok {
			videoSize = cSel.Size
			audioSize = 0
			totalSize = cSel.Size
			videoSizeStr = utils.FormatBytes(videoSize)
			audioSizeStr = "-"
			formatID = cSel.ID
		} else if vSel, ok := videoOnlyBestSize[h]; ok {
			videoSize = vSel.Size
			audioSize = 0
			totalSize = vSel.Size
			videoSizeStr = utils.FormatBytes(videoSize)
			audioSizeStr = "-"
			formatID = vSel.ID
		} else {
			continue
		}

		qualities = append(qualities, models.QualityOption{
			Label:      fmt.Sprintf("%dp", h),
			Value:      fmt.Sprintf("%dp", h),
			FormatID:   formatID,
			VideoBytes: videoSize,
			AudioBytes: audioSize,
			TotalBytes: totalSize,
			VideoSize:  videoSizeStr,
			AudioSize:  audioSizeStr,
			TotalSize:  utils.FormatBytes(totalSize),
		})
	}

	qualities = append(qualities, models.QualityOption{
		Label:      "Audio Only",
		Value:      "audio",
		FormatID:   bestAudioID,
		VideoBytes: 0,
		AudioBytes: bestAudioSize,
		TotalBytes: bestAudioSize,
		VideoSize:  "-",
		AudioSize:  utils.FormatBytes(bestAudioSize),
		TotalSize:  utils.FormatBytes(bestAudioSize),
	})

	info.Qualities = qualities

	return &info, nil
}

func pickThumbnail(thumbnail string, thumbnails []thumbnailEntry) string {
	if thumbnail == "" && len(thumbnails) > 0 {
		for i := len(thumbnails) - 1; i >= 0; i-- {
			if thumbnails[i].URL != "" {
				thumbnail = thumbnails[i].URL
				break
			}
		}
	}
	return utils.EnsureHTTPS(strings.TrimSpace(thumbnail))
}
