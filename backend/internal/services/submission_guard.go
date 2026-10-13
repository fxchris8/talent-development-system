package services

import (
	"errors"
	"net/http"

	"backend/internal/models/domain"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// ErrNotAssigned is returned when someone submits answers for an assessment that was never
// assigned to that seafarer code. The submit endpoints are public (the quiz pages have no
// login), so the assignment is what proves the submission is expected.
var ErrNotAssigned = errors.New("seafarer tidak ditugaskan untuk assessment ini")

func seafarerIsAssigned(db *gorm.DB, seafarerCode string, assessmentTypeID uint64) (bool, error) {
	var count int64
	err := db.Model(&domain.SeafarerAssessment{}).
		Where("seafarer_code = ? AND assessment_type_id = ?", seafarerCode, assessmentTypeID).
		Count(&count).Error
	return count > 0, err
}

// SubmissionErrorStatus maps an error from a submit/create service to the HTTP status the
// client should get: bad input is 400, a missing assignment is 403, anything else is 500.
func SubmissionErrorStatus(err error) int {
	var validationErrors validator.ValidationErrors
	switch {
	case errors.As(err, &validationErrors):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotAssigned):
		return http.StatusForbidden
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
