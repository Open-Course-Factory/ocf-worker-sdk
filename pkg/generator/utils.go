package generator

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// isSlidevFile vérifie si un fichier est supporté par Slidev
func isSlidevFile(filename string) bool {
	supportedExts := []string{
		".md", ".css", ".scss", ".less", ".js", ".ts", ".vue",
		".json", ".yaml", ".yml", ".png", ".jpg", ".jpeg",
		".gif", ".svg", ".woff", ".woff2", ".ttf", ".eot",
	}

	ext := strings.ToLower(filepath.Ext(filename))
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}

	// Fichiers spéciaux
	base := strings.ToLower(filepath.Base(filename))
	if base == "readme" || base == "license" {
		return true
	}

	return false
}

// detectContentType détecte le type MIME
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md":
		return "text/markdown"
	case ".css":
		return "text/css"
	case ".scss":
		return "text/scss"
	case ".js":
		return "application/javascript"
	case ".ts":
		return "application/typescript"
	case ".vue":
		return "text/x-vue"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/yaml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// saveReaderToFile sauvegarde un reader dans un fichier
func saveReaderToFile(reader io.Reader, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

// extractZipFile extrait un fichier ZIP
func extractZipFile(zipPath, outputDir string) ([]string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var files []string

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		outputPath := filepath.Join(outputDir, file.Name)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return nil, err
		}

		if err := extractZipEntry(file, outputPath); err != nil {
			return nil, err
		}

		files = append(files, outputPath)
	}

	return files, nil
}

type ProgressReader struct {
	io.Reader
	bytesRead int64
	lastLog   time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.bytesRead += int64(n)

	// Log progress every 5 seconds
	if time.Since(pr.lastLog) > 5*time.Second {
		fmt.Printf("Downloaded: %d bytes (%.2f MB)\n", pr.bytesRead, float64(pr.bytesRead)/(1024*1024))

		pr.lastLog = time.Now()
	}

	return n, err
}
