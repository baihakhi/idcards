package util

import (
	"os"
	"path/filepath"
	"strings"
)

func ImageWriter(data []byte, dir, name, format string) string {
	imgPath := filepath.Join(dir, name+format)
	webPath := strings.ReplaceAll(imgPath, `\`, `/`)
	os.WriteFile(imgPath, data, 0644)

	return webPath
}
