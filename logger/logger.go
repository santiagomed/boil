package blogger

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/santiagomed/boil/pkg/logger"
)

var (
	log  logger.Logger
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

		zerologLogger := zerolog.New(logFile).With().Timestamp().Logger()
		log = &ZerologAdapter{logger: &zerologLogger}
	})
}

// GetLogger returns the logger instance
func GetLogger() logger.Logger {
	return log
}

// ZerologAdapter adapts zerolog.Logger to our Logger interface
type ZerologAdapter struct {
	logger *zerolog.Logger
}

func (z *ZerologAdapter) Debug(msg string) { z.logger.Debug().Msg(msg) }
func (z *ZerologAdapter) Info(msg string)  { z.logger.Info().Msg(msg) }
func (z *ZerologAdapter) Warn(msg string)  { z.logger.Warn().Msg(msg) }
func (z *ZerologAdapter) Error(msg string) { z.logger.Error().Msg(msg) }
func (z *ZerologAdapter) Fatal(msg string) { z.logger.Fatal().Msg(msg) }
func (z *ZerologAdapter) WithField(key string, value interface{}) logger.Logger {
	newLogger := z.logger.With().Interface(key, value).Logger()
	return &ZerologAdapter{logger: &newLogger}
}
