package utils

import (
	"archive/tar"
	"archive/zip"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/xi2/xz"
)

func NormalizePath(dir string, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}

func ExtractZip(src, dest, targetFile string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/"+targetFile) || strings.HasSuffix(f.Name, "\\"+targetFile) || f.Name == targetFile || strings.HasSuffix(f.Name, targetFile+".exe") {
			// Found it
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			finalPath := filepath.Join(dest, filepath.Base(f.Name))
			outFile, err := os.Create(finalPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			return err
		}
	}
	return errors.New("ffmpeg binary not found in zip")
}

func ExtractTarXz(src, dest, targetFile string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	// xz.NewReader requires a dict cap. 0 means default.
	xzR, err := xz.NewReader(f, 0)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzR)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasSuffix(header.Name, "/"+targetFile) || header.Name == targetFile {
			finalPath := filepath.Join(dest, filepath.Base(header.Name))
			outFile, err := os.Create(finalPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tr)
			return err
		}
	}
	return errors.New("ffmpeg binary not found in tar.xz")
}

func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func GetSiteName(url string) string {
	url = strings.ToLower(url)
	if strings.Contains(url, "bilibili.com") || strings.Contains(url, "b23.tv") {
		return "bilibili"
	}
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return "youtube"
	}
	return "other"
}

func CreateCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	HideWindow(cmd)
	log.Printf("[CreateCommand] running command: %s %s", name, strings.Join(arg, " "))
	return cmd
}

func CreateCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	HideWindow(cmd)
	log.Printf("[CreateCommandContext] running command: %s %s", name, strings.Join(arg, " "))
	return cmd
}

// EnsureHTTPS replaces http:// with https:// in a URL
func EnsureHTTPS(url string) string {
	if strings.HasPrefix(url, "http://") {
		return strings.Replace(url, "http://", "https://", 1)
	}
	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}
	return url
}

func DetectLanguageFromText(text string) string {
	counts := map[string]int{}
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		switch {
		case unicode.In(r, unicode.Han):
			counts["zh-Hans"]++
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			counts["ja"]++
		case unicode.In(r, unicode.Hangul):
			counts["ko"]++
		case unicode.In(r, unicode.Cyrillic):
			counts["ru"]++
		case unicode.In(r, unicode.Arabic):
			counts["ar"]++
		case unicode.In(r, unicode.Devanagari):
			counts["hi"]++
		case unicode.In(r, unicode.Thai):
			counts["th"]++
		case unicode.In(r, unicode.Hebrew):
			counts["he"]++
		case unicode.In(r, unicode.Latin):
			counts["en"]++
		}
	}
	best := ""
	bestCount := 0
	for lang, count := range counts {
		if count > bestCount {
			best = lang
			bestCount = count
		}
	}
	if bestCount == 0 {
		return "und"
	}
	return best
}

func ExtractTextFromVTT(content string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "WEBVTT") {
			continue
		}
		if strings.Contains(trimmed, "-->") {
			continue
		}
		if isSubtitleIndex(trimmed) {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(trimmed)
	}
	return b.String()
}

func isSubtitleIndex(line string) bool {
	for _, r := range line {
		if r < '0' || r > '9' {
			return false
		}
	}
	return line != ""
}
