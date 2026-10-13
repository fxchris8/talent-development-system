package services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/helpers"
	"backend/internal/models/converter"
	"backend/internal/models/domain"
	"backend/internal/models/web"
	"backend/internal/repositories"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReportService struct {
	DB                      *gorm.DB
	Log                     *logrus.Logger
	Validate                *validator.Validate
	ReportRepository        *repositories.ReportRepository
	GapCompetencyRepository repositories.GapCompetencyRepository
	SeamenCacheRepo         *repositories.SeamenCacheRepository
	MutationCacheRepo       *repositories.MutationCacheRepository
}

func NewReportService(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	reportRepository *repositories.ReportRepository,
	gapCompetencyRepository repositories.GapCompetencyRepository,
	seamenCacheRepo *repositories.SeamenCacheRepository,
	mutationCacheRepo *repositories.MutationCacheRepository,
) *ReportService {
	return &ReportService{
		DB:                      db,
		Log:                     log,
		Validate:                validate,
		ReportRepository:        reportRepository,
		GapCompetencyRepository: gapCompetencyRepository,
		SeamenCacheRepo:         seamenCacheRepo,
		MutationCacheRepo:       mutationCacheRepo,
	}
}

func (service *ReportService) FindAll(ctx context.Context, request *web.DashboardRequest) (*web.SuccessResponse, error) {
	// TODO: Create Transaction
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var reports []domain.Report

	err := service.ReportRepository.SelectAll(tx, request, &reports)
	if err != nil {
		return nil, err
	}

	// TODO: If empty, return empty array and set first_id and last_id equal to null
	if len(reports) == 0 {
		return &web.SuccessResponse{
			Status: "Ok",
			Code:   http.StatusOK,
			Data: map[string]interface{}{
				"reports":    []domain.Report{},
				"first_id":   nil,
				"last_id":    nil,
				"page_size":  request.PageSize,
				"has_more":   false,
				"first_page": true,
			},
		}, nil
	}

	// TODO: Init response
	reportData := converter.ToReportDataList(&reports)
	hasMore := false
	isFirstPage := false

	// //! If for prev, we need to reverse the reports
	if request.Page == "prev" {
		if len(reportData) > request.PageSize {
			isFirstPage = false // Kalau ada lebih, pasti bukan first page
			reportData = reportData[:request.PageSize]
		} else {
			isFirstPage = true
		}

		// Untuk prev, hasil query DESC lalu dibalik biar tetap ASC
		helpers.Reverse(&reportData)
		hasMore = true // Karena page sekarang akan jadi page selanjutnya, jadi pasti TRUE
	} else {
		//! Next
		if len(reportData) > request.PageSize {
			hasMore = true
			reportData = reportData[:request.PageSize]
		}

		if request.AnchorID == 0 {
			isFirstPage = true // 0 digunakan untuk menampilkan page pertama
		} else {
			isFirstPage = false // Karena page sekarang pasti jadi page sebelumya, jadi pasti FALSE
		}
	}

	firstId := reportData[0].ID
	lastId := reportData[len(reportData)-1].ID

	return &web.SuccessResponse{
		Status: "Ok",
		Code:   http.StatusOK,
		Data: map[string]interface{}{
			"results":    reportData,
			"first_id":   firstId,
			"last_id":    lastId,
			"page_size":  request.PageSize,
			"has_more":   hasMore,
			"first_page": isFirstPage,
		},
	}, nil
}

