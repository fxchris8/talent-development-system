package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/internal/helpers"
	"backend/internal/models/converter"
	"backend/internal/models/domain"
	"backend/internal/models/web"
	"backend/internal/repositories"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MasterService provides business logic for master report operations
type MasterService struct {
	DB               *gorm.DB
	Log              *logrus.Logger
	Validate         *validator.Validate
	MasterRepository *repositories.MasterRepository
}

// Constructor
func NewMasterService(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, repo *repositories.MasterRepository) *MasterService {
	return &MasterService{
		DB: db, Log: log, Validate: validate, MasterRepository: repo,
	}
}

//
// ---------------------- HELPER FUNCTIONS ----------------------

// Convert mentoring report domain to web model
func (s *MasterService) toMentoringReportWeb(mr *domain.MentoringReport) *web.MentoringReportData {
	if mr == nil {
		return nil
	}

	// Parse mentee names from JSON string
	var menteeNames []string
	if mr.MenteeNames != "" {
		if err := json.Unmarshal([]byte(mr.MenteeNames), &menteeNames); err != nil {
			s.Log.WithError(err).Warn("failed to parse mentee names")
			menteeNames = []string{}
		}
	}

	// Parse report IDs from JSON string
	var reportIDs []int
	if mr.ReportIDs != "" {
		if err := json.Unmarshal([]byte(mr.ReportIDs), &reportIDs); err != nil {
			s.Log.WithError(err).Warn("failed to parse report IDs")
			reportIDs = []int{}
		}
	}

	createdAt := ""
	if mr.CreatedAt != nil {
		createdAt = mr.CreatedAt.Format(time.RFC3339)
	}

	updatedAt := ""
	if mr.UpdatedAt != nil {
		updatedAt = mr.UpdatedAt.Format(time.RFC3339)
	}

	return &web.MentoringReportData{
		ID:              mr.ID,
		MentorName:      mr.MentorName,
		Period:          mr.Period,
		MenteeNames:     menteeNames,
		Department:      mr.Department,
		Program:         mr.Program,
		ProgramTitle:    mr.ProgramTitle,
		SessionNumber:   mr.SessionNumber,
		Date:            mr.Date,
		Duration:        mr.Duration,
		Purpose:         mr.Purpose,
		Observation:     mr.Observation,
		Reflection:      mr.Reflection,
		ActionPlan:      mr.ActionPlan,
		AdditionalNotes: mr.AdditionalNotes,
		ReportIDs:       reportIDs,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

//
// ---------------------- READ OPERATIONS ----------------------

func (s *MasterService) FindAll(req web.MasterListRequest) (*web.SuccessResponse, error) {
	// If the selected batch is completed, serve read-only data from snapshot table
	if req.BatchID > 0 {
		var batch domain.Batch
		if err := s.DB.First(&batch, req.BatchID).Error; err == nil && batch.Status == "completed" {
			return s.findAllFromSnapshot(req)
		}
	}

	db := s.DB.Model(&domain.MasterReport{})

	// --- optional search filter ---
	if req.Query != "" {
		q := strings.ToLower(req.Query)
		db = db.Where(
			s.DB.
				Where("LOWER(nama) LIKE ?", "%"+q+"%").
				Or("seafarer_code LIKE ?", "%"+req.Query+"%"),
		)
	}

	// --- batch filter ---
	if req.BatchID > 0 {
		db = db.Where("batch_id = ?", req.BatchID)
	} else if req.BatchID == -1 {
		db = db.Where("batch_id IS NULL")
	}

	// --- cursor pagination logic ---
	if req.Page == "next" && req.AnchorID > 0 {
		db = db.Where("id > ?", req.AnchorID).Order("id ASC")
	} else if req.Page == "prev" && req.AnchorID > 0 {
		db = db.Where("id < ?", req.AnchorID).Order("id DESC")
	} else {
		db = db.Order("id ASC")
	}

	// --- enforce limit (fetch limit+1 to detect hasMore) ---
	limit := req.PageSize
	if limit <= 0 {
		limit = 10
	}
	db = db.Limit(limit + 1)

	// --- Preload GapCompetencies with CompetencyType, ReportScores with AssessmentType ---
	// Note: MentoringReport is not preloaded - use GET /api/master-reports/mentoring-programs?personName=<name>
	db = db.Preload("GapCompetencies").Preload("GapCompetencies.CompetencyType").
		Preload("ReportScores").Preload("ReportScores.AssessmentType")

	// --- execute query ---
	var rows []domain.MasterReport
	if err := db.Find(&rows).Error; err != nil {
		s.Log.WithError(err).Error("failed to query master reports")
		return nil, fmt.Errorf("failed to retrieve master reports: %w", err)
	}

	// --- complete blank personal columns from CITRIX so rows are never shown empty ---
	s.overlayCitrixData(rows)

	// --- reverse if prev (so UI order stays ascending) ---
	if req.Page == "prev" && len(rows) > 1 {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}

	// --- detect hasMore from the extra row and trim to limit ---
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	// --- map domain → web model ---
	result := make([]web.MasterReportData, 0, len(rows))
	for _, r := range rows {
		// Map competencies
		competencies := make([]web.GapCompetencyData, 0, len(r.GapCompetencies))
		for _, gc := range r.GapCompetencies {
			competencies = append(competencies, web.GapCompetencyData{
				ID:               gc.ID,
				CompetencyTypeID: gc.CompetencyTypeID,
				CompetencyType: web.CompetencyTypeData{
					ID:   gc.CompetencyType.ID,
					Code: gc.CompetencyType.Code,
					Name: gc.CompetencyType.Name,
				},
			})
		}

		// Map report scores
		reportScores := make([]web.ReportScoreData, 0, len(r.ReportScores))
		for _, rs := range r.ReportScores {
			if rs.AssessmentType != nil {
				reportScores = append(reportScores, web.ReportScoreData{
					ID:               rs.ID,
					Score:            rs.Score,
					AssessmentTypeID: rs.AssessmentTypeID,
					AssessmentType: web.AssessmentTypeData{
						ID:                 rs.AssessmentType.ID,
						AssessmentTypeName: rs.AssessmentType.AssessmentTypeName,
						StartTime:          rs.AssessmentType.StartTime,
						EndTime:            rs.AssessmentType.EndTime,
						MaxAttempts:        rs.AssessmentType.MaxAttempts,
					},
				})
			}
		}

		// Calculate total gap as count of gap competencies
		totalGap := len(r.GapCompetencies)

		// Note: Mentoring programs are not loaded here
		// Use GET /api/master-reports/mentoring-programs?personName=<name> to fetch them

		result = append(result, web.MasterReportData{
			ID:                         r.ID,
			VesselName:                 r.VesselName,
			Nama:                       r.Nama,
			Jabatan:                    r.Jabatan,
			User:                       r.User,
			SeamanCode:                 r.SeamanCode,
			SeafarerCode:               r.SeafarerCode,
			Certificate:                r.Certificate,
			Age:                        r.Age,
			KonditeReview:              r.KonditeReview,
			KpiVessel:                  r.KpiVessel,
			PerformanceScore:           r.PerformanceScore,
			ValueAssessment:            r.ValueAssessment,
			AssessmentCenter:           r.AssessmentCenter,
			PotentialScore:             r.PotentialScore,
			HavQuadran:                 r.HavQuadran,
			HavMapping:                 r.HavMapping,
			CompetencyGapAnalysis:      r.CompetencyGapAnalysis,
			TotalGap:                   totalGap, // Set as count of gap competencies
			Strength:                   r.Strength,
			TalentClassified:           r.TalentClassified,
			IDPProgram:                 r.IDPProgram,
			TrainingStatus:             domain.EffectiveTrainingStatus(r.IDPProgram, r.TrainingStatus),
			ExitedProgram:              r.ExitedProgram,
			HavQuadran2:                r.HavQuadran2,
			TalentClassified2:          r.TalentClassified2,
			Readiness:                  r.Readiness,
			CertificateEligible:        r.CertificateEligible,
			EducationFulfillmentMonths: r.EducationFulfillmentMonths,
			TotalReadinessUpdateMonths: r.TotalReadinessUpdateMonths,
			Keterangan:                 r.Keterangan,
			TmNm:                       r.TmNm,
			MentoringReportID:          r.MentoringReportID,
			MentoringReport:            nil, // Fetch via /api/master-reports/mentoring-programs?personName=<name>
			Competencies:               competencies,
			ReportScores:               reportScores,
		})
	}

	// --- detect boundaries for first/last page ---
	var isFirstPage bool
	if len(result) == 0 {
		isFirstPage = true
	} else {
		var count int64
		if err := s.DB.Model(&domain.MasterReport{}).
			Where("id < ?", result[0].ID).
			Count(&count).Error; err != nil {
			s.Log.WithError(err).Warn("failed to count previous records")
		}
		isFirstPage = (count == 0)
	}

	// --- Get total count matching current filters (search + batch) ---
	var totalCount int64
	countDB := s.DB.Model(&domain.MasterReport{})
	if req.Query != "" {
		q := strings.ToLower(req.Query)
		countDB = countDB.Where(
			s.DB.
				Where("LOWER(nama) LIKE ?", "%"+q+"%").
				Or("seafarer_code LIKE ?", "%"+req.Query+"%"),
		)
	}
	if req.BatchID > 0 {
		countDB = countDB.Where("batch_id = ?", req.BatchID)
	} else if req.BatchID == -1 {
		countDB = countDB.Where("batch_id IS NULL")
	}

	if err := countDB.Count(&totalCount).Error; err != nil {
		s.Log.WithError(err).Warn("failed to count total records")
	}

	// NOTE: Mentoring reports will be fetched per-person on the frontend using
	// GET /api/master-reports/mentoring-programs?personName=<NAME>
	// This avoids fetching all mentoring reports which could be expensive

	responsePayload := web.MasterReportListResponse{
		Data:                      result,
		PageSize:                  limit,
		HasMore:                   hasMore,
		FirstPage:                 isFirstPage,
		Total:                     int(totalCount),
		AvailableMentoringReports: []web.MentoringReportData{}, // Empty, fetch per-person instead
	}

	if len(result) > 0 {
		responsePayload.FirstID = result[0].ID
		responsePayload.LastID = result[len(result)-1].ID
	}

	return &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   responsePayload,
	}, nil
}

// findAllFromSnapshot serves the master report list from batch_report_snapshots when the batch is completed.
func (s *MasterService) findAllFromSnapshot(req web.MasterListRequest) (*web.SuccessResponse, error) {
	db := s.DB.Model(&domain.BatchReportSnapshot{}).Where("batch_id = ?", req.BatchID)

	if req.Query != "" {
		q := strings.ToLower(req.Query)
		db = db.Where(
			s.DB.Where("LOWER(nama) LIKE ?", "%"+q+"%").Or("seafarer_code LIKE ?", "%"+req.Query+"%"),
		)
	}

	if req.Page == "next" && req.AnchorID > 0 {
		db = db.Where("id > ?", req.AnchorID).Order("id ASC")
	} else if req.Page == "prev" && req.AnchorID > 0 {
		db = db.Where("id < ?", req.AnchorID).Order("id DESC")
	} else {
		db = db.Order("id ASC")
	}

	limit := req.PageSize
	if limit <= 0 {
		limit = 10
	}
	db = db.Limit(limit + 1)

	var rows []domain.BatchReportSnapshot
	if err := db.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}

	if req.Page == "prev" && len(rows) > 1 {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	result := make([]web.MasterReportData, 0, len(rows))
	for _, r := range rows {
		batchID := r.BatchID
		var tmNm *string
		if r.TMNM != "" {
			tmNm = &r.TMNM
		}
		result = append(result, web.MasterReportData{
			ID:                         r.ID,
			VesselName:                 r.VesselName,
			Nama:                       r.Nama,
			Jabatan:                    r.Jabatan,
			User:                       r.User,
			SeamanCode:                 r.SeamanCode,
			SeafarerCode:               r.SeafarerCode,
			Certificate:                r.Certificate,
			Age:                        r.Age,
			KonditeReview:              r.KonditeReview,
			KpiVessel:                  r.KpiVessel,
			PerformanceScore:           r.PerformanceScore,
			ValueAssessment:            r.ValueAssessment,
			AssessmentCenter:           r.AssessmentCenter,
			PotentialScore:             r.PotentialScore,
			HavQuadran:                 r.HavQuadran,
			HavMapping:                 r.HavMapping,
			CompetencyGapAnalysis:      r.CompetencyGapAnalysis,
			TotalGap:                   r.TotalGap,
			Strength:                   r.Strength,
			TalentClassified:           r.TalentClassified,
			IDPProgram:                 r.IDPProgram,
			HavQuadran2:                r.HavQuadran2,
			TalentClassified2:          r.TalentClassified2,
			Readiness:                  r.Readiness,
			CertificateEligible:        r.CertificateEligible,
			EducationFulfillmentMonths: r.EducationFulfillmentMonths,
			TotalReadinessUpdateMonths: r.TotalReadinessUpdateMonths,
			Keterangan:                 r.Keterangan,
			TmNm:                       tmNm,
			BatchID:                    &batchID,
			Competencies:               []web.GapCompetencyData{},
			ReportScores:               []web.ReportScoreData{},
		})
	}

	var isFirstPage bool
	if len(result) == 0 {
		isFirstPage = true
	} else {
		var count int64
		s.DB.Model(&domain.BatchReportSnapshot{}).
			Where("batch_id = ? AND id < ?", req.BatchID, result[0].ID).
			Count(&count)
		isFirstPage = (count == 0)
	}

	var totalCount int64
	countDB := s.DB.Model(&domain.BatchReportSnapshot{}).Where("batch_id = ?", req.BatchID)
	if req.Query != "" {
		q := strings.ToLower(req.Query)
		countDB = countDB.Where(
			s.DB.Where("LOWER(nama) LIKE ?", "%"+q+"%").Or("seafarer_code LIKE ?", "%"+req.Query+"%"),
		)
	}
	countDB.Count(&totalCount)

	payload := web.MasterReportListResponse{
		Data:                      result,
		PageSize:                  limit,
		HasMore:                   hasMore,
		FirstPage:                 isFirstPage,
		Total:                     int(totalCount),
		IsArchived:                true,
		AvailableMentoringReports: []web.MentoringReportData{},
	}
	if len(result) > 0 {
		payload.FirstID = result[0].ID
		payload.LastID = result[len(result)-1].ID
	}
	return &web.SuccessResponse{Code: http.StatusOK, Status: "OK", Data: payload}, nil
}

// FindById retrieves one master report with full competencies and report scores
func (s *MasterService) FindById(id uint) (*web.SuccessResponse, error) {
	var master domain.FullReport
	if err := s.DB.Preload("GapCompetencies.CompetencyType").
		Preload("ReportScores.AssessmentType").
		First(&master, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("master report not found")
		}
		s.Log.WithError(err).Error("failed to find master report by ID")
		return nil, fmt.Errorf("failed to retrieve master report: %w", err)
	}

	// Map competencies to web model
	competencies := make([]web.GapCompetencyData, 0, len(master.GapCompetencies))
	for _, gc := range master.GapCompetencies {
		competencies = append(competencies, web.GapCompetencyData{
			ID:               gc.ID,
			CompetencyTypeID: gc.CompetencyTypeID,
			CompetencyType: web.CompetencyTypeData{
				ID:   gc.CompetencyType.ID,
				Code: gc.CompetencyType.Code,
				Name: gc.CompetencyType.Name,
			},
		})
	}

	// Map report scores to web model
	reportScores := make([]web.ReportScoreData, 0, len(master.ReportScores))
	for _, rs := range master.ReportScores {
		if rs.AssessmentType != nil {
			reportScores = append(reportScores, web.ReportScoreData{
				ID:               rs.ID,
				Score:            rs.Score,
				AssessmentTypeID: rs.AssessmentTypeID,
				AssessmentType: web.AssessmentTypeData{
					ID:                 rs.AssessmentType.ID,
					AssessmentTypeName: rs.AssessmentType.AssessmentTypeName,
					StartTime:          rs.AssessmentType.StartTime,
					EndTime:            rs.AssessmentType.EndTime,
					MaxAttempts:        rs.AssessmentType.MaxAttempts,
				},
			})
		}
	}

	// Create response with competencies and report scores
	response := web.FullReportResponse{
		ID:                         master.ID,
		VesselName:                 master.VesselName,
		Nama:                       master.Nama,
		Jabatan:                    master.Jabatan,
		SeamanCode:                 master.SeamanCode,
		SeafarerCode:               master.SeafarerCode,
		Certificate:                master.Certificate,
		Age:                        master.Age,
		KonditeReview:              master.KonditeReview,
		KpiVessel:                  master.KpiVessel,
		PerformanceScore:           master.PerformanceScore,
		ValueAssessment:            master.ValueAssessment,
		AssessmentCenter:           master.AssessmentCenter,
		PotentialScore:             master.PotentialScore,
		HavQuadran:                 master.HavQuadran,
		HavMapping:                 master.HavMapping,
		CompetencyGapAnalysis:      master.CompetencyGapAnalysis,
		TotalGap:                   master.TotalGap,
		Strength:                   master.Strength,
		TalentClassified:           master.TalentClassified,
		IDPProgram:                 master.IDPProgram,
		HavQuadran2:                master.HavQuadran2,
		TalentClassified2:          master.TalentClassified2,
		Readiness:                  master.Readiness,
		CertificateEligible:        master.CertificateEligible,
		EducationFulfillmentMonths: master.EducationFulfillmentMonths,
		TotalReadinessUpdateMonths: master.TotalReadinessUpdateMonths,
		Keterangan:                 master.Keterangan,
		TmNm:                       master.TmNm,
		Competencies:               competencies,
		ReportScores:               reportScores,
	}

	return &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   response,
	}, nil
}

