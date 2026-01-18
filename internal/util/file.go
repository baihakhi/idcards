package util

import (
	"io"
	"mime"
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

func ServeDownloadables(w http.ResponseWriter, r *http.Request, filePath, filename string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read first 512 bytes to detect MIME type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	mimeType := http.DetectContentType(buf[:n])

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	// Force download by setting generic MIME type for known displayable types
	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "text/") {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", mimeType)
	}

	file.Seek(0, 0)
	http.ServeContent(w, r, filename, time.Now(), file)
	return nil
}

func GetFileFormat(filePath string) string {
	return strings.ToLower(filepath.Ext(filePath))
}

func GetMimeType(filePath string) string {
	return mime.TypeByExtension(GetFileFormat(filePath))
}