func (service *ReportService) CreateAll(ctx context.Context, request *web.ReportRequest) (*web.SuccessResponse, error) {
	// TODO: Create Transaction
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback() // Rollback just in case

	// TODO: Validate the request
	err := service.Validate.Struct(request)
	if err != nil {
		service.Log.Warnf("Invalid request body: %+v", err)
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	// TODO: Read Excel
	file, err := request.File.Open()
	if err != nil {
		service.Log.Warnf("Cannot open the file: %+v", err)
		return nil, fmt.Errorf("cannot open the file: %w", err)
	}
	defer file.Close()

	// TODO: Parse The Excel File from io.Reader
	excel, err := excelize.OpenReader(file)
	if err != nil {
		service.Log.Warnf("Cannot open the excel: %+v", err)
		return nil, fmt.Errorf("cannot open the excel: %w", err)
	}
	defer excel.Close()

	// TODO: Get the Sheet and the rows
	rows, err := excel.GetRows("Sheet1")
	if err != nil {
		service.Log.Warnf("Cannot get the rows: %+v", err)
		return nil, fmt.Errorf("cannot open the rows: %w", err)
	}

	var reports []domain.Report

	for i, row := range rows {
		// Skip the header
		if i == 0 {
			continue
		}

		// Parse all columns from Excel - Following frontend excel.tsx column order
		// Col 0: No (skip)
		// Col 1: Vessel Name
		// Col 2: Name
		// Col 3: Position
		// Col 4: Department (skip - not in model)
		// Col 5: Seaman Code
		// Col 6: Seafarer Code
		// Col 7: Certificate
		// Col 8: Age
		// Col 9: Kondite Review
		// Col 10: KPI Vessel
		// Col 11: Performance Score
		// Col 12: Value Assessment
		// Col 13: Competency Gap Analysis
		// Col 14: TOTAL GAP
		// Col 15: Strength Analysis
		// Col 16: IDP Program
		// Col 17: HAV Quadran 2
		// Col 18: Talent Classified 2
		// Col 19: Readiness (Month)
		// Col 20: Certificate Eligible
		// Col 21: Durasi Pemenuhan Pendidikan (Month)
		// Col 22: Total Readiness Update (Month)

		vesselName := helpers.SanitizeCell(helpers.GetCell(row, 1))
		nama := helpers.SanitizeCell(helpers.GetCell(row, 2))
		jabatan := helpers.SanitizeCell(helpers.GetCell(row, 3))
		seamanCode := helpers.SanitizeCell(helpers.GetCell(row, 5))
		seafarerCode := helpers.SanitizeCell(helpers.GetCell(row, 6))
		certificate := helpers.SanitizeCell(helpers.GetCell(row, 7))
		age := helpers.SanitizeCell(helpers.GetCell(row, 8))
		konditeReviewStr := helpers.SanitizeCell(helpers.GetCell(row, 9))
		kpiVesselStr := helpers.SanitizeCell(helpers.GetCell(row, 10))
		performanceScoreStr := helpers.SanitizeCell(helpers.GetCell(row, 11))
		valueAssessmentStr := helpers.SanitizeCell(helpers.GetCell(row, 12))
		competencyGapAnalysis := helpers.SanitizeCell(helpers.GetCell(row, 13))
		totalGapStr := helpers.SanitizeCell(helpers.GetCell(row, 14))
		strengthStr := helpers.SanitizeCell(helpers.GetCell(row, 15))
		idpProgram := helpers.SanitizeCell(helpers.GetCell(row, 16))
		havQuadran2Str := helpers.SanitizeCell(helpers.GetCell(row, 17))
		talentClassified2 := helpers.SanitizeCell(helpers.GetCell(row, 18))
		readinessMonthStr := helpers.SanitizeCell(helpers.GetCell(row, 19))
		certificateEligible := helpers.SanitizeCell(helpers.GetCell(row, 20))
		educationFulfillmentMonthsStr := helpers.SanitizeCell(helpers.GetCell(row, 21))
		totalReadinessUpdateMonthsStr := helpers.SanitizeCell(helpers.GetCell(row, 22))

		// Convert string values to integers with proper error handling
		totalGap, _ := strconv.Atoi(totalGapStr)
		valueAssessment, _ := strconv.Atoi(valueAssessmentStr)
		konditeReview, _ := strconv.Atoi(konditeReviewStr)
		kpiVessel, _ := strconv.Atoi(kpiVesselStr)
		performanceScore, _ := strconv.Atoi(performanceScoreStr)
		// Strength is a string (contains competency codes like "BAC; DCM; ING")
		strength := strengthStr
		havQuadran2, _ := strconv.Atoi(havQuadran2Str)
		readinessMonth, _ := strconv.Atoi(readinessMonthStr)
		educationFulfillmentMonths, _ := strconv.Atoi(educationFulfillmentMonthsStr)
		totalReadinessUpdateMonths, _ := strconv.Atoi(totalReadinessUpdateMonthsStr)

		// Check if report already exists by seafarerCode
		var existingReport domain.Report
		_, findErr := service.ReportRepository.FindBySeafarerCode(tx, seafarerCode, &existingReport)

		var report domain.Report
		isUpdate := findErr == nil

		if isUpdate {
			// Update existing report - keep the ID and timestamps
			report = existingReport
			service.Log.Infof("Row %d - Updating existing report ID=%d for Seafarer Code: %s", i, existingReport.ID, seafarerCode)
		} else {
			// New report
			service.Log.Infof("Row %d - Creating new report for Seafarer Code: %s", i, seafarerCode)
		}

		// Update all fields from Excel (non-readiness fields)
		report.SeamanCode = seamanCode
		report.SeafarerCode = seafarerCode
		report.Nama = nama
		report.Jabatan = jabatan
		report.VesselName = vesselName
		report.Certificate = certificate
		report.Age = age
		report.KonditeReview = konditeReview
		report.KpiVessel = kpiVessel
		report.PerformanceScore = performanceScore
		report.ValueAssessment = valueAssessment
		report.CompetencyGapAnalysis = competencyGapAnalysis
		report.TotalGap = totalGap
		report.Strength = strength
		report.IDPProgram = idpProgram
		report.HavQuadran2 = havQuadran2
		report.TalentClassified2 = talentClassified2

		// For readiness-related fields:
		// - NEW reports: Read from Excel
		// - UPDATE reports: Keep existing values (don't overwrite from Excel)
		if !isUpdate {
			// New report - read readiness fields from Excel
			report.ReadinessMonth = &readinessMonth
			report.CertificateEligible = certificateEligible
			report.EducationFulfillmentMonths = &educationFulfillmentMonths
			report.TotalReadinessUpdateMonths = &totalReadinessUpdateMonths

			// Set Readiness status based on ReadinessMonth value
			if readinessMonth == 0 {
				report.Readiness = "Ready Now"
			} else if readinessMonth <= 12 {
				report.Readiness = "Ready 1 Year"
			} else if readinessMonth <= 24 {
				report.Readiness = "Ready 2 Years"
			} else {
				report.Readiness = "Ready 3+ Years"
			}
		}
		// For UPDATE: readiness fields are preserved from existingReport (already copied above)

		// Log first 3 records for debugging
		if i <= 3 {
			service.Log.Infof("Row %d - Name: %s, Vessel: %s, Seaman: %s, Seafarer: %s, Position: %s, ValueAssessment: %d, IsUpdate: %v",
				i, nama, vesselName, seamanCode, seafarerCode, jabatan, valueAssessment, isUpdate)
		}

		// Save or Update the report
		if isUpdate {
			if err = service.ReportRepository.Update(tx, &report); err != nil {
				service.Log.Warnf("Failed updating report for seafarer %s: %+v", seafarerCode, err)
				return nil, fmt.Errorf("failed updating report: %w", err)
			}
			service.Log.Infof("Successfully updated report ID=%d for seafarer: %s", report.ID, seafarerCode)
		} else {
			if err = tx.Create(&report).Error; err != nil {
				service.Log.Warnf("Failed creating report for seafarer %s: %+v", seafarerCode, err)
				return nil, fmt.Errorf("failed creating report: %w", err)
			}
			service.Log.Infof("Successfully created report ID=%d for seafarer: %s", report.ID, seafarerCode)
		}

		reports = append(reports, report)
	}

	service.Log.Infof("Total reports processed: %d", len(reports))

	// TODO: Save Value Assessment scores to report_scores table
	if err = service.saveValueAssessmentScores(tx, &reports); err != nil {
		service.Log.Warnf("Failed saving value assessment scores: %+v", err)
		return nil, fmt.Errorf("failed saving value assessment scores: %w", err)
	}

	// TODO: Process Gap Competencies from CompetencyGapAnalysis field
	if err = service.processGapCompetencies(tx, &reports); err != nil {
		service.Log.Warnf("Failed processing gap competencies: %+v", err)
		return nil, fmt.Errorf("failed processing gap compWetencies: %w", err)
	}

	// TODO: Commit Transaction
	if err = tx.Commit().Error; err != nil {
		service.Log.Warnf("Failed commit transaction: %+v", err)
		return nil, fmt.Errorf("failed commit transaction: %w", err)
	}

	service.Log.Info("Reports created successfully")

	return &web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "Created",
		Data:   "Reports Created Successfully",
	}, nil
}

