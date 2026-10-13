package controllers

import (
	"backend/internal/models/web"
	"backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SeafarerAssessmentController struct {
	Log                          *logrus.Logger
	DB                           *gorm.DB
	SeafarerAssessmentService    services.SeafarerAssessmentService
}

func NewSeafarerAssessmentController(
	log *logrus.Logger,
	db *gorm.DB,
	seafarerAssessmentService services.SeafarerAssessmentService,
) *SeafarerAssessmentController {
	return &SeafarerAssessmentController{
		Log:                       log,
		DB:                        db,
		SeafarerAssessmentService: seafarerAssessmentService,
	}
}

func (controller *SeafarerAssessmentController) FindAll(ctx *gin.Context) {
	seafarerAssessments, err := controller.SeafarerAssessmentService.FindAll(controller.DB)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   seafarerAssessments,
	})
}

func (controller *SeafarerAssessmentController) FindByID(ctx *gin.Context) {
	id := ctx.Param("id")
	seafarerAssessmentID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid seafarer assessment ID",
		})
		return
	}

	seafarerAssessment, err := controller.SeafarerAssessmentService.FindByID(controller.DB, seafarerAssessmentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, web.ErrorResponse{
				Code:   http.StatusNotFound,
				Status: "NOT FOUND",
				Error:  "Seafarer assessment not found",
			})
			return
		}
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   seafarerAssessment,
	})
}

func (controller *SeafarerAssessmentController) FindBySeafarerCode(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	if seafarerCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Seafarer code is required",
		})
		return
	}

	seafarerAssessments, err := controller.SeafarerAssessmentService.FindBySeafarerCode(controller.DB, seafarerCode)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   seafarerAssessments,
	})
}

func (controller *SeafarerAssessmentController) FindByAssessmentTypeID(ctx *gin.Context) {
	id := ctx.Param("assessmentTypeId")
	assessmentTypeID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid assessment type ID",
		})
		return
	}

	seafarerAssessments, err := controller.SeafarerAssessmentService.FindByAssessmentTypeID(controller.DB, assessmentTypeID)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   seafarerAssessments,
	})
}

func (controller *SeafarerAssessmentController) Assign(ctx *gin.Context) {
	var request web.SeafarerAssessmentCreateRequest
	err := ctx.ShouldBindJSON(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	seafarerAssessment, err := controller.SeafarerAssessmentService.AssignAssessment(controller.DB, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "CREATED",
		Data:   seafarerAssessment,
	})
}

func (controller *SeafarerAssessmentController) UpdateStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	seafarerAssessmentID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid seafarer assessment ID",
		})
		return
	}

	var request web.SeafarerAssessmentUpdateStatusRequest
	err = ctx.ShouldBindJSON(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	request.ID = seafarerAssessmentID

	seafarerAssessment, err := controller.SeafarerAssessmentService.UpdateStatus(controller.DB, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   seafarerAssessment,
	})
}

func (controller *SeafarerAssessmentController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	seafarerAssessmentID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid seafarer assessment ID",
		})
		return
	}

	err = controller.SeafarerAssessmentService.Delete(controller.DB, seafarerAssessmentID)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   nil,
	})
}

func (controller *SeafarerAssessmentController) CheckAssignment(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	assessmentTypeIDStr := ctx.Param("assessmentTypeId")

	if seafarerCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Seafarer code is required",
		})
		return
	}

	assessmentTypeID, err := strconv.ParseUint(assessmentTypeIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid assessment type ID",
		})
		return
	}

	result, err := controller.SeafarerAssessmentService.CheckAssignment(controller.DB, seafarerCode, assessmentTypeID)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	if !result.IsAssigned {
		ctx.JSON(http.StatusUnauthorized, web.ErrorResponse{
			Code:   http.StatusUnauthorized,
			Status: "UNAUTHORIZED",
			Error:  result.Message,
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   result,
	})
}

func (controller *SeafarerAssessmentController) CheckAssignmentWithRole(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	assessmentTypeIDStr := ctx.Param("assessmentTypeId")
	role := ctx.Param("role")

	if seafarerCode == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Seafarer code is required",
		})
		return
	}

	if role == "" {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Role is required",
		})
		return
	}

	assessmentTypeID, err := strconv.ParseUint(assessmentTypeIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid assessment type ID",
		})
		return
	}

	result, err := controller.SeafarerAssessmentService.CheckAssignmentWithRole(controller.DB, seafarerCode, assessmentTypeID, role)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	if !result.IsAssigned {
		ctx.JSON(http.StatusUnauthorized, web.ErrorResponse{
			Code:   http.StatusUnauthorized,
			Status: "UNAUTHORIZED",
			Error:  result.Message,
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   result,
	})
}

func (controller *SeafarerAssessmentController) IncrementAttempts(ctx *gin.Context) {
	seafarerCode := ctx.Param("seafarerCode")
	assessmentTypeID, err := strconv.ParseUint(ctx.Param("assessmentTypeId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid assessment type ID",
		})
		return
	}

	result, err := controller.SeafarerAssessmentService.IncrementAttempts(controller.DB, seafarerCode, assessmentTypeID)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   result,
	})
}

