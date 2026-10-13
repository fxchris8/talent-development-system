package controllers

import (
	"backend/internal/models/web"
	"backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AssessmentResultController struct {
	AssessmentResultService services.AssessmentResultService
	Log                     *logrus.Logger
	DB                      *gorm.DB
}

func NewAssessmentResultController(assessmentResultService services.AssessmentResultService, log *logrus.Logger, db *gorm.DB) *AssessmentResultController {
	return &AssessmentResultController{
		AssessmentResultService: assessmentResultService,
		Log:                     log,
		DB:                      db,
	}
}


func (controller *AssessmentResultController) Submit(ctx *gin.Context) {
	var request web.AssessmentSubmitRequest
	err := ctx.ShouldBindJSON(&request)
	if err != nil {
		controller.Log.WithError(err).Error("Error binding assessment submit request")
		webResponse := web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		}
		ctx.JSON(http.StatusBadRequest, webResponse)
		return
	}

	// Map role based on hierarchy before submitting
	request.Role = getAssessmentRoleByJabatan(request.Role)

	assessmentResult, err := controller.AssessmentResultService.SubmitAssessment(controller.DB, &request)
	if err != nil {
		controller.Log.WithError(err).Error("Error submitting assessment")
		respondServiceError(ctx, err)
		return
	}

	webResponse := web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   assessmentResult,
	}
	ctx.JSON(http.StatusOK, webResponse)
}

func (controller *AssessmentResultController) FindBySeafarerCode(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	if seafarerCode == "" {
		controller.Log.Error("Seafarer code is required")
		webResponse := web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Seafarer code is required",
		}
		ctx.JSON(http.StatusBadRequest, webResponse)
		return
	}

	assessmentResult, err := controller.AssessmentResultService.FindBySeafarerCode(controller.DB, seafarerCode)
	if err != nil {
		// Check if it's a "not found" error (case for seafarer who hasn't completed assessment)
		if err.Error() == "assessment result not found" {
			webResponse := web.SuccessResponse{
				Code:   http.StatusOK,
				Status: "OK",
				Data:   nil,
			}
			ctx.JSON(http.StatusOK, webResponse)
			return
		}

		// For other errors (database connection, etc.), return internal server error
		controller.Log.WithError(err).Error("Error finding assessment result by seafarer code")
		webResponse := web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "INTERNAL SERVER ERROR",
			Error:  "Internal server error",
		}
		ctx.JSON(http.StatusInternalServerError, webResponse)
		return
	}

	webResponse := web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   assessmentResult,
	}
	ctx.JSON(http.StatusOK, webResponse)
}

func (controller *AssessmentResultController) GetValueAssessmentReportBySeafarerCode(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	if seafarerCode == "" {
		controller.Log.Error("Seafarer code is required")
		webResponse := web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Seafarer code is required",
		}
		ctx.JSON(http.StatusBadRequest, webResponse)
		return
	}

	report, err := controller.AssessmentResultService.GetValueAssessmentReport(controller.DB, seafarerCode)
	if err != nil {
		controller.Log.WithError(err).Error("Error getting value assessment report")

		// Check if it's a "not found" error
		if err.Error() == "no completed assessment found" {
			webResponse := web.ErrorResponse{
				Code:   http.StatusNotFound,
				Status: "NOT FOUND",
				Error:  err.Error(),
			}
			ctx.JSON(http.StatusNotFound, webResponse)
			return
		}

		webResponse := web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "INTERNAL SERVER ERROR",
			Error:  err.Error(),
		}
		ctx.JSON(http.StatusInternalServerError, webResponse)
		return
	}

	webResponse := web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   report,
	}
	ctx.JSON(http.StatusOK, webResponse)
}
