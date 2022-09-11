package logger

import (
	"log"
	"os"
	"path/filepath"

	config "github.com/maxime915/glauncher/config"
	utils "github.com/maxime915/glauncher/utils"
)

var logger *log.Logger

const DefaultLogFile = "~/.log/glauncher.log"

func init() {
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal("unable to load config: ", err)
	}

	logFile := conf.LogFile

	if len(logFile) == 0 {
		logFile, err = utils.ResolvePath(DefaultLogFile)
		if err != nil {
			// use the default logger
			logger = log.Default()
			return
		}
	}

	if logFile == config.LogToStderr {
		logger = log.Default()
		return
	}

	// expected to be a path to a file
	if !filepath.IsAbs(logFile) {
		log.Fatal("log file must be absolute")
	}

	// create directory
	dir := filepath.Dir(logFile)
	err = os.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		log.Fatalf("unable to make directory to save log: %s", err.Error())
	}

	// create file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("error opening log file: ", err)
	}

	logger = log.New(f, "ERROR in glauncher:", log.Ldate|log.Ltime|log.Lshortfile)
}

func FatalIfErr(err error) {
	if err != nil {
		Fatal(err.Error())
	}
}

func Fatal(v ...any) {
	logger.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	logger.Fatalf(format, v...)
}

func Print(v ...any) {
	logger.Print(v...)
}
