package logger

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	*log.Logger
	file *os.File
}

func New(path string) (*Logger, error) {
	_ = os.Remove(path)

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	log := log.New(file, "", log.LstdFlags)

	return &Logger{
		Logger: log,
		file:   file,
	}, nil
}

func (l *Logger) Info(info string) {
	l.Println(info)
}
func (l *Logger) Infof(format string, args ...any) {
	formated := fmt.Sprintf(format, args...)
	l.Info(formated)
}

func (l *Logger) Error(err error) {
	text := fmt.Sprintf("Error: %s", err)
	l.Println(text)
}

func (l *Logger) Errorf(format string, args ...any) {
	formated := fmt.Sprintf(format, args...)
	l.Println("Error: " + formated)
}

func (l *Logger) Fatal(err error) {
	l.Fatal(err)
}

func (l *Logger) Close() {
	l.file.Close()
}
