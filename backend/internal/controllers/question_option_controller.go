package controllers

import (
	"backend/internal/helpers"
	"backend/internal/models/web"
	"backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type QuestionOptionController struct {
	Log             *logrus.Logger
	DB              *gorm.DB
	QuestionService services.QuestionService
	OptionService   services.OptionService
}

func NewQuestionOptionController(
	db *gorm.DB,
	questionService services.QuestionService,
	optionService services.OptionService,
	log *logrus.Logger,
) *QuestionOptionController {
	return &QuestionOptionController{
		Log:             log,
		DB:              db,
		QuestionService: questionService,
		OptionService:   optionService,
	}
}

// GET /api/questions-with-options?role=xxx
func (controller *QuestionOptionController) FindAllQuestionsWithOptions(ctx *gin.Context) {
	var request web.QuestionWithOptionsRequest

	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	controller.Log.WithField("role", request.Role).Info("Fetching questions with options for role")

	// Get all questions for the role
	questionsData, err := controller.QuestionService.FindByRole(controller.DB, request.Role)
	if err != nil {
		controller.Log.WithError(err).Error("Failed to fetch questions")
		respondServiceError(ctx, err)
		return
	}

	// Get options for each question
	var questionsWithOptions []web.QuestionWithOptionsData
	for _, question := range questionsData {
		options, err := controller.OptionService.FindByQuestionId(controller.DB, question.QuestionID)
		if err != nil {
			controller.Log.WithError(err).WithField("questionId", question.QuestionID).Warn("Failed to fetch options for question")
			// Continue with empty options instead of failing
			options = []web.OptionData{}
		}

		questionWithOptions := web.QuestionWithOptionsData{
			QuestionId:        question.QuestionID,
			Role:              question.Role,
			QuestionText:      question.QuestionText,
			QuestionType:      question.QuestionType,
			AcceptableAnswers: question.AcceptableAnswers,
			Category:          helpers.PtrToString(question.Category),
			IsImage:           helpers.PtrToString(question.IsImage),
			ImageUrl:          helpers.PtrToString(question.ImageURL),
			Options:           options,
		}

		questionsWithOptions = append(questionsWithOptions, questionWithOptions)
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   web.QuestionWithOptionsListResponse{Data: questionsWithOptions},
	})
}