func (service *ReportService) IDPCount(ctx context.Context, request *web.DashboardRequest) (*web.SuccessResponse, error) {
	// TODO: Create Transaction
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	// TODO: Get IDP Count
	var data web.IDPCountData
	err := service.ReportRepository.IDPCount(tx, request, &data)
	if err != nil {
		return nil, err
	}

	// TODO: Commit Transaction
	if err = tx.Commit().Error; err != nil {
		service.Log.Warnf("Failed commit transaction: %+v", err)
		return nil, fmt.Errorf("failed commit transaction: %w", err)
	}

	return &web.SuccessResponse{
		Status: "Ok",
		Code:   http.StatusOK,
		Data:   data,
	}, nil
}

func (service *ReportService) FindBySeamanCode(ctx context.Context, seamanCode string) (*web.SuccessResponse, error) {
	// TODO: Create Transaction
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var report domain.Report
	err := service.ReportRepository.FindBySeamanCode(tx, seamanCode, &report)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("seaman code not found")
		}
		return nil, err
	}

	// TODO: Commit Transaction
	if err = tx.Commit().Error; err != nil {
		service.Log.Warnf("Failed commit transaction: %+v", err)
		return nil, fmt.Errorf("failed commit transaction: %w", err)
	}

	reportData := converter.ToReportData(&report)

	return &web.SuccessResponse{
		Status: "Ok",
		Code:   http.StatusOK,
		Data:   reportData,
	}, nil
}

