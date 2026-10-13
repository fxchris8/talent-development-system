package controllers

import (
	"backend/internal/models/web"
	"backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OptionController struct {
	OptionService services.OptionService
	DB            *gorm.DB
}

func NewOptionController(optionService services.OptionService, db *gorm.DB) *OptionController {
	return &OptionController{
		OptionService: optionService,
		DB:            db,
	}
}

func (controller *OptionController) Create(ctx *gin.Context) {
	var request web.OptionCreateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	optionData, err := controller.OptionService.Create(controller.DB, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "CREATED",
		Data:   optionData,
	})
}

func (controller *OptionController) FindAll(ctx *gin.Context) {
	optionDataList, err := controller.OptionService.FindAll(controller.DB)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   optionDataList,
	})
}

func (controller *OptionController) FindById(ctx *gin.Context) {
	optionIdStr := ctx.Param("optionId")
	optionId, err := strconv.Atoi(optionIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid option ID",
		})
		return
	}

	optionData, err := controller.OptionService.FindById(controller.DB, optionId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, web.ErrorResponse{
			Code:   http.StatusNotFound,
			Status: "NOT FOUND",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   optionData,
	})
}

func (controller *OptionController) FindByQuestionId(ctx *gin.Context) {
	questionIdStr := ctx.Param("questionId")
	questionId, err := strconv.Atoi(questionIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid question ID",
		})
		return
	}

	optionDataList, err := controller.OptionService.FindByQuestionId(controller.DB, questionId)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   optionDataList,
	})
}

func (controller *OptionController) Update(ctx *gin.Context) {
	var request web.OptionUpdateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	optionData, err := controller.OptionService.Update(controller.DB, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   optionData,
	})
}

func (controller *OptionController) Delete(ctx *gin.Context) {
	optionIdStr := ctx.Param("optionId")
	optionId, err := strconv.Atoi(optionIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid option ID",
		})
		return
	}

	err = controller.OptionService.Delete(controller.DB, optionId)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   "Option deleted successfully",
	})
}