package util

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ImageWriter(data []byte, dir, name, format string) (string, error) {
	imgPath := filepath.Join(dir, name+format)
	webPath := strings.ReplaceAll(imgPath, `\`, `/`)
	if err := os.WriteFile(imgPath, data, 0644); err != nil {
		return "", err
	}

	return webPath, nil
}

func ServeDownloadables(w http.ResponseWriter, r *http.Request, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	mimeType := GetFileFormat(filePath)

	// Set the headers to force download
	w.Header().Set("Content-Disposition", "attachment; filename="+filePath)
	w.Header().Set("Content-Type", "application/"+mimeType)

	http.ServeContent(w, r, filePath, time.Now(), file)
	return nil
}
