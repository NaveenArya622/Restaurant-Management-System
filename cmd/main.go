package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"rms/configuration"
	"rms/database"
	"rms/handler"
	"rms/logEditor"
	"rms/server"
	"syscall"
	"time"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

const shutDownTimeOut = 10 * time.Second

func main() {
	logrus := logEditor.GetLogger()
	config, err := configuration.GetConfig()
	logid, logErr := uuid.NewV4()
	if err != nil {
		logrus.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error("Error loading .env file")
	}
	if logErr != nil {
		logrus.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// create server instance
	//TODO :- please use database credentials on env file or set up in go env and access it by using os.getEnv() function **DONE**
	srv := server.SetupRoutes()
	if err := database.ConnectAndMigrate(
		config.DBHost,
		config.DBPort,
		config.DBName,
		config.DBUser,
		config.DBPassword,
		database.SSLModeDisable); err != nil {
		logrus.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Fatal("Failed to initialize and migrate database with error: %+v", err)
	}
	logrus.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": config,
	}).Info("migration successful!!")
	handler.RegisterAdmin(config)

	go func() {
		if err := srv.Run(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithFields(log.Fields{
				"time":        time.Now(),
				"uuid":        logid.String(),
				"requestBody": config,
			}).Error("Failed to run server with error: %+v", err)
		}
	}()
	logrus.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": config,
	}).Info("Server started at :8080")

	<-done

	logrus.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": config,
	}).Info("shutting down server")
	if err := database.ShutdownDatabase(); err != nil {
		logrus.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"data":        err,
			"requestBody": config,
		}).Error("failed to close database connection")
	}
	if err := srv.Shutdown(shutDownTimeOut); err != nil {
		logrus.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"data":        err,
			"requestBody": config,
		}).Panic("failed to gracefully shutdown server")
	}
	logEditor.Close()
}
