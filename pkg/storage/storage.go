package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

type FileType int

const (
	TOML FileType = iota
	JSON
)

type Manager struct {
	cachedPath    string
	peristentPath string
}

type Config struct {
	CachedPath     string
	PersistentPath string
}

func CreateStorageManager(c Config) *Manager {
	res := new(Manager)
	res.cachedPath = filepath.Join(c.CachedPath, "cached")
	res.peristentPath = filepath.Join(c.PersistentPath, "persistent")
	return res
}

func (sm *Manager) getFilenameandPath(Handlername, Filename string, persitent bool) (string, string) {
	Filename = path.Base(Filename)
	var path string
	if persitent {
		path = filepath.Join(sm.peristentPath, Handlername)
	} else {
		path = filepath.Join(sm.cachedPath, Handlername)
	}
	fullpath := filepath.Join(path, Filename)
	return fullpath, path
}

func (sm *Manager) DoesFileExist(Handlername, Filename string, persitent bool) bool {
	fullpath, _ := sm.getFilenameandPath(Handlername, Filename, persitent)
	if _, err := os.Stat(fullpath); err == nil {
		return true

	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		return false
	}
}

func (sm *Manager) GetFile(Handlername, Filename string, persitent bool) (*os.File, error) {
	fullpath, path := sm.getFilenameandPath(Handlername, Filename, persitent)
	os.MkdirAll(path, os.ModePerm)
	f, err := os.OpenFile(fullpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, errors.New("Error opening or creating file: " + err.Error())
	}
	return f, nil
}

func (sm *Manager) DeleteFile(Handlername, Filename string, persitent bool) error {
	fullpath, _ := sm.getFilenameandPath(Handlername, Filename, persitent)
	return os.Remove(fullpath)
}

func (sm *Manager) EncodeFile(Handlername, Filename string, FileType FileType, persistent bool, v interface{}) error {
	switch FileType {
	case TOML:
		return sm.encodeTOMLFile(Handlername, Filename, persistent, v)
	case JSON:
		return sm.encodeJSONFile(Handlername, Filename, persistent, v)
	}
	return nil
}

func (sm *Manager) encodeTOMLFile(Handlername, Filename string, persistent bool, v interface{}) error {
	f, err := sm.GetFile(Handlername, Filename, persistent)
	if err != nil {
		return errors.New("Error loading opening file: " + err.Error())
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	err = encoder.Encode(v)
	if err != nil {
		return errors.New("Error encoding file: " + err.Error())
	}
	return nil
}

func (sm *Manager) encodeJSONFile(Handlername, Filename string, persistent bool, v interface{}) error {
	f, err := sm.GetFile(Handlername, Filename, persistent)
	if err != nil {
		return errors.New("Error loading opening file: " + err.Error())
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	err = encoder.Encode(v)
	if err != nil {
		return errors.New("Error encoding file: " + err.Error())
	}
	return nil
}

func (sm *Manager) DecodeFile(Handlername, Filename string, FileType FileType, persistent bool, v interface{}) error {
	switch FileType {
	case TOML:
		return sm.decodeTOMLFile(Handlername, Filename, persistent, v)
	case JSON:
		return sm.decodeJSONFile(Handlername, Filename, persistent, v)
	}
	return nil
}

func (sm *Manager) decodeTOMLFile(Handlername, Filename string, persistent bool, v interface{}) error {
	f, err := sm.GetFile(Handlername, Filename, persistent)
	if err != nil {
		return errors.New("Error loading opening file: " + err.Error())
	}
	defer f.Close()
	decoder := toml.NewDecoder(f)
	err = decoder.Decode(v)
	if err != nil {
		return errors.New("Error decoding file: " + err.Error())
	}
	return nil
}

func (sm *Manager) decodeJSONFile(Handlername, Filename string, persistent bool, v interface{}) error {
	f, err := sm.GetFile(Handlername, Filename, persistent)
	if err != nil {
		return errors.New("Error loading opening file: " + err.Error())
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	err = decoder.Decode(v)
	if err != nil {
		return errors.New("Error decoding file: " + err.Error())
	}
	return nil
}

func (sm *Manager) DeleteCache() error {
	err := os.Remove(sm.cachedPath)
	if err != nil {
		return errors.New("Error removing cached Files: " + err.Error())
	}
	return nil
}
