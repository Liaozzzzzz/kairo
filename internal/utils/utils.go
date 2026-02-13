package utils

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xi2/xz"
)

// FormatBytes formats bytes into a human readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParseSize parses a size string (e.g. "10.00MiB") into bytes
func ParseSize(sizeStr string) (float64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.TrimPrefix(sizeStr, "~")
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Find the split point between number and unit
	var i int
	for i = 0; i < len(sizeStr); i++ {
		if (sizeStr[i] < '0' || sizeStr[i] > '9') && sizeStr[i] != '.' {
			break
		}
	}

	if i == 0 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	numStr := sizeStr[:i]
	unitStr := sizeStr[i:]

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}

	if unitStr == "" {
		return val, nil
	}

	switch unitStr {
	case "B":
		return val, nil
	case "KiB", "K", "k":
		return val * 1024, nil
	case "MiB", "M", "m":
		return val * 1024 * 1024, nil
	case "GiB", "G", "g":
		return val * 1024 * 1024 * 1024, nil
	case "TiB", "T", "t":
		return val * 1024 * 1024 * 1024 * 1024, nil
	case "KB":
		return val * 1000, nil
	case "MB":
		return val * 1000 * 1000, nil
	case "GB":
		return val * 1000 * 1000 * 1000, nil
	case "TB":
		return val * 1000 * 1000 * 1000 * 1000, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unitStr)
	}
}

// FormatSize formats bytes into a human readable string (deprecated, use FormatBytes)
func FormatSize(size float64) string {
	const (
		KiB = 1024
		MiB = 1024 * 1024
		GiB = 1024 * 1024 * 1024
		TiB = 1024 * 1024 * 1024 * 1024
	)

	switch {
	case size >= TiB:
		return fmt.Sprintf("%.2fTiB", size/TiB)
	case size >= GiB:
		return fmt.Sprintf("%.2fGiB", size/GiB)
	case size >= MiB:
		return fmt.Sprintf("%.2fMiB", size/MiB)
	case size >= KiB:
		return fmt.Sprintf("%.2fKiB", size/KiB)
	default:
		return fmt.Sprintf("%.2fB", size)
	}
}

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

func SanitizeFileName(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, char, "_")
	}
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")
	if name == "" {
		name = "unnamed_file"
	}
	return name
}
