package utils

import (
	"os"
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
		logFile, err := os.OpenFile("boil.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
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
