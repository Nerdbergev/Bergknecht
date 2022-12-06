package storage

import (
	"errors"
	"os"
	"path/filepath"
)

func GetFile(Handlername, Filename string) (*os.File, error) {
	path := filepath.Join("storage", Handlername)
	os.MkdirAll(path, os.ModePerm)
	path = filepath.Join(path, Filename)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.New("Error opening or creating file: " + err.Error())
	}
	return f, nil
}
