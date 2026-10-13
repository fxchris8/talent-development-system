package services

import (
	"backend/internal/models/domain"
	"backend/internal/models/web"
	"backend/internal/repositories"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AssessmentResultService interface {
	SubmitAssessment(db *gorm.DB, request *web.AssessmentSubmitRequest) (*web.AssessmentResultData, error)
	FindBySeafarerCode(db *gorm.DB, seafarerCode string) (*web.AssessmentResultData, error)
	GetValueAssessmentReport(db *gorm.DB, seafarerCode string) (*web.ValueAssessmentReportResponse, error)
}

type assessmentResultServiceImpl struct {
	AssessmentResultRepository repositories.AssessmentResultRepository
	QuestionRepository         repositories.QuestionRepository
	OptionRepository           repositories.OptionRepository
	ReportRepository           *repositories.ReportRepository
	Log                        *logrus.Logger
	Validate                   *validator.Validate
}

func NewAssessmentResultService(
	assessmentResultRepository repositories.AssessmentResultRepository,
	questionRepository repositories.QuestionRepository,
	optionRepository repositories.OptionRepository,
	reportRepository *repositories.ReportRepository,
	log *logrus.Logger,
	validate *validator.Validate,
) AssessmentResultService {
	return &assessmentResultServiceImpl{
		AssessmentResultRepository: assessmentResultRepository,
		QuestionRepository:         questionRepository,
		OptionRepository:           optionRepository,
		ReportRepository:           reportRepository,
		Log:                        log,
		Validate:                   validate,
	}
}

func (service *assessmentResultServiceImpl) SubmitAssessment(db *gorm.DB, request *web.AssessmentSubmitRequest) (*web.AssessmentResultData, error) {
	err := service.Validate.Struct(request)
	if err != nil {
		service.Log.WithError(err).Error("Error validating assessment submit request")
		return nil, err
	}

	// Get assessment by role
	var assessment domain.Assessment
	if err := db.Where("role = ?", request.Role).First(&assessment).Error; err != nil {
		service.Log.WithError(err).Error("Error finding assessment by role")
		return nil, err
	}

	// Only seafarers assigned to this assessment's type may submit answers for it.
	if assessment.AssessTypeID != nil {
		assigned, err := seafarerIsAssigned(db, request.SeafarerCode, *assessment.AssessTypeID)
		if err != nil {
			return nil, err
		}
		if !assigned {
			return nil, ErrNotAssigned
		}
	}

	// Find or create assessment result
	var assessmentResult *domain.AssessmentResult
	err = db.Where("seafarer_code = ? AND assessment_id = ?", request.SeafarerCode, assessment.AssessmentID).
		First(&assessmentResult).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new assessment result
		assessmentResult = &domain.AssessmentResult{
			SeafarerCode: request.SeafarerCode,
			AssessmentID: int64(assessment.AssessmentID),
			IsCompleted:  0,
		}
		if err := db.Create(assessmentResult).Error; err != nil {
			service.Log.WithError(err).Error("Error creating assessment result")
			return nil, err
		}
	} else if err != nil {
		service.Log.WithError(err).Error("Error finding assessment result")
		return nil, err
	}

	// Calculate scores based on aspects
	err = service.calculateAspectScores(db, assessmentResult, request.Answers, int(assessment.AssessmentID))
	if err != nil {
		service.Log.WithError(err).Error("Error calculating aspect scores")
		return nil, err
	}

	// Mark as completed
	assessmentResult.IsCompleted = 1
	now := time.Now()
	assessmentResult.CompletedAt = &now

	// Update assessment result
	if err := db.Save(assessmentResult).Error; err != nil {
		service.Log.WithError(err).Error("Error updating assessment result")
		return nil, err
	}

	// Reload with relations
	if err := db.Preload("ScoreResults.Aspect").First(assessmentResult, assessmentResult.ID).Error; err != nil {
		service.Log.WithError(err).Error("Error reloading assessment result")
		return nil, err
	}

	// Calculate and save to report_scores table
	if err := service.calculateAndSaveReportScore(db, &assessment, assessmentResult, request); err != nil {
		service.Log.WithError(err).Error("Error calculating and saving report score")
		// Don't return error, just log it - assessment submission should still succeed
	}

	return service.convertToAssessmentResultData(assessmentResult), nil
}

