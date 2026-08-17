package platform

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsafeUpload = errors.New("unsafe upload")

type FileStore struct {
	Root     string
	MaxBytes int64
	Allowed  map[string]bool
}

func (s FileStore) Save(file *multipart.FileHeader) (string, error) {
	if file.Size <= 0 || file.Size > s.MaxBytes {
		return "", ErrUnsafeUpload
	}
	name := filepath.Base(file.Filename)
	if name != file.Filename || strings.Contains(name, "..") || !s.Allowed[strings.ToLower(filepath.Ext(name))] {
		return "", ErrUnsafeUpload
	}
	if err := os.MkdirAll(s.Root, 0o750); err != nil {
		return "", err
	}
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	target := filepath.Join(s.Root, name)
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, io.LimitReader(src, s.MaxBytes+1)); err != nil {
		return "", err
	}
	return target, nil
}
