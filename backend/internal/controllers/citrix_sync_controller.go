package controllers

import (
	"errors"
	"net/http"

	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CitrixSyncController struct {
	Log     *logrus.Logger
	Service *services.CronService
}

func NewCitrixSyncController(service *services.CronService, log *logrus.Logger) *CitrixSyncController {
	return &CitrixSyncController{
		Log:     log,
		Service: service,
	}
}

// Sync runs the CITRIX sync right away instead of waiting for the next cron tick.
func (controller *CitrixSyncController) Sync(ctx *gin.Context) {
	controller.Log.Info("Manual CITRIX sync requested")

	if err := controller.Service.RunCitrixSync(); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrCitrixSyncRunning) {
			status = http.StatusConflict
		}
		ctx.JSON(status, gin.H{
			"code":    status,
			"status":  http.StatusText(status),
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"status":  "OK",
		"message": "Data CITRIX berhasil disinkronkan",
	})
}
