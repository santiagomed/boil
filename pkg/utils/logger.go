package utils

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
)

var (
	log  zerolog.Logger
	once sync.Once
)

// InitLogger initializes the logger
func InitLogger() {
	once.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("Failed to get user home directory: " + err.Error())
		}

		boilDir := filepath.Join(homeDir, ".boil")
		err = os.MkdirAll(boilDir, 0755)
		if err != nil {
			panic("Failed to create .boil directory: " + err.Error())
		}

		logFile, err := os.OpenFile(filepath.Join(boilDir, "boil.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			panic("Failed to open log file: " + err.Error())
		}

		log = zerolog.New(logFile).With().Timestamp().Logger()
	})
}

// GetLogger returns the logger instance
func GetLogger() *zerolog.Logger {
	return &log
}
