package utils

import (
	"log"
	"os"
	"sync"
)

var (
	logger *log.Logger
	once   sync.Once
)

// InitLogger initializes the logger
func InitLogger() {
	once.Do(func() {
		logFile, err := os.OpenFile("boil.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		logger = log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	})
}

// GetLogger returns the logger instance
func GetLogger() *log.Logger {
	return logger
}