// BulkAssignBatch assigns a batch to multiple reports
func (s *MasterService) BulkAssignBatch(ctx context.Context, request *web.BulkAssignBatchRequest) (*web.SuccessResponse, error) {
	// Validate: when not using SelectAll, reportIds must be provided
	if !request.SelectAll && len(request.ReportIDs) == 0 {
		return nil, fmt.Errorf("reportIds harus diisi")
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	assignedCount := 0

	if request.SelectAll {
		// Assign all reports matching the current filters (global select-all)
		if err := s.MasterRepository.BulkAssignBatchAll(tx, request.BatchID, request.Query, request.FilterBatchID); err != nil {
			s.Log.WithError(err).Error("failed to bulk assign batch (all)")
			return nil, fmt.Errorf("failed to bulk assign batch: %w", err)
		}
	} else {
		if err := s.MasterRepository.BulkAssignBatch(tx, request.ReportIDs, request.BatchID); err != nil {
			s.Log.WithError(err).Error("failed to bulk assign batch")
			return nil, fmt.Errorf("failed to bulk assign batch: %w", err)
		}
		assignedCount = len(request.ReportIDs)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   map[string]interface{}{"assigned_count": assignedCount},
	}, nil
}

//
// ---------------------- CREATE ----------------------
//

// Ensure required defaults before create
func fillReportDefaults(r *domain.FullReport) error {
	if r.Readiness == nil || *r.Readiness == "" {
		def := "Pending"
		r.Readiness = &def
	}
	if r.SeamanCode == nil || *r.SeamanCode == "" {
		return fmt.Errorf("seamanCode is required")
	}
	// SeafarerCode is optional, no longer required
	return nil
}

// Create adds a new master report with competencies
func (s *MasterService) Create(request *web.MasterReportData) (*web.SuccessResponse, error) {
	if err := s.Validate.Struct(request); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	master := converter.MasterReportRequestToDomain(request)

	// Prevent duplicate reports: check if seaman_code already exists
	if master.SeamanCode != nil && *master.SeamanCode != "" {
		var existing domain.FullReport
		if err := s.DB.Where("seaman_code = ?", *master.SeamanCode).First(&existing).Error; err == nil {
			return nil, fmt.Errorf("report with seaman code '%s' already exists (ID: %d)", *master.SeamanCode, existing.ID)
		}
	}

	// Fill the row from CITRIX (seamen_cache + mutation_cache): the lookup in the form only sends
	// the name and seaman code, so without this every other column is created empty.
	if master.SeamanCode != nil && strings.TrimSpace(*master.SeamanCode) != "" {
		code := strings.TrimSpace(*master.SeamanCode)
		var cache domain.SeamenCache
		if err := s.DB.Where("seaman_code = ?", code).First(&cache).Error; err == nil {
			var mutations []domain.MutationCache
			if err := s.DB.Where("seaman_code = ?", code).Order("transaction_date DESC").Find(&mutations).Error; err != nil {
				s.Log.WithError(err).Warn("failed to load mutation history from CITRIX cache")
			}
			FillReportFromCitrix(master, &cache, mutations)
		} else {
			s.Log.WithField("seaman_code", code).Warn("seaman code not found in CITRIX cache; creating report without CITRIX data")
		}
	}

	// IDP program follows the position (MUALIM III/MASINIS IV=FDP, MUALIM II/MASINIS III=MDP,
	// MUALIM I/MASINIS II=SDP); other positions get no program.
	state, _ := ProgramState{Program: helpers.PtrToString(master.IDPProgram)}.ApplyPosition(helpers.PtrToString(master.Jabatan))
	master.IDPProgram = nullifyStringPtr(&state.Program)
	master.TrainingStatus = nullifyStringPtr(&state.Status)

	// Context: User wants new reports to be assigned to the currently filtered batch (if any)
	// We assume the BatchID is passed in the request body (MasterReportData)
	if request.BatchID != nil && *request.BatchID > 0 {
		master.BatchID = request.BatchID
	}

	if err := fillReportDefaults(master); err != nil {
		return nil, err
	}

	// Process competencies if provided
	if len(request.Competencies) > 0 {
		competencies := make([]domain.GapCompetency, 0, len(request.Competencies))

		for _, comp := range request.Competencies {
			competencies = append(competencies, domain.GapCompetency{
				CompetencyTypeID: comp.CompetencyTypeID,
			})
		}

		master.GapCompetencies = competencies
	}

	if err := s.MasterRepository.Create(s.DB, master); err != nil {
		s.Log.WithError(err).Error("failed to create master report")
		return nil, fmt.Errorf("failed to create master report: %w", err)
	}

	// Reload with competency types for response
	var created domain.FullReport
	if err := s.MasterRepository.FindById(s.DB, &created, master.ID); err != nil {
		s.Log.WithError(err).Warn("created but failed to reload with relations")
	} else {
		master = &created
	}

	// Map competencies for response
	competencies := make([]web.GapCompetencyData, 0, len(master.GapCompetencies))
	for _, gc := range master.GapCompetencies {
		competencies = append(competencies, web.GapCompetencyData{
			ID:               gc.ID,
			CompetencyTypeID: gc.CompetencyTypeID,
			CompetencyType: web.CompetencyTypeData{
				ID:   gc.CompetencyType.ID,
				Code: gc.CompetencyType.Code,
				Name: gc.CompetencyType.Name,
			},
		})
	}

	response := web.CreateReportResponse{
		ID:           master.ID,
		Nama:         master.Nama,
		SeafarerCode: master.SeafarerCode,
		Competencies: competencies,
	}

	return &web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "Created",
		Data:   response,
	}, nil
}

