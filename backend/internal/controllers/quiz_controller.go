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

type QuizController struct {
	Log         *logrus.Logger
	DB          *gorm.DB
	QuizService services.QuizService
}

func NewQuizController(log *logrus.Logger, db *gorm.DB, quizService services.QuizService) *QuizController {
	return &QuizController{
		Log:         log,
		DB:          db,
		QuizService: quizService,
	}
}

func (controller *QuizController) GetQuizData(ctx *gin.Context) {
	assessmentTypeId, err := strconv.ParseUint(ctx.Param("assessmentTypeId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid assessment type ID",
		})
		return
	}

	quizData, err := controller.QuizService.GetQuizData(controller.DB, assessmentTypeId)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Fetch quiz data successfully",
		Data:   quizData,
	})
}

func (controller *QuizController) SubmitQuiz(ctx *gin.Context) {
	var request web.QuizSubmitRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	// Capture SeamanCode from param if needed, or stick to body
	// Usually body is safer for structured data.

	result, err := controller.QuizService.SubmitQuiz(controller.DB, request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Quiz submitted successfully",
		Data:   result,
	})
}

func (controller *QuizController) GetQuizHistory(ctx *gin.Context) {
	// Filter by seamanCode if passed in query
	seamanCode := ctx.Query("seamanCode")

	attempts, err := controller.QuizService.GetQuizHistory(controller.DB, seamanCode)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Fetch quiz history successfully",
		Data:   attempts,
	})
}

func (controller *QuizController) GetQuizAttempt(ctx *gin.Context) {
	attemptIdStr := ctx.Param("attemptId")
	attemptId, err := strconv.Atoi(attemptIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid attempt ID",
		})
		return
	}

	attempt, err := controller.QuizService.GetQuizAttempt(controller.DB, attemptId)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Fetch quiz attempt details successfully",
		Data:   attempt,
	})
}
