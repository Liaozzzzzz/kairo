package downloader

import (
	"context"
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

// Downloader manages yt-dlp binary and operations
type Downloader struct {
	BinPath string
	Ctx     context.Context
}

func NewDownloader(ctx context.Context) *Downloader {
	return &Downloader{
		Ctx: ctx,
	}
}

// readEmbedded reads a file from the embedded assets
// Note: This requires access to the embedded FS which is in main package.
// We might need to pass the FS or a reader function.
// For now, let's assume we pass a reader or byte slice provider.
// OR, we keep the embedded assets in main and pass them.
type AssetProvider func(name string) ([]byte, error)

func (d *Downloader) EnsureYtDlp(assetProvider AssetProvider) {
	if d.BinPath != "" {
		if _, err := os.Stat(d.BinPath); err == nil {
			return
		}
	}

	base, err := config.GetBinDir()
	if err != nil {
		return
	}

	// Ensure ffmpeg
	d.EnsureFFmpeg(base, assetProvider)

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
			d.BinPath = final
			return
		}
		if data, err := assetProvider(name); err == nil && len(data) > 1_000_000 {
			tmp := final + ".tmp"
			if err := os.WriteFile(tmp, data, 0o755); err == nil {
				_ = os.Chmod(tmp, 0o755)
				if err := os.Rename(tmp, final); err == nil {
					d.BinPath = final
					wailsRuntime.LogInfo(d.Ctx, "已从内置资源安装 yt-dlp")
					return
				}
			}
		}
	}
	for _, name := range candidates {
		url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/" + name
		wailsRuntime.LogInfo(d.Ctx, "正在下载 yt-dlp: "+name)
		req, _ := http.NewRequest("GET", url, nil)
		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			wailsRuntime.LogError(d.Ctx, err.Error())
			continue
		}
		if resp.StatusCode != http.StatusOK {
			wailsRuntime.LogError(d.Ctx, fmt.Sprintf("下载失败: %s", resp.Status))
			resp.Body.Close()
			continue
		}
		tmp := filepath.Join(base, name+".tmp")
		f, err := os.Create(tmp)
		if err != nil {
			wailsRuntime.LogError(d.Ctx, err.Error())
			resp.Body.Close()
			continue
		}
		_, err = io.Copy(f, resp.Body)
		resp.Body.Close()
		_ = f.Close()
		if err != nil {
			wailsRuntime.LogError(d.Ctx, err.Error())
			_ = os.Remove(tmp)
			continue
		}
		final := filepath.Join(base, name)
		_ = os.Chmod(tmp, 0o755)
		if err := os.Rename(tmp, final); err != nil {
			wailsRuntime.LogError(d.Ctx, err.Error())
			_ = os.Remove(tmp)
			continue
		}
		d.BinPath = final
		wailsRuntime.LogInfo(d.Ctx, "yt-dlp 下载完成")
		return
	}
}