//
// ---------------------- UPDATE ----------------------
//

func nullifyStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	if strings.TrimSpace(*p) == "" {
		return nil
	}
	return p
}

func (s *MasterService) Update(id uint, request *web.UpdateMasterRequest) (*web.SuccessResponse, error) {
	// Validate input
	if err := s.Validate.Struct(request); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Get existing report (basic fields only, no preload)
	var existing domain.FullReport
	if err := s.DB.First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("master report not found")
		}
		return nil, err
	}

	// Prevent breaking FK
	if request.SeafarerCode != nil && existing.SeafarerCode != nil &&
		*request.SeafarerCode != *existing.SeafarerCode {
		return nil, fmt.Errorf("cannot update seafarerCode; referenced by other tables")
	}

	// Apply normal fields update
	if request.Nama != nil {
		existing.Nama = nullifyStringPtr(request.Nama)
	}
	if request.Jabatan != nil {
		existing.Jabatan = nullifyStringPtr(request.Jabatan)
	}
	if request.VesselName != nil {
		existing.VesselName = nullifyStringPtr(request.VesselName)
	}
	if request.SeamanCode != nil {
		existing.SeamanCode = nullifyStringPtr(request.SeamanCode)
	}
	if request.Age != nil {
		existing.Age = nullifyStringPtr(request.Age)
	}
	if request.Certificate != nil {
		existing.Certificate = nullifyStringPtr(request.Certificate)
	}
	if request.IDPProgram != nil {
		newProgram := strings.ToUpper(strings.TrimSpace(*request.IDPProgram))
		oldProgram := strings.ToUpper(strings.TrimSpace(helpers.PtrToString(existing.IDPProgram)))
		existing.IDPProgram = nullifyStringPtr(request.IDPProgram)
		// Choosing a program by hand (re-)enrols the person, also after they left one.
		if programLevel(newProgram) > 0 && newProgram != oldProgram {
			ongoing := domain.TrainingStatusOngoing
			existing.TrainingStatus = &ongoing
			existing.ExitedProgram = nil
			existing.ProgramExitedAt = nil
			notCompleted := false
			existing.CompletedBeforeExit = &notCompleted
		}
	}
	if request.Jabatan != nil {
		// A position edited by hand follows the same promotion rule as a CITRIX sync.
		state, exited := ProgramState{
			Program:       helpers.PtrToString(existing.IDPProgram),
			Status:        helpers.PtrToString(existing.TrainingStatus),
			ExitedProgram: helpers.PtrToString(existing.ExitedProgram),
		}.ApplyPosition(helpers.PtrToString(existing.Jabatan))
		if exited || programLevel(state.Program) > 0 {
			existing.IDPProgram = nullifyStringPtr(&state.Program)
			existing.TrainingStatus = nullifyStringPtr(&state.Status)
			existing.ExitedProgram = nullifyStringPtr(&state.ExitedProgram)
			if exited {
				now := time.Now()
				existing.ProgramExitedAt = &now
				completed := state.CompletedBeforeExit
				existing.CompletedBeforeExit = &completed
			}
		}
	}
	if request.PerformanceScore != nil {
		existing.PerformanceScore = request.PerformanceScore
	}
	if request.TotalReadinessUpdateMonths != nil {
		existing.TotalReadinessUpdateMonths = request.TotalReadinessUpdateMonths
	}
	if request.Readiness != nil {
		existing.Readiness = nullifyStringPtr(request.Readiness)
	}
	if request.TalentClassified != nil {
		existing.TalentClassified = nullifyStringPtr(request.TalentClassified)
	}
	// Note: MentoringReportID is not stored in reports table
	// Mentoring programs are linked via mentee_names in mentoring_reports table

	// Save master report basic fields
	if err := s.DB.Save(&existing).Error; err != nil {
		return nil, fmt.Errorf("failed to update master report: %w", err)
	}

	// UPDATE COMPETENCIES if the field is present in request
	if request.CompetencyUpdateRequests != nil {
		s.Log.Infof("Updating competencies for report ID: %d (received %d competencies)", id, len(request.CompetencyUpdateRequests))

		// Start transaction for atomic operations
		err := s.DB.Transaction(func(tx *gorm.DB) error {
			// STEP 1: DELETE ALL existing gap_competencies for this report
			deleteResult := tx.Where("report_id = ?", id).Delete(&domain.GapCompetency{})
			if deleteResult.Error != nil {
				s.Log.WithError(deleteResult.Error).Error("failed to delete existing gap competencies")
				return fmt.Errorf("failed to delete existing gap competencies: %w", deleteResult.Error)
			}

			s.Log.Infof("✅ Deleted %d existing competencies", deleteResult.RowsAffected)

			// STEP 2: CREATE new competencies if array is not empty
			if len(request.CompetencyUpdateRequests) > 0 {
				s.Log.Infof("📝 Processing %d new competencies", len(request.CompetencyUpdateRequests))
				newGapCompetencies := make([]domain.GapCompetency, 0, len(request.CompetencyUpdateRequests))

				for i, comp := range request.CompetencyUpdateRequests {
					var competencyTypeID int

					// Option 1: Using existing competency type ID
					if comp.CompetencyTypeID != nil && *comp.CompetencyTypeID > 0 {
						var existingType domain.CompetencyType
						if err := tx.First(&existingType, *comp.CompetencyTypeID).Error; err != nil {
							if errors.Is(err, gorm.ErrRecordNotFound) {
								return fmt.Errorf("competency type with ID %d not found", *comp.CompetencyTypeID)
							}
							return fmt.Errorf("error checking competency type: %w", err)
						}
						competencyTypeID = *comp.CompetencyTypeID
						s.Log.Infof("  [%d] Using existing competency type ID: %d", i+1, competencyTypeID)

						// Option 2: Using code (find or create)
					} else if comp.Code != nil && *comp.Code != "" {
						code := strings.ToUpper(strings.TrimSpace(*comp.Code))

						var competencyType domain.CompetencyType
						err := tx.Where("code = ?", code).First(&competencyType).Error

						if err != nil {
							if errors.Is(err, gorm.ErrRecordNotFound) {
								// Create new competency type
								name := code
								if comp.Name != nil && *comp.Name != "" {
									name = *comp.Name
								}

								description := ""
								if comp.Description != nil {
									description = *comp.Description
								}

								category := "M"
								if comp.Category != nil && *comp.Category != "" {
									cat := strings.ToUpper(*comp.Category)
									if cat == "M" || cat == "NM" {
										category = cat
									}
								}

								newCompetencyType := domain.CompetencyType{
									Code:        code,
									Name:        name,
									Description: description,
									Category:    category,
									IsActive:    true,
								}

								if err := tx.Create(&newCompetencyType).Error; err != nil {
									s.Log.WithError(err).Errorf("failed to create competency type: %s", code)
									return fmt.Errorf("failed to create competency type '%s': %w", code, err)
								}

								competencyTypeID = newCompetencyType.ID
								s.Log.Infof("  [%d] ✨ Created new competency type: %s (ID: %d)", i+1, code, competencyTypeID)
							} else {
								return fmt.Errorf("error checking competency type: %w", err)
							}
						} else {
							// Use existing and optionally update fields
							competencyTypeID = competencyType.ID

							needUpdate := false
							if comp.Name != nil && *comp.Name != "" && *comp.Name != competencyType.Name {
								competencyType.Name = *comp.Name
								needUpdate = true
							}
							if comp.Description != nil && *comp.Description != "" && *comp.Description != competencyType.Description {
								competencyType.Description = *comp.Description
								needUpdate = true
							}
							if comp.Category != nil && *comp.Category != "" {
								cat := strings.ToUpper(*comp.Category)
								if (cat == "M" || cat == "NM") && cat != competencyType.Category {
									competencyType.Category = cat
									needUpdate = true
								}
							}

							if needUpdate {
								if err := tx.Save(&competencyType).Error; err != nil {
									s.Log.WithError(err).Warn("failed to update competency type fields")
								} else {
									s.Log.Infof("  [%d] Updated competency type: %s", i+1, code)
								}
							} else {
								s.Log.Infof("  [%d] Using existing competency type: %s (ID: %d)", i+1, code, competencyTypeID)
							}
						}
					} else {
						s.Log.Warnf("  [%d] ⚠️  Competency has no ID or Code, skipping", i+1)
						continue
					}

					// Add to batch
					newGapCompetencies = append(newGapCompetencies, domain.GapCompetency{
						ReportID:         int(id),
						CompetencyTypeID: competencyTypeID,
					})
				}

				// Bulk insert with ON DUPLICATE KEY UPDATE to prevent constraint violations
				if len(newGapCompetencies) > 0 {
					if err := tx.Clauses(clause.OnConflict{
						Columns:   []clause.Column{{Name: "report_id"}, {Name: "competency_type_id"}},
						DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
					}).Create(&newGapCompetencies).Error; err != nil {
						s.Log.WithError(err).Error("failed to create gap competencies")
						return fmt.Errorf("failed to create gap competencies: %w", err)
					}
					s.Log.Infof("✅ Successfully created %d new gap competencies", len(newGapCompetencies))
				}
			} else {
				s.Log.Infof("📭 Empty competencies array - all competencies removed")
			}

			return nil
		})

		if err != nil {
			s.Log.WithError(err).Error("Transaction failed")
			return nil, err
		}

		s.Log.Info("✅ Transaction committed successfully")
	}

	// UPDATE REPORT SCORES if the field is present in request
	if len(request.ReportScores) > 0 {
		s.Log.Infof("Updating report scores for report ID: %d (received %d scores)", id, len(request.ReportScores))

		// Start transaction for atomic operations
		err := s.DB.Transaction(func(tx *gorm.DB) error {
			// STEP 1: DELETE ALL existing report scores for this report
			deleteResult := tx.Where("report_id = ?", id).Delete(&domain.ReportScore{})
			if deleteResult.Error != nil {
				s.Log.WithError(deleteResult.Error).Error("failed to delete existing report scores")
				return fmt.Errorf("failed to delete existing report scores: %w", deleteResult.Error)
			}

			s.Log.Infof("✅ Deleted %d existing report scores", deleteResult.RowsAffected)

			// STEP 2: CREATE new report scores if array is not empty
			if len(request.ReportScores) > 0 {
				s.Log.Infof("📝 Processing %d new report scores", len(request.ReportScores))
				newReportScores := make([]domain.ReportScore, 0, len(request.ReportScores))

				for i, scoreData := range request.ReportScores {
					// Get assessment type by name
					var assessmentType domain.AssessmentType
					if scoreData.AssessmentType.AssessmentTypeName != "" {
						if err := tx.Where("assessment_type_name = ?", scoreData.AssessmentType.AssessmentTypeName).First(&assessmentType).Error; err != nil {
							if errors.Is(err, gorm.ErrRecordNotFound) {
								s.Log.Warnf("  [%d] ⚠️  Assessment type '%s' not found, skipping", i+1, scoreData.AssessmentType.AssessmentTypeName)
								continue
							}
							return fmt.Errorf("error checking assessment type: %w", err)
						}
					} else {
						s.Log.Warnf("  [%d] ⚠️  Report score has no assessment type name, skipping", i+1)
						continue
					}

					// Add to batch
					newReportScores = append(newReportScores, domain.ReportScore{
						ReportID:         int64(id),
						Score:            scoreData.Score,
						AssessmentTypeID: assessmentType.ID,
					})
					s.Log.Infof("  [%d] Added report score: %s = %d", i+1, scoreData.AssessmentType.AssessmentTypeName, scoreData.Score)
				}

				// Bulk insert
				if len(newReportScores) > 0 {
					if err := tx.Create(&newReportScores).Error; err != nil {
						s.Log.WithError(err).Error("failed to create report scores")
						return fmt.Errorf("failed to create report scores: %w", err)
					}
					s.Log.Infof("✅ Successfully created %d new report scores", len(newReportScores))
				}
			} else {
				s.Log.Infof("📭 Empty report scores array - all scores removed")
			}

			return nil
		})

		if err != nil {
			s.Log.WithError(err).Error("Transaction failed for report scores")
			return nil, err
		}

		s.Log.Info("✅ Report scores transaction committed successfully")
	}

	// CRITICAL FIX: Create a COMPLETELY FRESH database session to avoid cache
	// This ensures we get the latest data from database
	freshDB := s.DB.Session(&gorm.Session{NewDB: true})

	// Reload data with fresh session using repository
	var finalResult domain.FullReport
	if err := s.MasterRepository.FindById(freshDB, &finalResult, id); err != nil {
		s.Log.WithError(err).Error("failed to reload report with relations")
		return nil, fmt.Errorf("failed to reload report: %w", err)
	}

	s.Log.Infof("🔄 Reloaded report - found %d competencies in response", len(finalResult.GapCompetencies))

	// Build response
	competencies := make([]web.GapCompetencyData, 0, len(finalResult.GapCompetencies))
	for _, gc := range finalResult.GapCompetencies {
		competencyTypeData := web.CompetencyTypeData{}
		if gc.CompetencyType != nil {
			competencyTypeData = web.CompetencyTypeData{
				ID:          gc.CompetencyType.ID,
				Code:        gc.CompetencyType.Code,
				Name:        gc.CompetencyType.Name,
				Description: gc.CompetencyType.Description,
				Category:    gc.CompetencyType.Category,
				IsActive:    gc.CompetencyType.IsActive,
			}
		}

		competencies = append(competencies, web.GapCompetencyData{
			ID:               gc.ID,
			CompetencyTypeID: gc.CompetencyTypeID,
			CompetencyType:   competencyTypeData,
		})
	}

	response := web.UpdateReportResponse{
		ID:           finalResult.ID,
		Nama:         finalResult.Nama,
		Jabatan:      finalResult.Jabatan,
		Competencies: competencies,
	}

	return &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Updated",
		Data:   response,
	}, nil
}