// POST /api/questions-with-options
func (controller *QuestionOptionController) CreateQuestionWithOptions(ctx *gin.Context) {
	var request web.CreateQuestionWithOptionsRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	controller.Log.WithField("request", request).Info("Creating question with options")
	

	// START TRANSACTION
	tx := controller.DB.Begin()
	if tx.Error != nil {
		controller.Log.WithError(tx.Error).Error("Failed to begin transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to begin transaction",
		})
		return
	}

	// Ensure rollback on any error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			controller.Log.Error("Transaction rolled back due to panic")
		}
	}()

	// Update question within transaction
	var aspectID *int64
	if request.AspectID != nil {
		val := int64(*request.AspectID)
		aspectID = &val
	}

	// Create question within transaction
	questionRequest := web.QuestionCreateRequest{
		Role:              request.Role,
		AssessmentID:      request.AssessmentID,
		QuestionText:      request.QuestionText,
		QuestionType:      request.QuestionType,
		AcceptableAnswers: request.AcceptableAnswers,
		Category:          helpers.StringToPtr(request.Category),
		IsImage:           helpers.StringToPtr(request.IsImage),
		ImageURL:          helpers.StringToPtr(request.ImageUrl),
		AspectID:          aspectID,
	}

	questionData, err := controller.QuestionService.Create(tx, &questionRequest)
	if err != nil {
		tx.Rollback() // ROLLBACK
		controller.Log.WithError(err).Error("Failed to create question - transaction rolled back")
		respondServiceError(ctx, err)
		return
	}

	// Create ALL options within same transaction
	var createdOptions []web.OptionData
	for i, optionReq := range request.Options {
		optionRequest := web.OptionCreateRequest{
			QuestionID:   questionData.QuestionID,
			OptionLetter: optionReq.OptionLetter,
			OptionText:   optionReq.OptionText,
			Score:        optionReq.Score,
			IsImage:      optionReq.IsImage,
			ImageURL:     optionReq.ImageURL,
		}

		optionData, err := controller.OptionService.Create(tx, &optionRequest)
		if err != nil {
			tx.Rollback() // ROLLBACK ALL
			controller.Log.WithError(err).WithField("optionIndex", i).WithField("questionId", questionData.QuestionID).Error("Failed to create option - transaction rolled back")
			ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
				Code:   http.StatusInternalServerError,
				Status: "Internal Server Error",
				Error:  "Failed to create option: " + err.Error(),
			})
			return
		}

		createdOptions = append(createdOptions, optionData)
	}

	// 🎉 COMMIT TRANSACTION - All or nothing!
	if err := tx.Commit().Error; err != nil {
		controller.Log.WithError(err).Error("Failed to commit transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to commit transaction",
		})
		return
	}

	controller.Log.WithField("questionId", questionData.QuestionID).WithField("optionsCount", len(createdOptions)).Info("Successfully created question with options")

	// Prepare response
	questionWithOptions := web.QuestionWithOptionsData{
		QuestionId:        questionData.QuestionID,
		Role:              questionData.Role,
		QuestionText:      questionData.QuestionText,
		QuestionType:      questionData.QuestionType,
		AcceptableAnswers: questionData.AcceptableAnswers,
		Category:          helpers.PtrToString(questionData.Category),
		IsImage:           helpers.PtrToString(questionData.IsImage),
		ImageUrl:          helpers.PtrToString(questionData.ImageURL),
		Options:           createdOptions,
	}

	ctx.JSON(http.StatusCreated, web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "Created",
		Data:   questionWithOptions,
	})
}

