package missioncontrol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var storeSyncFile = func(f *os.File) error {
	return f.Sync()
}

var storeSyncDir = syncStoreDir

func WriteStoreJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	data = append(data, '\n')
	return WriteStoreFileAtomic(path, data)
}

func WriteStoreFileAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer func() {
		if tempFile != nil {
			if closeErr := tempFile.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err = tempFile.Write(data); err != nil {
		return err
	}
	if err = storeSyncFile(tempFile); err != nil {
		return err
	}
	if err = tempFile.Close(); err != nil {
		return err
	}
	tempFile = nil
	if err = os.Rename(tempPath, path); err != nil {
		return err
	}
	if err = storeSyncDir(dir); err != nil {
		return err
	}
	return nil
}

func LoadStoreJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func syncStoreDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return storeSyncFile(dir)
}
