package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CookiesInfo struct {
	Exists       bool      `json:"exists"`
	SizeBytes    int64     `json:"size_bytes,omitempty"`
	LastModified time.Time `json:"last_modified,omitempty"`
	AbsolutePath string    `json:"absolute_path,omitempty"`
}

func CheckCookiesFile() (*CookiesInfo, error) {
	path := "./pkg/pyConverter/cookies.txt"

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return &CookiesInfo{
			Exists: false,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error accediendo al archivo: %w", err)
	}

	return &CookiesInfo{
		Exists:       true,
		SizeBytes:    info.Size(),
		LastModified: info.ModTime(),
		AbsolutePath: absPath,
	}, nil
}