// Put /api/question-with-options
func (controller *QuestionOptionController) UpdateQuestionWithOptions(ctx *gin.Context) {
	questionId, _ := strconv.Atoi(ctx.Param("questionId"))
	var request web.UpdateQuestionWithOptionsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	controller.Log.WithField("request", request).Info("Updating question with options")

	// START TRANSACTION
	tx := controller.DB.Begin()
	if tx.Error != nil {
		controller.Log.WithError(tx.Error).Error("Failed to begin transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to begin transaction",
		})
		return
	}

	// Ensure rollback on any error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			controller.Log.Error("Transaction rolled back due to panic")
		}
	}()

	// Update question within transaction
	var aspectID *int64
	if request.AspectID != nil {
		val := int64(*request.AspectID)
		aspectID = &val
	}

	questionRequest := web.QuestionUpdateRequest{
		QuestionID:   questionId,
		Role:         request.Role,
		AssessmentID: request.AssessmentID,
		QuestionText:      request.QuestionText,
		QuestionType:      request.QuestionType,
		AcceptableAnswers: request.AcceptableAnswers,
		Category:          request.Category,
		IsImage:           request.IsImage,
		ImageURL:          request.ImageURL,
		AspectID:          aspectID,
	}

	questionData, err := controller.QuestionService.Update(tx, &questionRequest)
	if err != nil {
		tx.Rollback() // ROLLBACK
		controller.Log.WithError(err).Error("Failed to update question - transaction rolled back")
		respondServiceError(ctx, err)
		return
	}

	// Handle options operations (create, update, delete) within transaction
	var resultOptions []web.OptionData

	// Step 1: Get current options for this question to compare what needs to be deleted
	currentOptions, err := controller.OptionService.FindByQuestionId(tx, questionId)
	if err != nil {
		tx.Rollback()
		controller.Log.WithError(err).Error("Failed to fetch current options - transaction rolled back")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to fetch current options: " + err.Error(),
		})
		return
	}

	// Step 2: Create map of incoming option IDs for easy lookup
	incomingOptionIDs := make(map[int]bool)
	for _, optionReq := range request.Options {
		if optionReq.OptionID != nil && optionReq.Action != "delete" {
			incomingOptionIDs[*optionReq.OptionID] = true
		}
	}

	// Step 3: Delete options that are no longer in the request (implicit delete)
	for _, currentOption := range currentOptions {
		if !incomingOptionIDs[currentOption.OptionID] {
			err := controller.OptionService.Delete(tx, currentOption.OptionID)
			if err != nil {
				tx.Rollback()
				controller.Log.WithError(err).WithField("optionId", currentOption.OptionID).Error("Failed to delete option - transaction rolled back")
				ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
					Code:   http.StatusInternalServerError,
					Status: "Internal Server Error",
					Error:  "Failed to delete option: " + err.Error(),
				})
				return
			}
			controller.Log.WithField("optionId", currentOption.OptionID).Info("Option deleted (not in new list)")
		}
	}

	// Step 4: Process each option in the request
	for i, optionReq := range request.Options {
		switch optionReq.Action {
		case "delete":
			// Explicit delete operation
			if optionReq.OptionID != nil {
				err := controller.OptionService.Delete(tx, *optionReq.OptionID)
				if err != nil {
					tx.Rollback()
					controller.Log.WithError(err).WithField("optionIndex", i).WithField("optionId", *optionReq.OptionID).Error("Failed to delete option - transaction rolled back")
					ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
						Code:   http.StatusInternalServerError,
						Status: "Internal Server Error",
						Error:  "Failed to delete option: " + err.Error(),
					})
					return
				}
				controller.Log.WithField("optionId", *optionReq.OptionID).Info("Option deleted explicitly")
			}

		case "update":
			// Update existing option
			if optionReq.OptionID == nil {
				tx.Rollback()
				controller.Log.WithField("optionIndex", i).Error("Update action requires optionId - transaction rolled back")
				ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
					Code:   http.StatusBadRequest,
					Status: "Bad Request",
					Error:  "Update action requires optionId",
				})
				return
			}

			optionRequest := web.OptionUpdateRequest{
				OptionID:     *optionReq.OptionID,
				QuestionID:   questionId,
				OptionLetter: optionReq.OptionLetter,
				OptionText:   optionReq.OptionText,
				Score:        optionReq.Score,
				IsImage:      0, // default value
				ImageURL:     "",
			}

			if optionReq.IsImage != nil {
				optionRequest.IsImage = *optionReq.IsImage
			}

			if optionReq.ImageURL != nil {
				optionRequest.ImageURL = *optionReq.ImageURL
			}

			optionData, err := controller.OptionService.Update(tx, &optionRequest)
			if err != nil {
				tx.Rollback()
				controller.Log.WithError(err).WithField("optionIndex", i).WithField("optionId", *optionReq.OptionID).Error("Failed to update option - transaction rolled back")
				ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
					Code:   http.StatusInternalServerError,
					Status: "Internal Server Error",
					Error:  "Failed to update option: " + err.Error(),
				})
				return
			}
			resultOptions = append(resultOptions, optionData)

		case "create":
			// Create new option
			optionRequest := web.OptionCreateRequest{
				QuestionID:   questionId,
				OptionLetter: optionReq.OptionLetter,
				OptionText:   optionReq.OptionText,
				Score:        optionReq.Score,
				IsImage:      0, // default value
				ImageURL:     "",
			}

			if optionReq.IsImage != nil {
				optionRequest.IsImage = *optionReq.IsImage
			}

			if optionReq.ImageURL != nil {
				optionRequest.ImageURL = *optionReq.ImageURL
			}

			optionData, err := controller.OptionService.Create(tx, &optionRequest)
			if err != nil {
				tx.Rollback()
				controller.Log.WithError(err).WithField("optionIndex", i).Error("Failed to create option - transaction rolled back")
				ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
					Code:   http.StatusInternalServerError,
					Status: "Internal Server Error",
					Error:  "Failed to create option: " + err.Error(),
				})
				return
			}
			resultOptions = append(resultOptions, optionData)

		default:
			// Auto-detect operation based on OptionID presence (backwards compatibility)
			if optionReq.OptionID != nil {
				// Update existing option
				optionRequest := web.OptionUpdateRequest{
					OptionID:     *optionReq.OptionID,
					QuestionID:   questionId,
					OptionLetter: optionReq.OptionLetter,
					OptionText:   optionReq.OptionText,
					Score:        optionReq.Score,
					IsImage:      0,
					ImageURL:     "",
				}

				if optionReq.IsImage != nil {
					optionRequest.IsImage = *optionReq.IsImage
				}

				if optionReq.ImageURL != nil {
					optionRequest.ImageURL = *optionReq.ImageURL
				}

				optionData, err := controller.OptionService.Update(tx, &optionRequest)
				if err != nil {
					tx.Rollback()
					controller.Log.WithError(err).WithField("optionIndex", i).WithField("optionId", *optionReq.OptionID).Error("Failed to update option - transaction rolled back")
					ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
						Code:   http.StatusInternalServerError,
						Status: "Internal Server Error",
						Error:  "Failed to update option: " + err.Error(),
					})
					return
				}
				resultOptions = append(resultOptions, optionData)
			} else {
				// Create new option
				optionRequest := web.OptionCreateRequest{
					QuestionID:   questionId,
					OptionLetter: optionReq.OptionLetter,
					OptionText:   optionReq.OptionText,
					Score:        optionReq.Score,
					IsImage:      0,
					ImageURL:     "",
				}

				if optionReq.IsImage != nil {
					optionRequest.IsImage = *optionReq.IsImage
				}

				if optionReq.ImageURL != nil {
					optionRequest.ImageURL = *optionReq.ImageURL
				}

				if optionReq.ImageURL != nil {
					optionRequest.ImageURL = *optionReq.ImageURL
				}

				optionData, err := controller.OptionService.Create(tx, &optionRequest)
				if err != nil {
					tx.Rollback()
					controller.Log.WithError(err).WithField("optionIndex", i).Error("Failed to create option - transaction rolled back")
					ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
						Code:   http.StatusInternalServerError,
						Status: "Internal Server Error",
						Error:  "Failed to create option: " + err.Error(),
					})
					return
				}
				resultOptions = append(resultOptions, optionData)
			}
		}
	}

	// 🎉 COMMIT TRANSACTION - All or nothing!
	if err := tx.Commit().Error; err != nil {
		controller.Log.WithError(err).Error("Failed to commit transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to commit transaction",
		})
		return
	}

	controller.Log.WithField("questionId", questionData.QuestionID).WithField("optionsCount", len(resultOptions)).Info("Successfully updated question with options")

	// Response
	questionDataWithOptions := web.QuestionWithOptionsData{
		QuestionId:        questionData.QuestionID,
		Role:              questionData.Role,
		QuestionText:      questionData.QuestionText,
		QuestionType:      questionData.QuestionType,
		AcceptableAnswers: questionData.AcceptableAnswers,
		Category:          helpers.PtrToString(questionData.Category),
		IsImage:           helpers.PtrToString(questionData.IsImage),
		ImageUrl:          helpers.PtrToString(questionData.ImageURL),
		Options:           resultOptions,
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   questionDataWithOptions,
	})
}

