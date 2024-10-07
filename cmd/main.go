package main

import (
	"net/http"
	"os"
	"os/signal"
	"rms/database"
	"rms/server"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

const shutDownTimeOut = 10 * time.Second

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// create server instance
	srv := server.SetupRoutes()

	/*
		DB credential should be in ENVIRONMENT VARIABLES and should be accessed using os.Getenv()
	*/

	if err := database.ConnectAndMigrate(
		"localhost", // DB_HOST
		"5434",      // DB_PORT
		"rms",       // DB_NAME
		"local",     // DB_USER
		"local",     // DB_PASS
		database.SSLModeDisable); err != nil {
		logrus.Panicf("Failed to initialize and migrate database with error: %+v", err)
	}
	logrus.Print("migration successful!!")

	go func() {
		if err := srv.Run(":8080"); err != nil && err != http.ErrServerClosed {
			logrus.Panicf("Failed to run server with error: %+v", err)
		}
	}()
	logrus.Print("Server started at :8080")

	<-done

	logrus.Info("shutting down server")
	if err := database.ShutdownDatabase(); err != nil {
		logrus.WithError(err).Error("failed to close database connection")
	}
	if err := srv.Shutdown(shutDownTimeOut); err != nil {
		logrus.WithError(err).Panic("failed to gracefully shutdown server")
	}
}
