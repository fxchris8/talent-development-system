package controllers

import (
	"backend/internal/helpers"
	"backend/internal/models/web"
	"backend/internal/services"
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)


type AssessmentController struct {
	Log *logrus.Logger
	DB *gorm.DB
	AssessmentService services.AssessmentService
	QuestionService services.QuestionService
	OptionService services.OptionService
}

// getAssessmentRoleByJabatan returns the assessment role based on job position hierarchy
func getAssessmentRoleByJabatan(jabatan string) string {
	jabatanUpper := strings.ToUpper(strings.TrimSpace(jabatan))

	// Define the role hierarchy mapping
	roleMap := map[string]string{
		"NAHKODA":   "nahkoda",
		"MUALIM_1":  "nahkoda",
		"MUALIM_2":  "mualim_1",
		"MUALIM_3":  "mualim_2",
		"KKM":       "kkm",
		"MASINIS_2": "kkm",
		"MASINIS_3": "masinis_2",
		"MASINIS_4": "masinis_3",
	}

	if role, exists := roleMap[jabatanUpper]; exists {
		return role
	}

	// If no mapping found, return lowercase version of jabatan
	return strings.ToLower(jabatan)
}

func NewAssessmentController(
	log *logrus.Logger,
	db *gorm.DB,
	assessmentService services.AssessmentService,
	questionService services.QuestionService,
	optionService services.OptionService,
) *AssessmentController {
	return &AssessmentController{
		Log: log,
		DB: db,
		AssessmentService: assessmentService,
		QuestionService: questionService,
		OptionService: optionService,
	}
}

func (controller *AssessmentController) FindByRole (ctx *gin.Context) {
	role := ctx.Param("role")

	assessment, err := controller.AssessmentService.FindByRole(controller.DB, role)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	questionData, err := controller.QuestionService.FindByRole(controller.DB, role)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	var questionsWithOptions []web.QuestionOptionResponse
	for _, question := range questionData {
		options, err := controller.OptionService.FindByQuestionId(controller.DB, question.QuestionID)
		if err != nil {
			options = []web.OptionData{}
		}

		questionWithOptions := web.QuestionOptionResponse{
			QuestionId: question.QuestionID,
			QuestionText: question.QuestionText,
			Category: helpers.PtrToString(question.Category),
			IsImage: helpers.PtrToString(question.IsImage),
			ImageUrl: helpers.PtrToString(question.ImageURL),
			QuestionType: question.QuestionType,
			AcceptableAnswers: question.AcceptableAnswers,
			Options: options,
		}
		questionsWithOptions = append(questionsWithOptions, questionWithOptions)
	}

	assessmentResponse := web.AssessmentResponse{
		AssessmentID:         assessment.AssessmentID,
		Role:                 assessment.Role,
		AssessmentName:       assessment.AssessmentName,
		UsingTimer:           assessment.UsingTimer,
		TimerLimitMinutes:    assessment.TimerLimitMinutes,
		TutorialContent:      assessment.TutorialContent,
		TutorialTimerMinutes: assessment.TutorialTimerMinutes,
		Questions:            questionsWithOptions,
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse {
		Code: http.StatusOK,
		Status: "Fetch assessment successfully",
		Data: assessmentResponse,
	})
}



func (controller *AssessmentController) FindByRolePublic(ctx *gin.Context) {
	role := ctx.Param("role")

	// Map role based on hierarchy (e.g., mualim_1 -> nahkoda, mualim_2 -> mualim_1, etc.)
	assessmentRole := getAssessmentRoleByJabatan(role)

	assessment, err := controller.AssessmentService.FindByRole(controller.DB, assessmentRole)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	questionData, err := controller.QuestionService.FindByRole(controller.DB, assessmentRole)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	var questionsWithOptions []web.QuestionOptionPublicResponse
	for _, question := range questionData {
		options, err := controller.OptionService.FindByQuestionId(controller.DB, question.QuestionID)
		if err != nil {
			options = []web.OptionData{}
		}

		// Convert OptionData to OptionPublicResponse
		var publicOptions []web.OptionPublicResponse
		for _, option := range options {
			publicOption := web.OptionPublicResponse{
				OptionID:     option.OptionID,
				OptionLetter: option.OptionLetter,
				OptionText:   option.OptionText,
				IsImage:      option.IsImage,
				ImageUrl:     option.ImageUrl,
			}
			publicOptions = append(publicOptions, publicOption)
		}

		questionWithOptions := web.QuestionOptionPublicResponse{
			QuestionId: question.QuestionID,
			QuestionText: question.QuestionText,
			Category: helpers.PtrToString(question.Category),
			IsImage: helpers.PtrToString(question.IsImage),
			ImageUrl: helpers.PtrToString(question.ImageURL),
			QuestionType: question.QuestionType,
			Options: publicOptions,
		}
		questionsWithOptions = append(questionsWithOptions, questionWithOptions)
	}

	// Randomize question order using Fisher-Yates shuffle
	rand.Shuffle(len(questionsWithOptions), func(i, j int) {
		questionsWithOptions[i], questionsWithOptions[j] = questionsWithOptions[j], questionsWithOptions[i]
	})

	assessmentResponse := web.AssessmentPublicResponse{
		AssessmentID:         assessment.AssessmentID,
		Role:                 assessment.Role,
		UsingTimer:           assessment.UsingTimer,
		TimerLimitMinutes:    assessment.TimerLimitMinutes,
		TutorialContent:      assessment.TutorialContent,
		TutorialTimerMinutes: assessment.TutorialTimerMinutes,
		Questions:            questionsWithOptions,
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse {
		Code: http.StatusOK,
		Status: "Fetch assessment successfully",
		Data: assessmentResponse,
	})
}