// ---------------------- DELETE ----------------------

func (s *MasterService) Delete(id uint) (*web.SuccessResponse, error) {

	var master domain.FullReport
	if err := s.MasterRepository.FindById(s.DB, &master, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("master report not found")
		}
		return nil, err
	}

	// Delete all related tables using transaction
	var competenciesDeleted int64
	var scoresDeleted int64

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Delete gap competencies
		result := tx.Where("report_id = ?", id).Delete(&domain.GapCompetency{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete gap competencies: %w", result.Error)
		}
		competenciesDeleted = result.RowsAffected

		// Delete report scores
		result = tx.Where("report_id = ?", id).Delete(&domain.ReportScore{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete report scores: %w", result.Error)
		}
		scoresDeleted = result.RowsAffected

		// Delete the master report
		if err := tx.Delete(&master).Error; err != nil {
			return fmt.Errorf("failed to delete master report: %w", err)
		}

		return nil
	})

	if err != nil {
		s.Log.WithError(err).Error("failed to delete master report and related data")
		return nil, err
	}

	s.Log.WithFields(logrus.Fields{
		"report_id":           id,
		"competenciesDeleted": competenciesDeleted,
		"scoresDeleted":       scoresDeleted,
		"time":                time.Now(),
	}).Info("Deleted master report and all related data")

	return &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "Deleted",
		Data: gin.H{
			"message":             fmt.Sprintf("Master report %d deleted", id),
			"competenciesDeleted": competenciesDeleted,
			"scoresDeleted":       scoresDeleted,
		},
	}, nil
}

