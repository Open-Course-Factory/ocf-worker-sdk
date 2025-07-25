package generator

import (
	"log"
	"os"
)

// verboseLogger implémente un logger verbeux pour le SDK
type verboseLogger struct {
	logger *log.Logger
}

func NewVerboseLogger() *verboseLogger {
	return &verboseLogger{
		logger: log.New(os.Stderr, "[OCF] ", log.LstdFlags),
	}
}

func (l *verboseLogger) Debug(msg string, fields ...interface{}) {
	l.logger.Printf("[DEBUG] %s %v", msg, fields)
}

func (l *verboseLogger) Info(msg string, fields ...interface{}) {
	l.logger.Printf("[INFO] %s %v", msg, fields)
}

func (l *verboseLogger) Warn(msg string, fields ...interface{}) {
	l.logger.Printf("[WARN] %s %v", msg, fields)
}

func (l *verboseLogger) Error(msg string, fields ...interface{}) {
	l.logger.Printf("[ERROR] %s %v", msg, fields)
}
