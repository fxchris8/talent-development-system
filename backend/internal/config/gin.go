package config

import (
	"mime"
	"net/http"

	"backend/internal/models/web"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func PanicRecovery(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%s \"%s\" - Panic occured: %+v", c.Request.Method, c.Request.URL, err)

				// Return a unified error response
				c.JSON(http.StatusInternalServerError, web.ErrorResponse{
					Code:   http.StatusInternalServerError,
					Status: "Internal Server Error",
					// The details are in the log; they are not for the client.
					Error: "Terjadi kesalahan pada server",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

func NewGin(config *viper.Viper, log *logrus.Logger) *gin.Engine {
	if config.GetString("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// The slim runtime image has no MIME database, so name the generated file types ourselves;
	// otherwise the narration MP3 is served as application/octet-stream.
	_ = mime.AddExtensionType(".mp3", "audio/mpeg")
	_ = mime.AddExtensionType(".pdf", "application/pdf")
	_ = mime.AddExtensionType(".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")

	app := gin.Default()

	// CORS: credentials are only allowed for the frontend. An origin is accepted when it is
	// listed in FRONTEND_URL / CORS_ALLOWED_ORIGINS, or when it is the same site as the API
	// (same host, or same registrable domain such as app.x.co.id and api.x.co.id), which is
	// also the only case in which the browser sends the login cookie.
	cors := newCORSPolicy(config, log)
	app.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" && cors.allows(origin, c.Request) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Accept, Authorization, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			c.Writer.Header().Set("Access-Control-Max-Age", "43200") // 12 hours
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// app.Use(middlewares.PanicRecovery(log))
	app.Use(PanicRecovery(log))
	app.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":   http.StatusNotFound,
			"status": "Not Found",
		})
	})

	return app
}
