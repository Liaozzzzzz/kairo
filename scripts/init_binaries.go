package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "wails.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (wails.json not found)")
		}
		dir = parent
	}
}

func main() {
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Printf("Warning: %v. Using current directory as root.\n", err)
		projectRoot, _ = os.Getwd()
	}
	fmt.Printf("Project root: %s\n", projectRoot)

	var ffmpegURL, ytDlpURL string
	var ffmpegName, ytDlpName string
	var binDir string

	// Determine OS and set variables matching internal/downloader/downloader.go logic
	switch runtime.GOOS {
	case "darwin":
		// downloader.go uses "yt-dlp_macos" as first candidate for Darwin
		// URL construction in downloader.go: "https://.../download/" + name
		ffmpegURL = "https://evermeet.cx/ffmpeg/getrelease/zip"
		ytDlpName = "yt-dlp_macos"
		ytDlpURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/" + ytDlpName
		ffmpegName = "ffmpeg"
		binDir = filepath.Join(projectRoot, "assets", "bin", "darwin")
	case "windows":
		// downloader.go uses "yt-dlp.exe" for Windows
		ffmpegURL = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
		ytDlpName = "yt-dlp.exe"
		ytDlpURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/" + ytDlpName
		ffmpegName = "ffmpeg.exe"
		binDir = filepath.Join(projectRoot, "assets", "bin", "windows")
	default:
		fmt.Printf("Warning: Automatic binary download not configured for OS: %s\n", runtime.GOOS)
		return
	}

	fmt.Printf("Initializing binaries for %s...\n", runtime.GOOS)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		panic(fmt.Errorf("failed to create directory %s: %v", binDir, err))
	}

	// Download yt-dlp
	ytDlpPath := filepath.Join(binDir, ytDlpName)
	if !fileExists(ytDlpPath) {
		fmt.Printf("Downloading yt-dlp to %s...\n", ytDlpPath)
		if err := downloadFile(ytDlpURL, ytDlpPath); err != nil {
			panic(fmt.Errorf("failed to download yt-dlp: %v", err))
		}
		// Chmod is done inside downloadFile for temp, but we do it here on final just in case
		if err := os.Chmod(ytDlpPath, 0755); err != nil {
			panic(fmt.Errorf("failed to chmod yt-dlp: %v", err))
		}
		fmt.Println("yt-dlp downloaded successfully.")
	} else {
		fmt.Println("yt-dlp already exists.")
	}

	// Download ffmpeg
	ffmpegPath := filepath.Join(binDir, ffmpegName)
	if !fileExists(ffmpegPath) {
		fmt.Printf("Downloading ffmpeg to %s...\n", ffmpegPath)

		zipPath := filepath.Join(binDir, "ffmpeg_temp.zip")
		// We use downloadFileGeneric for zip because we don't need chmod on the zip itself
		if err := downloadFileGeneric(ffmpegURL, zipPath); err != nil {
			panic(fmt.Errorf("failed to download ffmpeg zip: %v", err))
		}
		defer os.Remove(zipPath)

		fmt.Println("Extracting ffmpeg...")
		if err := extractFileFromZip(zipPath, ffmpegName, binDir); err != nil {
			panic(fmt.Errorf("failed to extract ffmpeg: %v", err))
		}

		if err := os.Chmod(ffmpegPath, 0755); err != nil {
			panic(fmt.Errorf("failed to chmod ffmpeg: %v", err))
		}
		fmt.Println("ffmpeg downloaded successfully.")
	} else {
		fmt.Println("ffmpeg already exists.")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// downloadFile downloads to a temp file first, then renames to target.
// Matches robustness of downloader.go
func downloadFile(url, targetPath string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temp file
	tmpPath := targetPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	// Copy data
	_, err = io.Copy(out, resp.Body)
	out.Close() // Close before rename
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Chmod temp file
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Rename to final
	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// downloadFileGeneric is for non-executable files like zips (no chmod 755 needed usually, but safe to keep simple)
func downloadFileGeneric(url, targetPath string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractFileFromZip(zipPath, targetName, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Logic matches internal/utils/utils.go ExtractZip
		// Checks:
		// 1. suffix /targetName
		// 2. suffix \targetName
		// 3. exact match targetName
		// 4. suffix targetName.exe (if looking for ffmpeg but finding ffmpeg.exe - though here we usually pass full name)

		// Note: internal/utils uses "/" hardcoded for check 1.
		// We implement similar flexible checking.

		normalizedName := strings.ReplaceAll(f.Name, "\\", "/")
		targetNameBase := filepath.Base(targetName) // ffmpeg or ffmpeg.exe

		isMatch := false
		if strings.HasSuffix(normalizedName, "/"+targetNameBase) {
			isMatch = true
		} else if normalizedName == targetNameBase {
			isMatch = true
		}

		if isMatch {
			if f.FileInfo().IsDir() {
				continue
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			// Extract to destDir/targetName
			destPath := filepath.Join(destDir, targetName)

			// Use temp file for extraction too
			tmpDestPath := destPath + ".tmp"
			outFile, err := os.Create(tmpDestPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()
			if err != nil {
				os.Remove(tmpDestPath)
				return err
			}

			if err := os.Rename(tmpDestPath, destPath); err != nil {
				os.Remove(tmpDestPath)
				return err
			}

			return nil
		}
	}
	return fmt.Errorf("file %s not found in zip", targetName)
}