func (service *ReportService) FindBySeafarerCode(ctx context.Context, seafarerCode string) (*web.SuccessResponse, error) {
	// TODO: Create Transaction
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var report domain.Report
	_, err := service.ReportRepository.FindBySeafarerCode(tx, seafarerCode, &report)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("seafarer code not found")
		}
		return nil, err
	}

	// TODO: Commit Transaction
	if err = tx.Commit().Error; err != nil {
		service.Log.Warnf("Failed commit transaction: %+v", err)
		return nil, fmt.Errorf("failed commit transaction: %w", err)
	}

	reportData := converter.ToReportData(&report)

	// Get value assessment score from report_scores table
	reportData.ValueAssessmentScore = service.getValueAssessmentScore(service.DB, seafarerCode)

	return &web.SuccessResponse{
		Status: "Ok",
		Code:   http.StatusOK,
		Data:   reportData,
	}, nil
}

// getValueAssessmentScore retrieves the value assessment score from report_scores table
func (service *ReportService) getValueAssessmentScore(db *gorm.DB, seafarerCode string) int {
	service.Log.Infof("=== START getValueAssessmentScore for seafarer: %s ===", seafarerCode)

	// First, find the report for this seafarer
	var reportModel domain.Report
	report, reportErr := service.ReportRepository.FindBySeafarerCode(db, seafarerCode, &reportModel)
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

// processGapCompetencies parses CompetencyGapAnalysis field and creates gap_competencies records
func (service *ReportService) processGapCompetencies(db *gorm.DB, reports *[]domain.Report) error {
	service.Log.Info("Starting to process gap competencies from Excel upload")

	for _, report := range *reports {
		// Delete all existing gap competencies for this report first
		// This ensures fresh data from Excel replaces old data
		deleteResult := db.Where("report_id = ?", report.ID).Delete(&domain.GapCompetency{})
		if deleteResult.Error != nil {
			service.Log.Warnf("Failed to delete existing gap competencies for report ID %d: %v", report.ID, deleteResult.Error)
		} else {
			service.Log.Infof("Deleted %d existing gap competencies for report ID=%d", deleteResult.RowsAffected, report.ID)
		}

		// Skip if no competency gap analysis data
		if report.CompetencyGapAnalysis == "" {
			service.Log.Infof("No competency gap analysis data for report ID %d, skipping", report.ID)
			continue
		}

		// Parse the competency codes (e.g., "L1; L2; M3")
		codes := strings.Split(report.CompetencyGapAnalysis, ";")

		for _, code := range codes {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}

			// Find the competency type by code
			var competencyType domain.CompetencyType
			if err := db.Where("code = ?", code).First(&competencyType).Error; err != nil {
				service.Log.Warnf("Competency type not found for code '%s' in report ID %d, skipping", code, report.ID)
				continue
			}

			// Create gap competency record with ON DUPLICATE KEY UPDATE to prevent duplicates
			gapCompetency := domain.GapCompetency{
				ReportID:         report.ID,
				CompetencyTypeID: competencyType.ID,
				GapLevel:         "MEDIUM",
				Priority:         1,
			}

			// Use Clauses to handle duplicate (report_id, competency_type_id) gracefully
			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "report_id"}, {Name: "competency_type_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
			}).Create(&gapCompetency).Error; err != nil {
				service.Log.Warnf("Failed to create gap competency for report ID %d, competency '%s': %v", report.ID, code, err)
				continue
			}
			service.Log.Infof("Created gap competency: Report ID=%d, Competency=%s (%d)", report.ID, code, competencyType.ID)
		}
	}

	service.Log.Info("Finished processing gap competencies")
	return nil
}

