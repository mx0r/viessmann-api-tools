package main

import (
	"os"
	"path/filepath"
)

type Cache interface {
	Get(key string) (string, error)
	Has(key string) bool
	Set(key string, value string) error
	Delete(key string) error
}

type FileCache struct {
	Path string
}

func (f FileCache) Get(key string) (string, error) {
	filePath := f.GetCacheFilePath(key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (f FileCache) Has(key string) bool {
	filePath := f.GetCacheFilePath(key)
	_, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return true
}

func (f FileCache) Set(key string, value string) error {
	filePath := f.GetCacheFilePath(key)
	err := os.WriteFile(filePath, []byte(value), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (f FileCache) Delete(key string) error {
	filePath := f.GetCacheFilePath(key)
	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

func (f FileCache) GetCacheFilePath(key string) string {
	return filepath.Join(f.Path, key+".cache")
}
