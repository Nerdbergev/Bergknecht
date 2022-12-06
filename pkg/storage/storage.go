package storage

import (
	"errors"
	"os"
	"path/filepath"
)

type Manager struct {
	cachedPath    string
	peristentPath string
}

type Config struct {
	CachedPath    string
	PersistenPath string
}

func CreateStorageManager(c Config) *Manager {
	res := new(Manager)
	res.cachedPath = filepath.Join(c.CachedPath, "cached")
	res.peristentPath = filepath.Join(c.PersistenPath, "persistent")
	return res
}

func (sm *Manager) GetFile(Handlername, Filename string, persitent bool) (*os.File, error) {
	var path string
	if persitent {
		path = filepath.Join(sm.peristentPath, Handlername)
	} else {
		path = filepath.Join(sm.cachedPath, Handlername)
	}
	os.MkdirAll(path, os.ModePerm)
	path = filepath.Join(path, Filename)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.New("Error opening or creating file: " + err.Error())
	}
	return f, nil
}

func (sm *Manager) DeleteCache() error {
	err := os.Remove(sm.cachedPath)
	if err != nil {
		return errors.New("Error removing cached Files: " + err.Error())
	}
	return nil
}