// saveValueAssessmentScores saves Value Assessment scores to report_scores table
func (service *ReportService) saveValueAssessmentScores(db *gorm.DB, reports *[]domain.Report) error {
	service.Log.Info("🔍 Starting to save Value Assessment scores to report_scores table")

	// First, let's check what assessment types exist in the database
	var allAssessmentTypes []domain.AssessmentType
	if err := db.Find(&allAssessmentTypes).Error; err != nil {
		service.Log.Warnf("❌ Failed to fetch all assessment types: %v", err)
	} else {
		service.Log.Infof("📊 Available Assessment Types in DB: %d types", len(allAssessmentTypes))
		for _, at := range allAssessmentTypes {
			service.Log.Infof("  - ID=%d, Name=%s", at.ID, at.AssessmentTypeName)
		}
	}

	// Get Value Assessment type ID
	var assessmentType domain.AssessmentType
	typeErr := db.Where("assessment_type_name = ?", "Value Assessment").First(&assessmentType).Error
	if typeErr != nil {
		service.Log.Warnf("❌ Value Assessment type NOT found in assessment_types table: %v", typeErr)
		service.Log.Info("⚠️ Will skip report_scores creation for Value Assessment")
		return nil // Don't fail the entire upload, just skip this step
	}
	service.Log.Infof("✅ Found Value Assessment type: ID=%d, Name=%s", assessmentType.ID, assessmentType.AssessmentTypeName)

	if len(*reports) == 0 {
		service.Log.Info("⏭️ No reports to process")
		return nil
	}

	successCount := 0
	skippedCount := 0

	for i, report := range *reports {
		// Only create report_scores if ValueAssessment > 0
		if report.ValueAssessment <= 0 {
			service.Log.Infof("⏭️ Row %d: Skipping report_scores for Report ID=%d (ValueAssessment=%d is 0 or negative)", i, report.ID, report.ValueAssessment)
			skippedCount++
			continue
		}

		service.Log.Infof("🔄 Row %d: Processing Report ID=%d, ValueAssessment=%d", i, report.ID, report.ValueAssessment)

		// Check if report_scores entry already exists
		var existingScore domain.ReportScore
		err := db.Where("report_id = ? AND assessment_type_id = ?", report.ID, assessmentType.ID).First(&existingScore).Error

		if err == gorm.ErrRecordNotFound {
			// Create new report_scores entry
			reportScore := domain.ReportScore{
				ReportID:         int64(report.ID),
				AssessmentTypeID: assessmentType.ID,
				Score:            report.ValueAssessment,
			}

			createErr := db.Create(&reportScore).Error
			if createErr != nil {
				service.Log.Errorf("❌ Failed to CREATE report_scores for Report ID=%d, Assessment Type ID=%d, Score=%d. Error: %v",
					report.ID, assessmentType.ID, report.ValueAssessment, createErr)

				// Fallback: try raw SQL INSERT
				service.Log.Infof("🔧 Trying fallback raw SQL INSERT...")
				rawErr := db.Exec(
					"INSERT INTO report_scores (report_id, assessment_type_id, score) VALUES (?, ?, ?)",
					report.ID, assessmentType.ID, report.ValueAssessment,
				).Error
				if rawErr != nil {
					service.Log.Errorf("❌ Raw SQL INSERT also failed: %v", rawErr)
					continue
				}
				service.Log.Infof("✅ Raw SQL INSERT succeeded")
			}

			// Verify the insert
			var verifyScore domain.ReportScore
			verifyErr := db.Where("report_id = ? AND assessment_type_id = ?", report.ID, assessmentType.ID).First(&verifyScore).Error
			if verifyErr != nil {
				service.Log.Warnf("⚠️ Failed to verify created report_scores for Report ID=%d: %v", report.ID, verifyErr)
			} else {
				service.Log.Infof("✅ VERIFIED - Created report_scores: ReportScore ID=%d, Report ID=%d, Assessment Type=%s (ID=%d), Score=%d",
					verifyScore.ID, report.ID, assessmentType.AssessmentTypeName, assessmentType.ID, verifyScore.Score)
				successCount++
			}
		} else if err == nil {
			// Update existing report_scores entry
			updateErr := db.Model(&existingScore).Update("score", report.ValueAssessment).Error
			if updateErr != nil {
				service.Log.Errorf("❌ Failed to UPDATE report_scores ID=%d for report ID %d with score %d: %v",
					existingScore.ID, report.ID, report.ValueAssessment, updateErr)
				continue
			}

			// Verify the update
			var verifyScore domain.ReportScore
			verifyErr := db.Where("id = ?", existingScore.ID).First(&verifyScore).Error
			if verifyErr != nil {
				service.Log.Warnf("⚠️ Failed to verify updated report_scores ID=%d: %v", existingScore.ID, verifyErr)
			} else {
				service.Log.Infof("✅ VERIFIED - Updated report_scores: ReportScore ID=%d, Report ID=%d, Assessment Type=%s (ID=%d), Score=%d",
					verifyScore.ID, report.ID, assessmentType.AssessmentTypeName, assessmentType.ID, verifyScore.Score)
				successCount++
			}
		} else {
			service.Log.Warnf("⚠️ Unexpected error checking existing report_scores for report ID %d: %v", report.ID, err)
			continue
		}
	}

	service.Log.Infof("📊 Finished saving Value Assessment scores - Success: %d, Skipped: %d, Total: %d",
		successCount, skippedCount, len(*reports))
	return nil
}