// Delete /api/question-with-options/:questionId
func (controller *QuestionOptionController) DeleteQuestionWithOptions(ctx *gin.Context) {
	// 1) Parse & validate path param
	qidStr := ctx.Param("questionId")
	questionId, err := strconv.Atoi(qidStr)
	if err != nil || questionId <= 0 {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "invalid questionId",
		})
		return
	}

	controller.Log.WithField("questionId", questionId).Info("Deleting question with options")

	// 2) Begin transaction
	tx := controller.DB.Begin()
	if tx.Error != nil {
		controller.Log.WithError(tx.Error).Error("Failed to begin transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to begin transaction",
		})
		return
	}

	// 3) Ensure rollback if panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			controller.Log.Error("Transaction rolled back due to panic")
		}
	}()

	// 4) Hapus semua options milik question menggunakan DeleteByQuestionId
	err = controller.OptionService.DeleteByQuestionId(tx, questionId)
	if err != nil {
		tx.Rollback()
		controller.Log.WithError(err).Error("Failed to delete options by question ID - transaction rolled back")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to delete options: " + err.Error(),
		})
		return
	}

	// 5) Hapus question-nya
	if derr := controller.QuestionService.Delete(tx, questionId); derr != nil {
		tx.Rollback()
		controller.Log.WithError(derr).WithField("questionId", questionId).Error("Failed to delete question - transaction rolled back")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to delete question: " + derr.Error(),
		})
		return
	}

	// 6) Commit
	if cerr := tx.Commit().Error; cerr != nil {
		controller.Log.WithError(cerr).Error("Failed to commit transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  "Failed to commit transaction",
		})
		return
	}

	// 7) Sukses
	controller.Log.WithField("questionId", questionId).Info("Successfully deleted question and its options")

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data: gin.H{
			"questionId": questionId,
			"message":    "Question and all associated options deleted successfully",
		},
	})
}