func (service *assessmentResultServiceImpl) calculateAspectScores(db *gorm.DB, assessmentResult *domain.AssessmentResult, answers map[int]int, assessmentID int) error {
	// Get all aspects for this assessment
	var aspects []domain.Aspect
	if err := db.Where("assessment_id = ?", assessmentID).Find(&aspects).Error; err != nil {
		return err
	}

	// Group questions by aspect
	questionAspectMap := make(map[int]int) // questionID -> aspectID
	var questions []domain.Question
	if err := db.Where("assessment_id = ?", assessmentID).Find(&questions).Error; err != nil {
		return err
	}

	for _, q := range questions {
		if q.AspectID != nil {
			questionAspectMap[q.QuestionID] = int(*q.AspectID)
		}
	}

	// Calculate raw scores per aspect
	aspectRawScores := make(map[int]int)
	for questionID, optionID := range answers {
		option, err := service.OptionRepository.FindById(db, optionID)
		if err != nil {
			service.Log.WithError(err).Errorf("Error finding option %d", optionID)
			continue
		}

		if aspectID, exists := questionAspectMap[questionID]; exists {
			aspectRawScores[aspectID] += option.Score
		}
	}

	// Create or update score results for each aspect
	for _, aspect := range aspects {
		rawScore := aspectRawScores[aspect.ID]
		// Calculate final score based on weight
		finalScore := (rawScore * aspect.Weight) / 100

		// Check if score result already exists
		var existingScoreResult domain.ScoreResult
		err := db.Where("assessment_result_id = ? AND aspect_id = ?", assessmentResult.ID, aspect.ID).
			First(&existingScoreResult).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new score result
			scoreResult := domain.ScoreResult{
				RawScore:           rawScore,
				FinalScore:         finalScore,
				AssessmentResultID: assessmentResult.ID,
				AspectID:           aspect.ID,
			}
			if err := db.Create(&scoreResult).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			// Update existing score result
			existingScoreResult.RawScore = rawScore
			existingScoreResult.FinalScore = finalScore
			if err := db.Save(&existingScoreResult).Error; err != nil {
				return err
			}
		}
	}

	return nil
}



func (service *assessmentResultServiceImpl) FindBySeafarerCode(db *gorm.DB, seafarerCode string) (*web.AssessmentResultData, error) {
	assessmentResult, err := service.AssessmentResultRepository.FindBySeafarerCode(db, seafarerCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("assessment result not found")
		}
		service.Log.WithError(err).Error("Error finding assessment result by seafarer code")
		return nil, err
	}

	result := service.convertToAssessmentResultData(assessmentResult)

	// Get value assessment from report_scores table
	service.Log.Infof("FindBySeafarerCode: Getting value assessment for seafarer %s", seafarerCode)
	result.ValueAssessment = service.getValueAssessmentScore(db, seafarerCode)
	service.Log.Infof("FindBySeafarerCode: Value assessment result: %d", result.ValueAssessment)

	return result, nil
}