func (service *ReportService) GetTrainingDataBySeafarerCode(ctx context.Context, seafarerCode string, apolloService *ApolloAPIService) ([]map[string]interface{}, error) {
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var report domain.Report
	_, err := service.ReportRepository.FindBySeafarerCode(tx, seafarerCode, &report)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}

	var gapCompetencies []domain.GapCompetency
	if err := tx.Where("report_id = ?", report.ID).Preload("CompetencyType").Find(&gapCompetencies).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch gap competencies: %w", err)
	}

	var trainingSchedules []domain.TrainingSchedule
	if err := tx.Where("is_started = ?", true).Find(&trainingSchedules).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch training schedules: %w", err)
	}

	gapMap := make(map[string]bool)
	for _, gap := range gapCompetencies {
		gapMap[gap.CompetencyType.Code] = true
	}

	var trainingData []map[string]interface{}

	for _, schedule := range trainingSchedules {
		if !gapMap[schedule.CompetencyCode] {
			continue
		}

		courseName := schedule.TrainingTopic
		if schedule.ApolloCourseName != nil && *schedule.ApolloCourseName != "" {
			courseName = *schedule.ApolloCourseName
		}

		apolloResp, err := apolloService.GetTrainingData(report.SeamanCode, courseName)
		if err != nil {
			service.Log.Warnf("Failed to get Apollo data for %s - %s: %v", report.SeamanCode, courseName, err)
			continue
		}

		if apolloResp != nil && len(apolloResp.OutputParams.CsrResult) > 0 {
			for _, record := range apolloResp.OutputParams.CsrResult {
				trainingData = append(trainingData, map[string]interface{}{
					"courseName":    record.CoursesName,
					"startDate":     record.StartDate,
					"finishDate":    record.FinishDate,
					"pointPre":      record.PointPre,
					"pointPost":     record.PointPost,
					"minimumPoint":  record.MinimumPoint,
					"coursesHours":  record.CoursesHours,
					"competencyCode": schedule.CompetencyCode,
				})
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return trainingData, nil
}

func (service *ReportService) GetTrainingSummaryBySeafarerCode(ctx context.Context, seafarerCode string, apolloService *ApolloAPIService) (map[string]interface{}, error) {
	tx := service.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var report domain.Report
	_, err := service.ReportRepository.FindBySeafarerCode(tx, seafarerCode, &report)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}

	var gapCompetencies []domain.GapCompetency
	if err := tx.Where("report_id = ?", report.ID).Preload("CompetencyType").Find(&gapCompetencies).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch gap competencies: %w", err)
	}

	var allGapCompetencies []domain.GapCompetency
	if err := tx.Joins("JOIN reports ON reports.id = gap_competencies.report_id").
		Where("reports.idp_program = ?", report.IDPProgram).
		Preload("CompetencyType").
		Find(&allGapCompetencies).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch all gap competencies for program: %w", err)
	}

	reportGaps := make(map[int]map[string]bool)
	for _, gc := range allGapCompetencies {
		if reportGaps[gc.ReportID] == nil {
			reportGaps[gc.ReportID] = make(map[string]bool)
		}
		if gc.CompetencyType != nil {
			reportGaps[gc.ReportID][gc.CompetencyType.Code] = true
		}
	}

	totalParticipants := len(reportGaps)
	competencyCountMap := make(map[string]int)

	for _, gaps := range reportGaps {
		for code := range gaps {
			competencyCountMap[code]++
		}
	}

	mandatoryGaps := make(map[string]bool)
	nonMandatoryGaps := make(map[string]bool)

	for _, gap := range gapCompetencies {
		code := gap.CompetencyType.Code
		count := competencyCountMap[code]
		percentage := float64(count) / float64(totalParticipants) * 100

		if percentage >= 60.0 {
			mandatoryGaps[code] = true
		} else {
			nonMandatoryGaps[code] = true
		}
	}

	mandatoryCompleted := 0
	mandatoryNotCompleted := 0
	nonMandatoryCompleted := 0
	nonMandatoryNotCompleted := 0

	for _, gap := range gapCompetencies {
		code := gap.CompetencyType.Code
		isMandatory := mandatoryGaps[code]

		var trainingSchedule domain.TrainingSchedule
		err := tx.Where("program = ? AND competency_code = ?", report.IDPProgram, code).First(&trainingSchedule).Error
		
		if err != nil {
			service.Log.Warnf("Training schedule not found for program %s and competency %s: %v", report.IDPProgram, code, err)
			if isMandatory {
				mandatoryNotCompleted++
			} else {
				nonMandatoryNotCompleted++
			}
			continue
		}

		courseName := trainingSchedule.TrainingTopic
		if trainingSchedule.IsStarted && trainingSchedule.ApolloCourseName != nil && *trainingSchedule.ApolloCourseName != "" {
			courseName = *trainingSchedule.ApolloCourseName
		}

		apolloResp, err := apolloService.GetTrainingData(report.SeamanCode, courseName)
		isCompleted := false

		if err == nil && apolloResp != nil && len(apolloResp.OutputParams.CsrResult) > 0 {
			for _, record := range apolloResp.OutputParams.CsrResult {
				if record.FinishDate != "" && record.FinishDate != "-" {
					isCompleted = true
					break
				}
			}
		}

		if isMandatory {
			if isCompleted {
				mandatoryCompleted++
			} else {
				mandatoryNotCompleted++
			}
		} else {
			if isCompleted {
				nonMandatoryCompleted++
			} else {
				nonMandatoryNotCompleted++
			}
		}
	}

	totalCompleted := mandatoryCompleted + nonMandatoryCompleted
	totalNotCompleted := mandatoryNotCompleted + nonMandatoryNotCompleted
	totalTrainings := totalCompleted + totalNotCompleted

	mandatoryPercentage := 0.0
	if mandatoryCompleted+mandatoryNotCompleted > 0 {
		mandatoryPercentage = float64(mandatoryCompleted) / float64(mandatoryCompleted+mandatoryNotCompleted) * 100
	}

	nonMandatoryPercentage := 0.0
	if nonMandatoryCompleted+nonMandatoryNotCompleted > 0 {
		nonMandatoryPercentage = float64(nonMandatoryCompleted) / float64(nonMandatoryCompleted+nonMandatoryNotCompleted) * 100
	}

	totalPercentage := 0.0
	if totalTrainings > 0 {
		totalPercentage = float64(totalCompleted) / float64(totalTrainings) * 100
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return map[string]interface{}{
		"mandatory": map[string]interface{}{
			"completed":    mandatoryCompleted,
			"notCompleted": mandatoryNotCompleted,
			"percentage":   mandatoryPercentage,
		},
		"nonMandatory": map[string]interface{}{
			"completed":    nonMandatoryCompleted,
			"notCompleted": nonMandatoryNotCompleted,
			"percentage":   nonMandatoryPercentage,
		},
		"total": map[string]interface{}{
			"completed":    totalCompleted,
			"notCompleted": totalNotCompleted,
			"percentage":   totalPercentage,
		},
	}, nil
}

