package logEditor

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var logFile *os.File

func init() {
	var err error
	logFile, err = os.OpenFile("RMS.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
}

func GetLogger() *logrus.Logger {

	// Setting up Logrus with a multi-writer
	myLogger := logrus.New()
	mw := io.MultiWriter(os.Stdout, logFile) // MultiWriter to log to stdout and file
	myLogger.SetOutput(mw)
	myLogger.Formatter = &logrus.JSONFormatter{}
	myLogger.Level = logrus.DebugLevel
	return myLogger
}

func Close() {
	defer logFile.Close()
}
