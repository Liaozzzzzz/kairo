package video

import (
	"path/filepath"
	"strings"
)

type videoPathInfo struct {
	Dir      string
	BaseName string
	Ext      string
}

func buildVideoPathInfo(filePath string) videoPathInfo {
	ext := filepath.Ext(filePath)
	return videoPathInfo{
		Dir:      filepath.Dir(filePath),
		BaseName: strings.TrimSuffix(filepath.Base(filePath), ext),
		Ext:      ext,
	}
}

func buildOutputTemplate(filePath string) string {
	info := buildVideoPathInfo(filePath)
	return filepath.Join(info.Dir, info.BaseName+".%(ext)s")
}
