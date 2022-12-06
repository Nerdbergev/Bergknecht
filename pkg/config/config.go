package config

import (
	"errors"
	"os"

	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"github.com/pelletier/go-toml"
	"go.uber.org/zap"
)

type Config struct {
	Serversettings  serverSettings
	LoggerSettings  zap.Config
	StorageSettings storage.Config
}

type serverSettings struct {
	Homserver string
	Username  string
	Password  string
	Rooms     []string
}

func LoadConfig(filepath string) (Config, error) {
	var res Config
	file, err := os.Open(filepath)
	if err != nil {
		return res, errors.New("Error opening file: " + err.Error())
	}
	defer file.Close()
	decoder := toml.NewDecoder(file)
	err = decoder.Decode(&res)
	if err != nil {
		return res, errors.New("Error decoding file: " + err.Error())
	}
	return res, nil
}
