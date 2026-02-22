package deps

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	runtime "runtime"
	"strings"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/utils"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (m *Manager) EnsureFFmpeg() {
	base, err := config.GetBinDir()
	if err != nil {
		return
	}
	ffmpegPath := filepath.Join(base, "ffmpeg")
	if runtime.GOOS == "windows" {
		ffmpegPath += ".exe"
	}

	if _, err := os.Stat(ffmpegPath); err == nil {
		m.FFmpegPath = ffmpegPath
		return
	}

	embeddedName := "ffmpeg"
	if runtime.GOOS == "windows" {
		embeddedName = "ffmpeg.exe"
	}

	if m.AssetProvider != nil {
		if data, err := m.AssetProvider(embeddedName); err == nil && len(data) > 0 {
			wailsRuntime.LogInfo(m.Ctx, "正在从内置资源安装 FFmpeg...")
			tmp := ffmpegPath + ".tmp"
			if err := os.WriteFile(tmp, data, 0o755); err == nil {
				_ = os.Chmod(tmp, 0o755)
				if err := os.Rename(tmp, ffmpegPath); err == nil {
					wailsRuntime.LogInfo(m.Ctx, "FFmpeg 内置资源安装成功")
					m.FFmpegPath = ffmpegPath
					return
				}
			}
		}
	}

	if path, err := exec.LookPath("ffmpeg"); err == nil {
		_ = os.Symlink(path, ffmpegPath)
		m.FFmpegPath = ffmpegPath
		return
	}

	wailsRuntime.LogInfo(m.Ctx, "正在下载 FFmpeg...")

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
		wailsRuntime.LogWarning(m.Ctx, "不支持的操作系统，无法自动下载 FFmpeg")
		return
	}

	tmpArchive := filepath.Join(base, "ffmpeg_archive.tmp")
	defer os.Remove(tmpArchive)

	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		wailsRuntime.LogError(m.Ctx, "FFmpeg 下载请求失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		wailsRuntime.LogError(m.Ctx, fmt.Sprintf("FFmpeg 下载失败: %s", resp.Status))
		return
	}

	out, err := os.Create(tmpArchive)
	if err != nil {
		wailsRuntime.LogError(m.Ctx, "无法创建临时文件: "+err.Error())
		return
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		wailsRuntime.LogError(m.Ctx, "写入文件失败: "+err.Error())
		return
	}

	wailsRuntime.LogInfo(m.Ctx, "FFmpeg 下载完成，正在解压...")

	var extractErr error
	if strings.HasSuffix(url, ".zip") {
		extractErr = utils.ExtractZip(tmpArchive, base, "ffmpeg")
	} else {
		extractErr = utils.ExtractTarXz(tmpArchive, base, "ffmpeg")
	}

	if extractErr != nil {
		wailsRuntime.LogError(m.Ctx, "FFmpeg 解压失败: "+extractErr.Error())
	} else {
		_ = os.Chmod(ffmpegPath, 0o755)
		wailsRuntime.LogInfo(m.Ctx, "FFmpeg 安装成功")
		m.FFmpegPath = ffmpegPath
	}
}