// DELETE /api/question-with-options (body: { "questionIds": [1,2,3] })
func (controller *QuestionOptionController) BulkDelete(ctx *gin.Context) {
	var request struct {
		QuestionIds []int `json:"questionIds" binding:"required"`
	}

	// 1) Validasi payload
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

	controller.Log.WithField("questionIds", request.QuestionIds).
		Info("Bulk deleting questions WITH options (transactional)")

	// 2) Begin transaction
	tx := controller.DB.Begin()
	if tx.Error != nil {
		controller.Log.WithError(tx.Error).Error("Failed to begin transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "INTERNAL SERVER ERROR",
			Error:  "Failed to begin transaction",
		})
		return
	}

	// 3) Pastikan rollback kalau panic
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			controller.Log.Error("Transaction rolled back due to panic")
		}
	}()

	// 4) Hapus OPTIONS dulu untuk tiap question (aman untuk skema tanpa ON DELETE CASCADE)
	//    Jika kamu punya method batch `DeleteByQuestionIds`, gunakan itu agar lebih efisien.
	for _, qid := range request.QuestionIds {
		if qid <= 0 {
			_ = tx.Rollback()
			controller.Log.WithField("questionId", qid).
				Error("Invalid questionId in payload - transaction rolled back")
			ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
				Code:   http.StatusBadRequest,
				Status: "BAD REQUEST",
				Error:  "Invalid questionId in payload",
			})
			return
		}
		// Asumsikan kamu punya OptionService.DeleteByQuestionId(*gorm.DB, int) error
		if err := controller.OptionService.DeleteByQuestionId(tx, qid); err != nil {
			_ = tx.Rollback()
			controller.Log.WithError(err).WithField("questionId", qid).
				Error("Failed to delete options for question - transaction rolled back")
			ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
				Code:   http.StatusInternalServerError,
				Status: "INTERNAL SERVER ERROR",
				Error:  "Failed to delete options for questionId " + strconv.Itoa(qid) + ": " + err.Error(),
			})
			return
		}
	}

	// 5) Setelah semua options dihapus, hapus QUESTIONS secara bulk
	if err := controller.QuestionService.BulkDelete(tx, request.QuestionIds); err != nil {
		_ = tx.Rollback()
		controller.Log.WithError(err).
			WithField("questionIds", request.QuestionIds).
			Error("Failed to bulk delete questions - transaction rolled back")
		respondServiceError(ctx, err)
		return
	}

	// 6) Commit
	if err := tx.Commit().Error; err != nil {
		controller.Log.WithError(err).Error("Failed to commit transaction")
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "INTERNAL SERVER ERROR",
			Error:  "Failed to commit transaction",
		})
		return
	}

	// 7) Sukses
	controller.Log.WithField("questionIds", request.QuestionIds).
		Info("Successfully bulk deleted questions and their options")

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data: gin.H{
			"message":          "Questions and their options deleted",
			"questionsDeleted": len(request.QuestionIds),
			"questionIds":      request.QuestionIds,
		},
	})
}
