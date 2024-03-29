package main

import (
	"flag"
	"log"

	"github.com/Nerdbergev/Bergknecht/pkg/bergknecht"
	"github.com/Nerdbergev/Bergknecht/pkg/config"
)

var confpath string

func init() {
	flag.StringVar(&confpath, "c", "config.toml", "Path to config file")
}

func main() {
	flag.Parse()

	c, err := config.LoadConfig(confpath)
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	err = bergknecht.RunBot(c)
	if err != nil {
		log.Fatal("Error Running Bot:", err)
	}
}