func (controller *AssessmentController) UpdateAssessment(ctx *gin.Context) {
	assessmentId, _ := strconv.Atoi(ctx.Param("assessmentId"))
	var request web.AssessmentDataRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code: http.StatusBadRequest,
			Status: "Bad Request",
			Error: err.Error(),
		})
		return
	}

	assessmentRequest := web.AssessmentUpdateRequest{
		AssessmentID:      uint64(assessmentId),
		Role:              request.Role,
		AssessmentName:    request.AssessmentName,
		UsingTimer:        request.UsingTimer,
		TimerLimitMinutes: request.TimerLimitMinutes,
	}

	assessmentData, err := controller.AssessmentService.Update(controller.DB, &assessmentRequest)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code: http.StatusOK,
		Status: "Update assessment successfully",
		Data: assessmentData,
	})
}

func (controller *AssessmentController) CreateAssessment(ctx *gin.Context) {
	var request web.AssessmentCreateRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code: http.StatusBadRequest,
			Status: "Bad Request",
			Error: err.Error(),
		})
		return
	}

	assessmentData, err := controller.AssessmentService.Create(controller.DB, &request)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, web.SuccessResponse{
		Code: http.StatusCreated,
		Status: "Create assessment successfully",
		Data: assessmentData,
	})
}

func (controller *AssessmentController) FindAllAssessments(ctx *gin.Context) {
	assessments, err := controller.AssessmentService.FindAll(controller.DB)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code: http.StatusOK,
		Status: "Fetch all assessments successfully",
		Data: assessments,
	})
}

func (controller *AssessmentController) DeleteAssessment(ctx *gin.Context) {
	assessmentId, err := strconv.ParseUint(ctx.Param("assessmentId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code: http.StatusBadRequest,
			Status: "Bad Request",
			Error: "Invalid assessment ID",
		})
		return
	}

	err = controller.AssessmentService.Delete(controller.DB, assessmentId)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code: http.StatusOK,
		Status: "Delete assessment successfully",
		Data: nil,
	})
}

func (controller *AssessmentController) UploadAssessmentImage(ctx *gin.Context) {
	file, err := ctx.FormFile("file")

	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code: http.StatusBadRequest,
			Status: "Bad Request",
			Error: err.Error(),
		})
		return
	}

	// Validasi size (10mb)
	if file.Size > 10*1024*1024 {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code: http.StatusBadRequest,
			Status: "Bad Request",
			Error: "File size exceeds 10MB limit",
		})
		return
	}
	
	// Generate unique filename dengan UUID
	filename := uuid.New().String() + filepath.Ext(file.Filename)
	path := filepath.Join("storage/assessments", filename)

	if err := ctx.SaveUploadedFile(file, path); err != nil {
		respondServiceError(ctx, err)
		return
	}

	imageUrl := fmt.Sprintf("/storage/assessments/%s", filename)

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code: http.StatusOK,
		Status: "Upload file successfully",
		Data: gin.H{
			"imageUrl": imageUrl,
		},
	})

}

func (controller *AssessmentController) FindUnassigned(ctx *gin.Context) {
	assessments, err := controller.AssessmentService.FindUnassigned(controller.DB)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code: http.StatusOK,
		Status: "Fetch unassigned assessments successfully",
		Data: assessments,
	})
}

func (controller *AssessmentController) AssignToType(ctx *gin.Context) {
	var request web.AssessmentAssignRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code: http.StatusBadRequest,
			Status: "Bad Request",
			Error: err.Error(),
		})
		return
	}

	err := controller.AssessmentService.AssignAssessment(controller.DB, request.AssessmentID, request.AssessmentTypeID)

	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code: http.StatusOK,
		Status: "Assign assessment successfully",
		Data: nil,
	})
}

func (controller *AssessmentController) UpdateTutorial(ctx *gin.Context) {
	assessmentId, err := strconv.ParseUint(ctx.Param("assessmentId"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid assessment ID",
		})
		return
	}

	var request web.AssessmentTutorialRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	assessmentData, err := controller.AssessmentService.UpdateTutorial(controller.DB, assessmentId, &request)
	if err != nil {
		respondServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Update tutorial successfully",
		Data:   assessmentData,
	})
}