package main

import (
	"net/http"
	"os"
	"os/signal"
	"rms/database"
	"rms/handler"
	"rms/server"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const shutDownTimeOut = 10 * time.Second

func main() {
	err := godotenv.Load()
	if err != nil {
		logrus.Printf("Error loading .env file")
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// create server instance
	//TODO :- please use database credentials on env file or set up in go env and access it by using os.getEnv() function **DONE**
	srv := server.SetupRoutes()
	if err := database.ConnectAndMigrate(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		database.SSLModeDisable); err != nil {
		logrus.Errorf("Failed to initialize and migrate database with error: %+v", err)
	}
	logrus.Infof("migration successful!!")
	handler.RegisterAdmin()

	go func() {
		// user updated errors.Is() function
		if err := srv.Run(":8080"); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Failed to run server with error: %+v", err)
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
