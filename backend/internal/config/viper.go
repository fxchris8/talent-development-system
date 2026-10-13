package config

import (
	"log"

	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {
	config := viper.New()

	// Set default values (optional)
	config.SetDefault("DB_HOST", "localhost")
	config.SetDefault("DB_PORT", "3306")
	config.SetDefault("WEB_PORT", 8080)
	config.SetDefault("ENV", "development")
	// logrus levels: 2=error 3=warn 4=info 5=debug. Without a default an unset LOG_LEVEL means 0
	// (panic only) and every log line, including startup errors, is silently dropped.
	config.SetDefault("LOG_LEVEL", 4)
	config.SetDefault("SSO_BASE_URL", "")
	config.SetDefault("SSO_FRONTEND_URL", "")
	config.SetDefault("SSO_CLIENT_ID", "")
	config.SetDefault("SSO_CLIENT_SECRET", "")
	config.SetDefault("SSO_CALLBACK_URL", "")
	config.SetDefault("FRONTEND_URL", "http://localhost:3000")

	// Try to read .env file (optional for Docker)
	config.SetConfigFile(".env")
	config.AddConfigPath("./../")
	config.AddConfigPath("./")

	err := config.ReadInConfig()
	if err != nil {
		// .env file not found - this is OK in Docker
		log.Printf("Warning: .env file not found, using environment variables: %v", err)
	}

	// AutomaticEnv will override .env values with environment variables
	// This is crucial for Docker where we inject env via docker-compose
	config.AutomaticEnv()

	return config
}
