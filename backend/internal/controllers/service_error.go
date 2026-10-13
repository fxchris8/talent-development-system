package controllers

import (
	"net/http"
	"strings"

	"backend/internal/models/web"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

// respondServiceError answers with the status that fits the error: 400 for invalid input,
// 403 for a seafarer without an assignment, 404 for a missing record, otherwise 500.
func respondServiceError(ctx *gin.Context, err error) {
	status := services.SubmissionErrorStatus(err)
	ctx.JSON(status, web.ErrorResponse{
		Code:   status,
		Status: strings.ToUpper(http.StatusText(status)),
		Error:  err.Error(),
	})
}
