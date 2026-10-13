package web

import (
	"strconv"
)

// CompetencyTypeData represents competency type information

type CompetencyTypeData struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsActive    bool   `json:"isActive"`
}

// GapCompetencyData represents gap competency with its type
type GapCompetencyData struct {
	ID               int                `json:"id"`
	CompetencyTypeID int                `json:"competencyTypeId"`
	CompetencyType   CompetencyTypeData `json:"competencyType"`
}

// ReportScoreData represents score for an assessment type
type ReportScoreData struct {
	ID               int                `json:"id"`
	Score            int                `json:"score"`
	AssessmentTypeID uint64             `json:"assessmentTypeId"`
	AssessmentType   AssessmentTypeData `json:"assessmentType,omitempty"`
}

// Mirrors domain.MasterReport 1:1 so the frontend gets all columns.
type MasterReportData struct {
	ID                         int                  `json:"id"`
	VesselName                 string               `json:"vesselName"`
	Nama                       string               `json:"nama"`
	Jabatan                    string               `json:"jabatan"`
	User                       string               `json:"user"`
	SeamanCode                 string               `json:"seamanCode"`
	SeafarerCode               string               `json:"seafarerCode"`
	Certificate                string               `json:"certificate"`
	Age                        string               `json:"age"`
	KonditeReview              int                  `json:"konditeReview"`
	KpiVessel                  int                  `json:"kpiVessel"`
	PerformanceScore           int                  `json:"performanceScore"`
	ValueAssessment            int                  `json:"valueAssessment"`
	AssessmentCenter           int                  `json:"assessmentCenter"`
	PotentialScore             int                  `json:"potentialScore"`
	HavQuadran                 int                  `json:"havQuadran"`
	HavMapping                 string               `json:"havMapping"`
	CompetencyGapAnalysis      string               `json:"competencyGapAnalysis"`
	TotalGap                   int                  `json:"totalGap"`
	Strength                   string               `json:"strength"`
	TalentClassified           string               `json:"talentClassified"`
	IDPProgram                 string               `json:"idpProgram"`
	TrainingStatus             string               `json:"trainingStatus"`
	ExitedProgram              string               `json:"exitedProgram"`
	HavQuadran2                int                  `json:"havQuadran2"`
	TalentClassified2          string               `json:"talentClassified2"`
	Readiness                  string               `json:"readiness"`
	CertificateEligible        string               `json:"certificateEligible"`
	EducationFulfillmentMonths *int                 `json:"educationFulfillmentMonths"`
	TotalReadinessUpdateMonths *int                 `json:"totalReadinessUpdateMonths"`
	Keterangan                 string               `json:"keterangan"`
	TmNm                       *string              `json:"tmNm"`
	MentoringReportID          *int64               `json:"mentoringReportId,omitempty"`
	MentoringReport            *MentoringReportData `json:"mentoringReport,omitempty"`
	Competencies               []GapCompetencyData  `json:"competencies,omitempty"`
	ReportScores               []ReportScoreData    `json:"reportScores,omitempty"`
	BatchID                    *int                 `json:"batchId"`
}

type MasterListRequest struct {
	AnchorID     int    `form:"anchor_id" validate:"min=0"`
	Page         string `form:"page" validate:"required,oneof=next prev"`
	PageSize     int    `form:"page_size" validate:"required,min=1,max=200"`
	Name         string `form:"name"`
	SeafarerCode string `form:"seafarer_code"`
	Query        string `form:"query"`
	BatchID      int    `form:"batch_id"`
}

type DeleteMasterRequest struct {
	ID uint `json:"id"`
}

func (r *DeleteMasterRequest) ParseID(param string) error {
	id, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return err
	}
	r.ID = uint(id)
	return nil
}

type UpdateMasterRequest struct {
	ID                         uint    `json:"-"`
	Nama                       *string `json:"nama"`
	SeafarerCode               *string `json:"seafarerCode"`
	SeamanCode                 *string `json:"seamanCode"`
	Jabatan                    *string `json:"jabatan"`
	VesselName                 *string `json:"vesselName"`
	StartDate                  *string `json:"startDate"`
	Age                        *string `json:"age"`
	Certificate                *string `json:"certificate"`
	IDPProgram                 *string `json:"idpProgram"`
	PerformanceScore           *int    `json:"performanceScore"`
	Readiness                  *string `json:"readiness"`
	TotalReadinessUpdateMonths *int    `json:"totalReadinessUpdateMonths"`
	TalentClassified           *string `json:"talentClassified"`
	MentoringReportID          *int64  `json:"mentoringReportId"` // Link to mentoring report
	// only competency types
	CompetencyUpdateRequests []CompetencyUpdateRequest `json:"competencies"`
	// Assessment type scores
	ReportScores []ReportScoreData `json:"reportScores"`
}

