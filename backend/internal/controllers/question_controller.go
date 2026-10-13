package controllers

import (
	"backend/internal/helpers"
	"backend/internal/models/web"
	"backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type QuestionController struct {
	QuestionService services.QuestionService
	OptionService   services.OptionService
	DB              *gorm.DB
}

func NewQuestionController(questionService services.QuestionService, optionService services.OptionService, db *gorm.DB) *QuestionController {
	return &QuestionController{
		QuestionService: questionService,
		OptionService:   optionService,
		DB:              db,
	}
}

func (controller *QuestionController) Create(ctx *gin.Context) {
	var request web.QuestionCreateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	questionData, err := controller.QuestionService.Create(controller.DB, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "CREATED",
		Data:   questionData,
	})
}

func (controller *QuestionController) FindAll(ctx *gin.Context) {
	questionDataList, err := controller.QuestionService.FindAll(controller.DB)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   questionDataList,
	})
}

func (controller *QuestionController) FindById(ctx *gin.Context) {
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

	questionData, err := controller.QuestionService.FindById(controller.DB, questionId)
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
		Data:   questionData,
	})
}

func (controller *QuestionController) FindByRole(ctx *gin.Context) {
	role := ctx.Param("role")

	questionDataList, err := controller.QuestionService.FindByRole(controller.DB, role)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   questionDataList,
	})
}

func (controller *QuestionController) FindByAssessmentId(ctx *gin.Context) {
	assessmentIdStr := ctx.Param("assessmentId")
	assessmentId, err := strconv.ParseUint(assessmentIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "Invalid assessment ID",
		})
		return
	}

	questionDataList, err := controller.QuestionService.FindByAssessmentId(controller.DB, assessmentId)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	// Convert to questions with options format
	var questionsWithOptions []web.QuestionOptionResponse
	for _, question := range questionDataList {
		options, err := controller.OptionService.FindByQuestionId(controller.DB, question.QuestionID)
		if err != nil {
			options = []web.OptionData{}
		}

		questionWithOptions := web.QuestionOptionResponse{
			QuestionId:   question.QuestionID,
			QuestionText: question.QuestionText,
			QuestionType: question.QuestionType,
			AcceptableAnswers: question.AcceptableAnswers,
			Category:     helpers.PtrToString(question.Category),
			IsImage:      helpers.PtrToString(question.IsImage),
			ImageUrl:     helpers.PtrToString(question.ImageURL),
			AspectId:     question.AspectID,
			Options:      options,
		}
		questionsWithOptions = append(questionsWithOptions, questionWithOptions)
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   questionsWithOptions,
	})
}
func (controller *QuestionController) Update(ctx *gin.Context) {
	var request web.QuestionUpdateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	questionData, err := controller.QuestionService.Update(controller.DB, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   questionData,
	})
}

func (controller *QuestionController) Delete(ctx *gin.Context) {
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

	err = controller.QuestionService.Delete(controller.DB, questionId)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   "Question deleted successfully",
	})
}

func (controller *QuestionController) BulkDelete(ctx *gin.Context) {
	var request struct {
		QuestionIds []int `json:"questionIds" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	if len(request.QuestionIds) == 0 {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "No question IDs provided",
		})
		return
	}

	err := controller.QuestionService.BulkDelete(controller.DB, request.QuestionIds)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   "Questions deleted successfully",
	})
}

func (controller *QuestionController) UpdateAspect(ctx *gin.Context) {
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

	var request web.QuestionUpdateAspectRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	questionData, err := controller.QuestionService.UpdateAspect(controller.DB, questionId, request.AspectID)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   questionData,
	})
}

func (controller *QuestionController) BulkUpdateAspect(ctx *gin.Context) {
	var request web.QuestionBulkUpdateAspectRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  err.Error(),
		})
		return
	}

	if len(request.QuestionIds) == 0 {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "No question IDs provided",
		})
		return
	}

	err := controller.QuestionService.BulkUpdateAspect(controller.DB, request.QuestionIds, request.AspectID)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   "Questions aspect updated successfully",
	})
}