func (d *Downloader) EnsureFFmpeg(base string, assetProvider AssetProvider) {
	ffmpegPath := filepath.Join(base, "ffmpeg")
	if runtime.GOOS == "windows" {
		ffmpegPath += ".exe"
	}

	// Check if already exists
	if _, err := os.Stat(ffmpegPath); err == nil {
		return
	}

	// Try to extract from embedded assets first
	embeddedName := "ffmpeg"
	if runtime.GOOS == "windows" {
		embeddedName = "ffmpeg.exe"
	}

	if data, err := assetProvider(embeddedName); err == nil && len(data) > 0 {
		wailsRuntime.LogInfo(d.Ctx, "正在从内置资源安装 FFmpeg...")
		tmp := ffmpegPath + ".tmp"
		if err := os.WriteFile(tmp, data, 0o755); err == nil {
			_ = os.Chmod(tmp, 0o755)
			if err := os.Rename(tmp, ffmpegPath); err == nil {
				wailsRuntime.LogInfo(d.Ctx, "FFmpeg 内置资源安装成功")
				return
			}
		}
	}

	// Try to find in path first
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		// Symlink or copy? Symlink is better
		_ = os.Symlink(path, ffmpegPath)
		return
	}

	wailsRuntime.LogInfo(d.Ctx, "正在下载 FFmpeg...")

	// Download from yt-dlp/FFmpeg-Builds
	url := ""
	switch runtime.GOOS {
	case "darwin":
		url = "https://evermeet.cx/ffmpeg/getrelease/zip"
	case "linux":
		url = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz"
	case "windows":
		url = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
	}

	if url == "" {
		wailsRuntime.LogWarning(d.Ctx, "不支持的操作系统，无法自动下载 FFmpeg")
		return
	}

	// Create temp file
	tmpArchive := filepath.Join(base, "ffmpeg_archive.tmp")
	defer os.Remove(tmpArchive)

	// Download
	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		wailsRuntime.LogError(d.Ctx, "FFmpeg 下载请求失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		wailsRuntime.LogError(d.Ctx, fmt.Sprintf("FFmpeg 下载失败: %s", resp.Status))
		return
	}

	out, err := os.Create(tmpArchive)
	if err != nil {
		wailsRuntime.LogError(d.Ctx, "无法创建临时文件: "+err.Error())
		return
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		wailsRuntime.LogError(d.Ctx, "写入文件失败: "+err.Error())
		return
	}

	wailsRuntime.LogInfo(d.Ctx, "FFmpeg 下载完成，正在解压...")

	// Extract
	var extractErr error
	if strings.HasSuffix(url, ".zip") {
		extractErr = utils.ExtractZip(tmpArchive, base, "ffmpeg")
	} else {
		extractErr = utils.ExtractTarXz(tmpArchive, base, "ffmpeg")
	}

	if extractErr != nil {
		wailsRuntime.LogError(d.Ctx, "FFmpeg 解压失败: "+extractErr.Error())
	} else {
		// Make executable
		_ = os.Chmod(ffmpegPath, 0o755)
		wailsRuntime.LogInfo(d.Ctx, "FFmpeg 安装成功")
	}
}

func (d *Downloader) GetVideoInfo(url string, assetProvider AssetProvider) (*models.VideoInfo, error) {
	if url == "" {
		return nil, errors.New("url is empty")
	}

	d.EnsureYtDlp(assetProvider)
	if d.BinPath == "" {
		return nil, errors.New("yt-dlp not found")
	}

	// Use --dump-json to get metadata
	args := []string{"--dump-json", "--no-playlist"}
	if proxy := config.GetProxyUrl(); proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	if cookieArgs := config.GetCookieArgs(url); len(cookieArgs) > 0 {
		args = append(args, cookieArgs...)
	}

	args = append(args, url)

	cmd := exec.Command(d.BinPath, args...)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	return d.ParseVideoInfo(output)
}

func (d *Downloader) ParseVideoInfo(output []byte) (*models.VideoInfo, error) {
	var rawInfo struct {
		Title     string                   `json:"title"`
		Thumbnail string                   `json:"thumbnail"`
		Duration  float64                  `json:"duration"`
		Formats   []map[string]interface{} `json:"formats"`
	}

	if err := json.NewDecoder(strings.NewReader(string(output))).Decode(&rawInfo); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	thumbnail := strings.TrimSpace(rawInfo.Thumbnail)
	if strings.HasPrefix(thumbnail, "http://") {
		thumbnail = "https://" + strings.TrimPrefix(thumbnail, "http://")
	} else if strings.HasPrefix(thumbnail, "//") {
		thumbnail = "https:" + thumbnail
	}
	info := models.VideoInfo{
		Title:     rawInfo.Title,
		Thumbnail: thumbnail,
		Duration:  rawInfo.Duration,
	}

	// Helper to get size
	getSize := func(f map[string]interface{}) int64 {
		if s, ok := f["filesize"].(float64); ok {
			return int64(s)
		}
		if s, ok := f["filesize_approx"].(float64); ok {
			return int64(s)
		}
		return 0
	}

	// Find best audio size (last is best assumption)
	var bestAudioSize int64
	var bestAudioID string
	for _, f := range rawInfo.Formats {
		// acodec != "none" and vcodec == "none"
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

	// Parse formats to extract unique heights and their best sizes
	// We trust yt-dlp's sorting (last is best) so we overwrite
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
			// fallback check
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

	// Collect all available heights
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

	// Add Audio Only option
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