func (r *UpdateMasterRequest) ParseID(param string) error {
	id, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return err
	}
	r.ID = uint(id)
	return nil
}

type MasterReportListResponse struct {
	Data                      []MasterReportData    `json:"data"`
	FirstID                   int                   `json:"firstId,omitempty"`
	LastID                    int                   `json:"lastId,omitempty"`
	PageSize                  int                   `json:"pageSize"`
	HasMore                   bool                  `json:"hasMore"`
	FirstPage                 bool                  `json:"firstPage"`
	Total                     int                   `json:"total"`
	IsArchived                bool                  `json:"isArchived"`
	AvailableMentoringReports []MentoringReportData `json:"availableMentoringReports,omitempty"`
}

// ReportData is the request body for creating a new report

// FullReportResponse is the detailed response for FindById
type FullReportResponse struct {
	ID                         uint                `json:"id"`
	VesselName                 *string             `json:"vesselName"`
	Nama                       *string             `json:"nama"`
	Jabatan                    *string             `json:"jabatan"`
	SeamanCode                 *string             `json:"seamanCode"`
	SeafarerCode               *string             `json:"seafarerCode"`
	Certificate                *string             `json:"certificate"`
	Age                        *string             `json:"age"`
	KonditeReview              *int                `json:"konditeReview"`
	KpiVessel                  *int                `json:"kpiVessel"`
	PerformanceScore           *int                `json:"performanceScore"`
	ValueAssessment            *int                `json:"valueAssessment"`
	AssessmentCenter           *int                `json:"assessmentCenter"`
	PotentialScore             *int                `json:"potentialScore"`
	HavQuadran                 *int                `json:"havQuadran"`
	HavMapping                 *string             `json:"havMapping"`
	CompetencyGapAnalysis      *string             `json:"competencyGapAnalysis"`
	TotalGap                   *int                `json:"totalGap"`
	Strength                   *string             `json:"strength"`
	TalentClassified           *string             `json:"talentClassified"`
	IDPProgram                 *string             `json:"idpProgram"`
	HavQuadran2                *int                `json:"havQuadran2"`
	TalentClassified2          *string             `json:"talentClassified2"`
	Readiness                  *string             `json:"readiness"`
	CertificateEligible        *string             `json:"certificateEligible"`
	EducationFulfillmentMonths *int                `json:"educationFulfillmentMonths"`
	TotalReadinessUpdateMonths *int                `json:"totalReadinessUpdateMonths"`
	Keterangan                 *string             `json:"keterangan"`
	TmNm                       *string             `json:"tmNm"`
	Competencies               []GapCompetencyData `json:"competencies"`
	ReportScores               []ReportScoreData   `json:"reportScores,omitempty"`
	BatchID                    *int                `json:"batchId"`
}

// CreateReportResponse is the response after creating a report
type CreateReportResponse struct {
	ID           uint                `json:"id"`
	Nama         *string             `json:"nama"`
	SeafarerCode *string             `json:"seafarerCode"`
	Competencies []GapCompetencyData `json:"competencies"`
}

// UpdateReportResponse is the response after updating a report
type UpdateReportResponse struct {
	ID           uint                `json:"id"`
	Nama         *string             `json:"nama"`
	Jabatan      *string             `json:"jabatan"`
	Competencies []GapCompetencyData `json:"competencies"`
}

// All fields are pointers to handle omitempty and distinguish between not provided and empty string
type CompetencyUpdateRequest struct {
	CompetencyTypeID *int    `json:"competencyTypeId,omitempty"` // Use existing competency type by ID
	Code             *string `json:"code,omitempty"`             // Create or find by code
	Name             *string `json:"name,omitempty"`             // Name for new/update competency type
	Description      *string `json:"description,omitempty"`      // Description for new/update
	Category         *string `json:"category,omitempty"`         // M or NM
}

type BulkAssignBatchRequest struct {
	ReportIDs    []uint `json:"reportIds"`   // IDs to assign (ignored when SelectAll=true)
	BatchID      *int   `json:"batchId"`     // Pointer to allow nullifying batch assignment
	SelectAll    bool   `json:"selectAll"`   // If true, assign all matching records by filter
	Query        string `json:"query"`       // Name/seafarer code filter (used with SelectAll)
	FilterBatchID *int  `json:"filterBatchId"` // Current batch filter (used with SelectAll)
}
