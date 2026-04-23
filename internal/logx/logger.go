package logx

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

func New(filePath string) (*log.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	logger := log.New(io.MultiWriter(os.Stdout, file), "", log.Ldate|log.Ltime|log.Lmicroseconds)
	return logger, file.Close, nil
}