// GetMentoringProgramsByPersonName retrieves all mentoring programs for a specific person
func (s *MasterService) GetMentoringProgramsByPersonName(personName string) (*web.SuccessResponse, error) {
	var mentoringReports []domain.MentoringReport

	// Search for mentoring reports where the person's name appears in mentee_names JSON array
	if err := s.DB.Where("mentee_names LIKE ?", "%\""+personName+"\"%").Find(&mentoringReports).Error; err != nil {
		s.Log.WithError(err).Error("failed to find mentoring reports by person name")
		return nil, fmt.Errorf("failed to retrieve mentoring reports: %w", err)
	}

	// Convert to web response format
	result := make([]web.MentoringReportData, 0, len(mentoringReports))
	for _, mr := range mentoringReports {
		result = append(result, *s.toMentoringReportWeb(&mr))
	}

	return &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   result,
	}, nil
}

// overlayCitrixData fills blank vessel/position/age/certificate/... of the given rows from
// seamen_cache, with a single query for the whole page. It is read-only (nothing is saved);
// "Refresh Personal Data" / the CITRIX sync still persist the values.
func (s *MasterService) overlayCitrixData(rows []domain.MasterReport) {
	codes := make([]string, 0, len(rows))
	for _, r := range rows {
		if code := strings.TrimSpace(r.SeamanCode); code != "" {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return
	}

	var caches []domain.SeamenCache
	if err := s.DB.Where("seaman_code IN ?", codes).Find(&caches).Error; err != nil {
		s.Log.WithError(err).Warn("failed to load CITRIX data for master rows")
		return
	}
	byCode := make(map[string]*domain.SeamenCache, len(caches))
	for i := range caches {
		byCode[caches[i].SeamanCode] = &caches[i]
	}
	for i := range rows {
		if cache, ok := byCode[strings.TrimSpace(rows[i].SeamanCode)]; ok {
			overlayCitrixOnMasterRow(&rows[i], cache)
		}
	}
}
