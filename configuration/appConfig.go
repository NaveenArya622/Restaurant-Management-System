package configuration

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	SessionKey         string `envconfig:"SESSION_KEY" required:"true"`
	DBHost             string `envconfig:"DB_HOST" required:"true"`
	DBPort             string `envconfig:"DB_PORT" required:"true"`
	DBUser             string `envconfig:"DB_USER" required:"true"`
	DBPassword         string `envconfig:"DB_PASSWORD" required:"true"`
	DBName             string `envconfig:"DB_NAME" required:"true"`
	AdminName          string `envconfig:"ADMIN_NAME" required:"true"`
	AdminEmail         string `envconfig:"ADMIN_EMAIL" required:"true"`
	AdminFirstPassword string `envconfig:"ADMIN_FIRST_PASSWORD" required:"true"`
}

func GetConfig() (Config, error) {
	var config Config
	// Load .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Logger.Fatal("Error loading .env file")
	}

	// Process environment variables
	err = envconfig.Process("", &config)
	if err != nil {
		log.Logger.Fatal(err)
	}

	log.Logger.Printf("Config: %+v\n", config)
	return config, err
}