func (service *assessmentResultServiceImpl) GetValueAssessmentReport(db *gorm.DB, seafarerCode string) (*web.ValueAssessmentReportResponse, error) {
	// Find all assessment results for this seafarer
	var assessmentResults []domain.AssessmentResult
	err := db.Preload("ScoreResults.Aspect").
		Where("seafarer_code = ? AND is_completed = ?", seafarerCode, 1).
		Order("completed_at DESC").
		Find(&assessmentResults).Error

	if err != nil {
		service.Log.WithError(err).Error("Error finding assessment results by seafarer code")
		return nil, err
	}

	if len(assessmentResults) == 0 {
		return nil, errors.New("no completed assessment found")
	}

	// Get the most recent completed assessment
	latestAssessment := assessmentResults[0]

	// Initialize response
	response := &web.ValueAssessmentReportResponse{
		SeafarerCode: seafarerCode,
		CompletedAt:  latestAssessment.CompletedAt,
		Coreva: web.CorevaReportData{
			ScoreResult: []web.AspectScoreResult{},
		},
	}

	// Group assessments by assessment ID (1=COREVA, 2=AWARE, 3=GRIT)
	var corevaResults, awareResults, gritResults []domain.AssessmentResult
	for _, result := range assessmentResults {
		switch result.AssessmentID {
		case 1: // COREVA
			corevaResults = append(corevaResults, result)
		case 2: // AWARE
			awareResults = append(awareResults, result)
		case 3: // GRIT
			gritResults = append(gritResults, result)
		}
	}

	// Process COREVA (with aspects)
	if len(corevaResults) > 0 {
		for _, result := range corevaResults {
			for _, scoreResult := range result.ScoreResults {
				response.Coreva.RawScore += scoreResult.RawScore
				response.Coreva.FinalScore += scoreResult.FinalScore

				if scoreResult.Aspect != nil {
					response.Coreva.ScoreResult = append(response.Coreva.ScoreResult, web.AspectScoreResult{
						AspectID:   scoreResult.AspectID,
						AspectName: scoreResult.Aspect.Name,
						RawScore:   scoreResult.RawScore,
						FinalScore: scoreResult.FinalScore,
					})
				}
			}
		}
		response.Coreva.KonversiScore = float64(response.Coreva.FinalScore) * 0.4
	}

	// Process AWARE
	if len(awareResults) > 0 {
		for _, result := range awareResults {
			for _, scoreResult := range result.ScoreResults {
				response.Aware.RawScore += scoreResult.RawScore
				response.Aware.FinalScore += scoreResult.FinalScore
			}
		}
		response.Aware.KonversiScore = float64(response.Aware.FinalScore) * 0.3
	}

	// Process GRIT
	if len(gritResults) > 0 {
		for _, result := range gritResults {
			for _, scoreResult := range result.ScoreResults {
				response.Grit.RawScore += scoreResult.RawScore
				response.Grit.FinalScore += scoreResult.FinalScore
			}
		}
		response.Grit.KonversiScore = float64(response.Grit.FinalScore) * 0.3
	}

	// Calculate total score
	response.TotalScore = response.Coreva.KonversiScore + response.Aware.KonversiScore + response.Grit.KonversiScore

	// Get value assessment score from report_scores table
	response.ValueAssessmentScore = service.getValueAssessmentScore(db, seafarerCode)

	// Keep valueAssessment for backward compatibility (sum of final scores)
	response.ValueAssessment = response.Coreva.FinalScore + response.Aware.FinalScore + response.Grit.FinalScore

	// Determine interpretation based on total score
	// Logic based on Excel formula: =IF(J2="Low";"...";IF(J2="Fair";"...";IF(J2="Excellence";"...";"")))
	// Assuming score ranges: Low (0-40), Fair (41-70), Excellence (71-100)
	if response.TotalScore >= 0 && response.TotalScore <= 40 {
		response.Interpretasi = "Menunjukkan kesenjangan signifikan antara nilai-nilai pribadi dan nilai kerja yang diharapkan."
	} else if response.TotalScore > 40 && response.TotalScore <= 70 {
		response.Interpretasi = "Memiliki dasar nilai kerja yang cukup baik, namun belum sepenuhnya stabil."
	} else if response.TotalScore > 70 && response.TotalScore <= 100 {
		response.Interpretasi = "Menunjukkan internalisasi nilai kerja yang kuat. Individu cenderung bertindak selaras dengan nilai etika, tanggung jawab, dan daya tahan psikologis jangka panjang."
	} else {
		response.Interpretasi = ""
	}

	return response, nil
}

func (service *assessmentResultServiceImpl) convertToAssessmentResultData(assessmentResult *domain.AssessmentResult) *web.AssessmentResultData {
	data := &web.AssessmentResultData{
		ID:           assessmentResult.ID,
		SeafarerCode: assessmentResult.SeafarerCode,
		AssessmentID: assessmentResult.AssessmentID,
		IsCompleted:  assessmentResult.IsCompleted,
		CompletedAt:  assessmentResult.CompletedAt,
		CreatedAt:    assessmentResult.CreatedAt,
		UpdatedAt:    assessmentResult.UpdatedAt,
	}

	// Convert score results if available
	if len(assessmentResult.ScoreResults) > 0 {
		data.ScoreResults = make([]web.ScoreResultData, len(assessmentResult.ScoreResults))
		for i, sr := range assessmentResult.ScoreResults {
			scoreData := web.ScoreResultData{
				ID:                 sr.ID,
				RawScore:           sr.RawScore,
				FinalScore:         sr.FinalScore,
				AssessmentResultID: sr.AssessmentResultID,
				AspectID:           sr.AspectID,
			}

			// Convert aspect if available
			if sr.Aspect != nil {
				aspectData := web.AspectData{
					ID:           sr.Aspect.ID,
					Name:         sr.Aspect.Name,
					Weight:       sr.Aspect.Weight,
					AssessmentID: sr.Aspect.AssessmentID,
					QuestionID:   sr.Aspect.QuestionID,
				}
				scoreData.Aspect = &aspectData
			}

			data.ScoreResults[i] = scoreData
		}
	}

	return data
}

