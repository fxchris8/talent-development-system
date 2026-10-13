package web

import (
	"mime/multipart"

	"backend/internal/helpers"

	"github.com/go-playground/validator/v10"
)

type ReportRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required" validate:"required"`
}

type DashboardRequest struct {
	AnchorID int    `form:"anchor_id" binding:"min=0"`
	Page     string `form:"page" binding:"required,oneof=next prev"`
	PageSize int    `form:"page_size" binding:"required,gte=1"`
	Filter   string `form:"filter" binding:"omitempty,oneof=MDP FDP SDP"`
	BatchID  int    `form:"batch_id"`
	Query    string `form:"query"`
	// InProgram limits the result to people currently in FDP/MDP/SDP (the dashboard list).
	InProgram bool `form:"in_program"`
}

func (request *DashboardRequest) ParseError(errs validator.ValidationErrors) map[string]string {
	errorMessages := make(map[string]string)

	for _, e := range errs {
		field := helpers.GetFieldTagName(request, e.Field())
		switch e.Tag() {
		case "required":
			errorMessages[field] = "is required"
		case "oneof":
			errorMessages[field] = "must be one of: " + e.Param()
		case "gte":
			errorMessages[field] = "must be greater or equal to " + e.Param()
		case "lte":
			errorMessages[field] = "must be less or equal to " + e.Param()
		case "min":
			errorMessages[field] = "must be at least " + e.Param()
		default:
			errorMessages[field] = "is invalid"
		}
	}

	return errorMessages
}
