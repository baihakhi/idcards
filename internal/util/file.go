package util

import (
	"os"
	"path/filepath"
	"strings"
)

func ImageWriter(data []byte, dir, name, format string) (string, error) {
	imgPath := filepath.Join(dir, name+format)
	webPath := strings.ReplaceAll(imgPath, `\`, `/`)
	if err := os.WriteFile(imgPath, data, 0644); err != nil {
		return "", err
	}

	return webPath, nil
}