func (service *assessmentResultServiceImpl) calculateAndSaveReportScore(db *gorm.DB, assessment *domain.Assessment, assessmentResult *domain.AssessmentResult, request *web.AssessmentSubmitRequest) error {
	// Get the assessment type
	var assessmentType domain.AssessmentType
	if err := db.Where("id = ?", assessment.AssessTypeID).First(&assessmentType).Error; err != nil {
		service.Log.WithError(err).Error("Error finding assessment type")
		return err
	}

	var finalScore int

	// Check if this is a Value Assessment
	if assessmentType.AssessmentTypeName == "Value Assessment" {
		// For Value Assessment: aggregate all scores from same seafarer + Value Assessment type
		var err error
		finalScore, err = service.calculateValueAssessmentScore(db, request.SeafarerCode)
		if err != nil {
			return err
		}

		// Find or create report_scores entry for this assessment type
		if err := service.saveReportScore(db, request.SeafarerCode, assessment.AssessTypeID, finalScore); err != nil {
			return err
		}

		service.Log.Infof("Saved Value Assessment score %d to report_scores for seafarer %s", finalScore, request.SeafarerCode)
	} else {
		// For other assessment types: use the final score directly from assessment_results
		// Calculate total final score from all aspects of this assessment result
		finalScore = 0
		for _, scoreResult := range assessmentResult.ScoreResults {
			finalScore += scoreResult.FinalScore
		}

		// Find or create report_scores entry
		if err := service.saveReportScore(db, request.SeafarerCode, assessment.AssessTypeID, finalScore); err != nil {
			return err
		}

		service.Log.Infof("Saved %s score %d to report_scores for seafarer %s", assessmentType.AssessmentTypeName, finalScore, request.SeafarerCode)
	}

	return nil
}

// calculateValueAssessmentScore aggregates all Value Assessment scores for a seafarer
// Weight: VA_1 = 40%, VA_2 = 30%, VA_3 = 30%
// Conversion: <60 = 1, <80 = 2, >=80 = 3
func (service *assessmentResultServiceImpl) calculateValueAssessmentScore(db *gorm.DB, seafarerCode string) (int, error) {
	// Get Value Assessment type ID
	var assessmentType domain.AssessmentType
	if err := db.Where("assessment_type_name = ?", "Value Assessment").First(&assessmentType).Error; err != nil {
		service.Log.WithError(err).Error("Error finding Value Assessment type")
		return 0, err
	}

	// Get all assessments of type Value Assessment
	var assessments []domain.Assessment
	if err := db.Where("assess_type_id = ?", assessmentType.ID).Find(&assessments).Error; err != nil {
		service.Log.WithError(err).Error("Error finding assessments for Value Assessment type")
		return 0, err
	}

	// Map to store scores for each VA role
	vaScores := make(map[string]int) // role -> score

	// Get assessment results for this seafarer with each VA assessment
	for _, assessment := range assessments {
		var assessmentResult domain.AssessmentResult
		if err := db.Where("seafarer_code = ? AND assessment_id = ?", seafarerCode, assessment.AssessmentID).
			Preload("ScoreResults.Aspect").
			First(&assessmentResult).Error; err != nil {
			// If no result found for this VA, continue to next
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			service.Log.WithError(err).Error("Error finding assessment result")
			return 0, err
		}

		// Calculate total final score for this VA assessment
		vaScore := 0
		for _, sr := range assessmentResult.ScoreResults {
			vaScore += sr.FinalScore
		}

		// Store score by role
		vaScores[assessment.Role] = vaScore
	}

	// Calculate weighted total
	// VA_1: 40%, VA_2: 30%, VA_3: 30%
	va1Score := vaScores["va_1"]
	va2Score := vaScores["va_2"]
	va3Score := vaScores["va_3"]

	totalScore := (va1Score * 40 / 100) + (va2Score * 30 / 100) + (va3Score * 30 / 100)

	// Convert to scale 1-3
	// <60 = 1, <80 = 2, >=80 = 3
	var finalScore int
	if totalScore < 60 {
		finalScore = 1
	} else if totalScore < 80 {
		finalScore = 2
	} else {
		finalScore = 3
	}

	service.Log.Infof("Value Assessment calculation for seafarer %s: VA_1=%d, VA_2=%d, VA_3=%d, Total=%.0f, Final=%d",
		seafarerCode, va1Score, va2Score, va3Score, float64(totalScore), finalScore)

	return finalScore, nil
}