func (service *ReportService) calculateEducationFulfillmentMonths(certificate, idpProgram string) int {
	switch idpProgram {
	case "SDP": // SDP Program (highest level)
		switch certificate {
		case "ANT-I", "ATT-I":
			return 0
		case "ANT-II", "ATT-II":
			return 5
		case "ANT-III", "ATT-III":
			return 8
		default:
			return 0
		}
	case "MDP": // MDP Program (middle level)
		switch certificate {
		case "ANT-I", "ATT-I", "ANT-II", "ATT-II":
			return 0
		case "ANT-III", "ATT-III":
			return 8
		default:
			return 0
		}
	case "FDP": // FDP Program (lowest level - all certificates result in 0)
		return 0
	default:
		return 0
	}
}

func (service *ReportService) RefreshPersonalDataForAllReports() error {
	service.Log.Info("Refreshing personal data for all reports...")

	var reports []domain.Report
	if err := service.DB.Find(&reports).Error; err != nil {
		return fmt.Errorf("failed to get reports: %w", err)
	}

	updatedCount := 0
	var unmatched []string
	for _, report := range reports {
		seamenCache := service.findSeamenCache(report.SeamanCode, report.SeafarerCode)
		if seamenCache == nil {
			unmatched = append(unmatched, fmt.Sprintf("%s(%s)", report.SeamanCode, report.Nama))
			continue
		}

		mutations, _ := service.MutationCacheRepo.GetAllBySeamanCode(seamenCache.SeamanCode)
		updates := citrixReportUpdates(seamenCache, mutations)

		// Program follows the position: enrol when there is no program yet, leave the
		// program when CITRIX shows a promotion (never enrol into the next one automatically).
		state, exited := ProgramState{
			Program:       report.IDPProgram,
			Status:        report.TrainingStatus,
			ExitedProgram: report.ExitedProgram,
		}.ApplyPosition(seamenCache.LastPosition)
		if exited || programLevel(state.Program) > 0 {
			updates["idp_program"] = state.Program
			updates["training_status"] = state.Status
			updates["exited_program"] = state.ExitedProgram
		}
		if exited {
			updates["program_exited_at"] = time.Now()
			updates["completed_before_exit"] = state.CompletedBeforeExit
			service.Log.Infof("Report %s (%s) left %s: promoted to %s",
				report.SeamanCode, report.Nama, state.ExitedProgram, seamenCache.LastPosition)
		}

		if err := service.DB.Model(&domain.Report{}).
			Where("id = ?", report.ID).
			Updates(updates).Error; err != nil {
			service.Log.Errorf("Failed to update report %s: %v", report.SeamanCode, err)
			continue
		}

		updatedCount++
	}

	if len(unmatched) > 0 {
		sample := unmatched
		if len(sample) > 20 {
			sample = sample[:20]
		}
		service.Log.Warnf("%d report(s) not found in CITRIX seamen cache (first %d): %s",
			len(unmatched), len(sample), strings.Join(sample, ", "))
	}
	service.Log.Infof("✅ Successfully updated %d reports", updatedCount)
	return nil
}

// findSeamenCache matches a report to its CITRIX record: by seaman code (trimmed), then by
// seafarer code when the seaman code does not match.
func (service *ReportService) findSeamenCache(seamanCode, seafarerCode string) *domain.SeamenCache {
	if code := strings.TrimSpace(seamanCode); code != "" {
		if cache, err := service.SeamenCacheRepo.GetBySeamanCode(code); err == nil {
			return cache
		}
	}
	if code := strings.TrimSpace(seafarerCode); code != "" {
		if cache, err := service.SeamenCacheRepo.GetBySeafarerCode(code); err == nil {
			return cache
		}
	}
	return nil
}

func (service *ReportService) buildVesselHistoryJSON(mutations []domain.MutationCache) string {
	return buildVesselHistoryJSON(mutations)
}

func getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func (service *ReportService) parseStartDate(startDate string) *time.Time {
	return parseCitrixDate(startDate)
}
