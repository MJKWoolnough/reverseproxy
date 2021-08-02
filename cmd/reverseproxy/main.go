package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

type config struct {
	Port     uint16
	Username string
	Password string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var configFile, logFile string
	flag.StringVar(&configFile, "c", "", "config file")
	flag.StringVar(&logFile, "l", "-", "log file")
	flag.Parse()
	if configFile == "" {
		return errors.New("no config file specified")
	}
	var c config
	f, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("error while opening config file: %w", err)
	}
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return fmt.Errorf("error while decoding config file: %w", err)
	}
	f.Close()
	return nil
}
