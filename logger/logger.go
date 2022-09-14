package logger

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	utils "github.com/maxime915/glauncher/utils"
)

var (
	ErrDefaultLoggerUnavailable = errors.New("unable to open the default logger")
	ErrRelativePath             = errors.New("path must be absolute")
)

const DefaultLogFile = "~/.log/glauncher.log"

type Logger struct {
	*log.Logger
}

func (l Logger) FatalIfErr(err error) {
	if err != nil {
		l.Fatal(err.Error())
	}
}

func LogFile(path string) (io.Writer, error) {
	err := error(nil)

	if len(path) == 0 {
		path, err = utils.ResolvePath(DefaultLogFile)
		if err != nil {
			return nil, ErrDefaultLoggerUnavailable
		}
	}

	// expected to be a path to a file
	if !filepath.IsAbs(path) {
		return nil, ErrRelativePath
	}

	// create directory
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		return nil, fmt.Errorf("unable to make directory for log file: %w", err)
	}

	// create file
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to open log file: %w", err)
	}

	return f, nil
}

func LoggerTo(writers ...io.Writer) Logger {
	writer := io.MultiWriter(writers...)
	return Logger{log.New(writer, "ERROR (glauncher):", log.Ldate|log.Ltime|log.Lshortfile)}
}

func LoggerToFile(path string, addStderr bool) (Logger, error) {
	writer, err := LogFile(path)
	if err != nil {
		return Logger{}, err
	}

	if addStderr {
		return LoggerTo(writer, os.Stderr), nil
	}
	return LoggerTo(writer), nil
}

func LoggerToStderr() Logger {
	return LoggerTo(os.Stderr)
}
