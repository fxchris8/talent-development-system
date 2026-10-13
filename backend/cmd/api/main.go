package main

import (
	"strconv"
	"strings"

	"backend/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	validate := config.NewValidator()
	app := config.NewGin(viperConfig, log)
	db := config.NewDatabase(viperConfig, log)

	// Bootstrap all configs
	config.Bootstrap(&config.BootstrapConfig{
		DB:       db,
		App:      app,
		Log:      log,
		Validate: validate,
		Config:   viperConfig,
	})

	// Get the Web Port
	webPort := viperConfig.GetInt("WEB_PORT")

	if viperConfig.GetString("ENV") == "production" {
		// Without a secret every login token could be forged; stop with a clear message
		// instead of running insecurely.
		if strings.TrimSpace(viperConfig.GetString("JWT_SECRET_KEY")) == "" {
			log.Fatal("JWT_SECRET_KEY kosong. Isi JWT_SECRET di file .env production lalu jalankan ulang.")
		}
		log.Info("ON PRODUCTION ENVIRONTMENT")
		gin.SetMode(gin.ReleaseMode)
	}

	err := app.Run(":" + strconv.Itoa(webPort))
	if err != nil {
		log.Fatal("Something went wrong")
	}
}
