package controllers

import (
	"net/http"

	"backend/internal/middlewares"
	"backend/internal/models/web"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type ReportController struct {
	Log           *logrus.Logger
	Service       *services.ReportService
	ApolloService *services.ApolloAPIService
}

func NewReportController(service *services.ReportService, log *logrus.Logger, apolloService *services.ApolloAPIService) *ReportController {
	return &ReportController{
		Log:           log,
		Service:       service,
		ApolloService: apolloService,
	}
}

func (controller *ReportController) CreateAll(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		panic(err)
	}

	// TODO: Make request
	reportRequest := &web.ReportRequest{
		File: fileHeader,
	}

	// TODO: Call Service
	response, err := controller.Service.CreateAll(ctx, reportRequest)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

func (controller *ReportController) FindAll(ctx *gin.Context) {
	var request web.DashboardRequest

	// TODO: Bind / Validate the request
	if err := ctx.ShouldBindQuery(&request); err != nil {
		//! Parse validation errors
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
				Code:   http.StatusBadRequest,
				Status: "Bad Request",
				Error:  err.Error(),
			})
			return
		}

		//! Custom error
		errorMessages := request.ParseError(errs)

		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  errorMessages,
		})
		return
	}

	// Success
	response, err := controller.Service.FindAll(ctx, &request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}
	ctx.JSON(response.Code, response)
}

func (controller *ReportController) IDPCount(ctx *gin.Context) {
	var request web.DashboardRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	// TODO: Call Service
	response, err := controller.Service.IDPCount(ctx, &request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(response.Code, response)
}

// ProgramExits lists the people who left an IDP program because of a promotion.
func (controller *ReportController) ProgramExits(ctx *gin.Context) {
	exits, err := controller.Service.ProgramExits()
	if err != nil {
		controller.Log.WithError(err).Error("Failed to load program exits")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Gagal memuat riwayat keluar program",
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   exits,
	})
}

func (controller *ReportController) FindBySeamanCode(ctx *gin.Context) {
	seamanCode := ctx.Param("seamanCode")
	if seamanCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Seaman code is required",
		})
		return
	}

	response, err := controller.Service.FindBySeamanCode(ctx, seamanCode)
	if err != nil {
		if err.Error() == "seaman code not found" {
			ctx.JSON(http.StatusNotFound, web.ErrorResponse{
				Code:   http.StatusNotFound,
				Status: "Not Found",
				Error:  "Seaman code not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(response.Code, response)
}

func (controller *ReportController) FindBySeafarerCode(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	if seafarerCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Seafarer code is required",
		})
		return
	}

	response, err := controller.Service.FindBySeafarerCode(ctx, seafarerCode)
	if err != nil {
		if err.Error() == "seafarer code not found" {
			ctx.JSON(http.StatusNotFound, web.ErrorResponse{
				Code:   http.StatusNotFound,
				Status: "Not Found",
				Error:  "Seafarer code not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	// The quiz pages are public, so an anonymous caller only learns who the code belongs to.
	if !middlewares.IsAuthenticated(ctx) {
		if full, ok := response.Data.(web.ReportData); ok {
			response.Data = web.ReportIdentity{
				ID:           full.ID,
				SeamanCode:   full.SeamanCode,
				SeafarerCode: full.SeafarerCode,
				Nama:         full.Nama,
				Jabatan:      full.Jabatan,
				VesselName:   full.VesselName,
			}
		}
	}

	ctx.JSON(response.Code, response)
}

func (controller *ReportController) GetTrainingData(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	if seafarerCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Seafarer code is required",
		})
		return
	}

	trainingData, err := controller.Service.GetTrainingDataBySeafarerCode(ctx, seafarerCode, controller.ApolloService)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Status: "Ok",
		Code:   http.StatusOK,
		Data:   trainingData,
	})
}

func (controller *ReportController) GetTrainingSummary(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	if seafarerCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Seafarer code is required",
		})
		return
	}

	summary, err := controller.Service.GetTrainingSummaryBySeafarerCode(ctx, seafarerCode, controller.ApolloService)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Status: "Ok",
		Code:   http.StatusOK,
		Data:   summary,
	})
}

func (controller *ReportController) RefreshPersonalData(ctx *gin.Context) {
	controller.Log.Info("Manual refresh personal data requested")

	if err := controller.Service.RefreshPersonalDataForAllReports(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"status":  "Internal Server Error",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"status":  "OK",
		"message": "Personal data refreshed successfully",
	})
}