// saveReportScore saves or updates a report score entry
func (service *assessmentResultServiceImpl) saveReportScore(db *gorm.DB, seafarerCode string, assessmentTypeID *uint64, score int) error {
	// Find report by seafarer code
	report, err := (*service.ReportRepository).FindBySeafarerCode(db, seafarerCode, &domain.Report{})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			service.Log.Warnf("Report not found for seafarer code: %s", seafarerCode)
			return nil // Don't fail if report doesn't exist
		}
		return err
	}

	// Check if report_scores entry already exists
	var reportScore domain.ReportScore
	err = db.Where("report_id = ? AND assessment_type_id = ?", report.ID, assessmentTypeID).
		First(&reportScore).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new entry
		reportScore = domain.ReportScore{
			ReportID:         int64(report.ID),
			AssessmentTypeID: *assessmentTypeID,
			Score:            score,
		}
		if err := db.Create(&reportScore).Error; err != nil {
			service.Log.WithError(err).Error("Error creating report score")
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update existing entry
		reportScore.Score = score
		if err := db.Save(&reportScore).Error; err != nil {
			service.Log.WithError(err).Error("Error updating report score")
			return err
		}
	}

	return nil
}

// getValueAssessmentScore retrieves the value assessment score from report_scores table
func (service *assessmentResultServiceImpl) getValueAssessmentScore(db *gorm.DB, seafarerCode string) int {
	service.Log.Infof("=== START getValueAssessmentScore for seafarer: %s ===", seafarerCode)

	// First, find the report for this seafarer
	var reportModel domain.Report
	report, reportErr := (*service.ReportRepository).FindBySeafarerCode(db, seafarerCode, &reportModel)
	if reportErr != nil {
		// If report not found, return 0
		service.Log.Warnf("❌ Report not found for seafarer %s, error: %v, returning 0", seafarerCode, reportErr)
		return 0
	}
	service.Log.Infof("✅ Found report for seafarer %s: report.ID=%d, report.ValueAssessment=%d", seafarerCode, report.ID, report.ValueAssessment)

	// Get Value Assessment type ID
	var assessmentType domain.AssessmentType
	typeErr := db.Where("assessment_type_name = ?", "Value Assessment").First(&assessmentType).Error
	if typeErr != nil {
		// If assessment type not found, try to get from reports table
		service.Log.Warnf("❌ Value Assessment type not found, error: %v. Using fallback from reports table. Score: %d", typeErr, report.ValueAssessment)
		return report.ValueAssessment
	}
	service.Log.Infof("✅ Found Value Assessment type: ID=%d, Name=%s", assessmentType.ID, assessmentType.AssessmentTypeName)

	// Get the score from report_scores table
	var reportScore domain.ReportScore
	scoreErr := db.Where("report_id = ? AND assessment_type_id = ?", report.ID, assessmentType.ID).
		First(&reportScore).Error
	if scoreErr != nil {
		// Fallback: use value from reports table if report_scores not found
		service.Log.Warnf("❌ report_scores not found for report_id=%d, assessment_type_id=%d. Error: %v. Using fallback from reports table. Score: %d", report.ID, assessmentType.ID, scoreErr, report.ValueAssessment)

		// Let's also check what's actually in report_scores table
		var allReportScores []domain.ReportScore
		db.Where("report_id = ?", report.ID).Find(&allReportScores)
		service.Log.Infof("📊 All report_scores for report_id=%d: %+v", report.ID, allReportScores)

		return report.ValueAssessment
	}

	service.Log.Infof("✅ Successfully retrieved value assessment from report_scores for seafarer %s. reportScore.ID=%d, reportScore.Score=%d (from report_scores table)", seafarerCode, reportScore.ID, reportScore.Score)
	service.Log.Infof("=== END getValueAssessmentScore: returning %d ===", reportScore.Score)
	return reportScore.Score
}


