package domain

import (
	"time"
)

type MasterReport struct {
	ID                         int     `json:"id" gorm:"column:id;primaryKey"`
	VesselName                 string  `json:"vesselName" gorm:"column:vessel_name"`
	Nama                       string  `json:"nama" gorm:"column:nama"`
	Jabatan                    string  `json:"jabatan" gorm:"column:jabatan"`
	User                       string  `json:"user" gorm:"column:user"`
	SeamanCode                 string  `json:"seamanCode" gorm:"column:seaman_code"`
	SeafarerCode               string  `json:"seafarerCode" gorm:"column:seafarer_code"`
	Certificate                string  `json:"certificate" gorm:"column:certificate"`
	Age                        string  `json:"age" gorm:"column:age"`
	KonditeReview              int     `json:"konditeReview" gorm:"column:kondite_review"`
	KpiVessel                  int     `json:"kpiVessel" gorm:"column:kpi_vessel"`
	PerformanceScore           int     `json:"performanceScore" gorm:"column:performance_score"`
	ValueAssessment            int     `json:"valueAssessment" gorm:"column:value_assessment"`
	AssessmentCenter           int     `json:"assessmentCenter" gorm:"column:assessment_center"`
	PotentialScore             int     `json:"potentialScore" gorm:"column:potential_score"`
	HavQuadran                 int     `json:"havQuadran" gorm:"column:hav_quadran"`
	HavMapping                 string  `json:"havMapping" gorm:"column:hav_mapping"`
	CompetencyGapAnalysis      string  `json:"competencyGapAnalysis" gorm:"column:competency_gap_analysis"`
	TotalGap                   int     `json:"totalGap" gorm:"column:total_gap"`
	Strength                   string  `json:"strength" gorm:"column:strength"`
	TalentClassified           string  `json:"talentClassified" gorm:"column:talent_classified"`
	IDPProgram                 string  `json:"idpProgram" gorm:"column:idp_program"`
	TrainingStatus             string  `json:"trainingStatus" gorm:"column:training_status"`
	ExitedProgram              string  `json:"exitedProgram" gorm:"column:exited_program"`
	HavQuadran2                int     `json:"havQuadran2" gorm:"column:hav_quadran2"`
	TalentClassified2          string  `json:"talentClassified2" gorm:"column:talent_classified2"`
	Readiness                  string  `json:"readiness" gorm:"column:readiness"`
	CertificateEligible        string  `json:"certificateEligible" gorm:"column:certificate_eligible"`
	EducationFulfillmentMonths *int    `json:"educationFulfillmentMonths" gorm:"column:education_fulfillment_months"`
	TotalReadinessUpdateMonths *int    `json:"totalReadinessUpdateMonths" gorm:"column:total_readiness_update_months"`
	Keterangan                 string  `json:"keterangan" gorm:"column:keterangan"`
	TmNm                       *string `json:"tmNm" gorm:"column:tm_nm"`
	StartDate                  string  `json:"startDate" gorm:"column:start_date"` //
	//optional, but if provided must be a valid date
	MentoringReportID *int64           `json:"mentoringReportId" gorm:"-"` // Not stored in DB, linked via mentee_names
	GapCompetencies   []GapCompetency  `json:"gapCompetencies" gorm:"foreignKey:ReportID"`
	ReportScores      []ReportScore    `json:"reportScores" gorm:"foreignKey:ReportID;references:ID"`
	MentoringReport   *MentoringReport `json:"mentoringReport,omitempty" gorm:"-"` // Not stored in DB, fetched via API
	BatchID           *int             `json:"batchId" gorm:"column:batch_id"`
	Batch             *Batch           `json:"batch,omitempty" gorm:"foreignKey:BatchID;references:ID"`
}

type FullReport struct {
	ID      uint    `gorm:"primaryKey;autoIncrement;column:id"`
	Nama    *string `gorm:"column:nama;default:null"`
	Jabatan *string `gorm:"column:jabatan;default:null"`
	User    *string `gorm:"column:user;default:null"`

	SeamanCode   *string `gorm:"column:seaman_code;type:varchar(50);default:null;uniqueIndex:idx_seaman_code"`
	SeafarerCode *string `gorm:"column:seafarer_code;default:null"`
	VesselName   *string `gorm:"column:vessel_name;default:null"`
	Certificate  *string `gorm:"column:certificate;default:null"`
	Age          *string `gorm:"column:age;default:null"`

	TanggalLahir *string `gorm:"column:tanggal_lahir;default:null"`

	StartDate *time.Time `gorm:"column:start_date;default:null"`

	WarningLetter *string `gorm:"column:warning_letter;default:null"`
	CaseHistory   *string `gorm:"column:case_history;default:null"`
	YearOfCase    *string `gorm:"column:year_of_case;default:null"`
	VesselHistory *string `gorm:"column:vessel_history;default:null"`

	KonditeReview         *int       `gorm:"column:kondite_review;default:0"`
	KpiVessel             *int       `gorm:"column:kpi_vessel;default:0"`
	PerformanceScore      *int       `gorm:"column:performance_score;default:0"`
	ValueAssessment       *int       `gorm:"column:value_assessment;default:0"`
	AssessmentCenter      *int       `gorm:"column:assessment_center;default:0"`
	PotentialScore        *int       `gorm:"column:potential_score;default:0"`
	HavQuadran            *int       `gorm:"column:hav_quadran;default:0"`
	HavMapping            *string    `gorm:"column:hav_mapping;default:null"`
	CompetencyGapAnalysis *string    `gorm:"column:competency_gap_analysis;default:null"`
	TotalGap              *int       `gorm:"column:total_gap;default:0"`
	Strength              *string    `gorm:"column:strength;default:null"`
	TalentClassified      *string    `gorm:"column:talent_classified;default:null"`
	IDPProgram            *string    `gorm:"column:idp_program;default:null"`
	TrainingStatus        *string    `gorm:"column:training_status;default:null"`
	ExitedProgram         *string    `gorm:"column:exited_program;default:null"`
	ProgramExitedAt       *time.Time `gorm:"column:program_exited_at;default:null"`
	CompletedBeforeExit   *bool      `gorm:"column:completed_before_exit;default:false"`
	HavQuadran2           *int       `gorm:"column:hav_quadran2;default:0"`
	TalentClassified2     *string    `gorm:"column:talent_classified2;default:null"`
	Readiness             *string    `gorm:"column:readiness;default:null"`
	CertificateEligible   *string    `gorm:"column:certificate_eligible;default:null"`

	TrainingCompleted  *string `gorm:"column:training_completed;default:null"`
	TrainingPlanned    *string `gorm:"column:training_planned;default:null"`
	MentoringCompleted *string `gorm:"column:mentoring_completed;default:null"`
	MentoringPlanned   *string `gorm:"column:mentoring_planned;default:null"`
	CoachingCompleted  *string `gorm:"column:coaching_completed;default:null"`
	CoachingPlanned    *string `gorm:"column:coaching_planned;default:null"`

	DataIncumbent    *string `gorm:"column:data_incumbent;default:null"`
	SuccessionVessel *string `gorm:"column:succession_vessel;default:null"`
	SuccessionRank   *string `gorm:"column:succession_rank;default:null"`

	IDPStart  *string `gorm:"column:idp_start;default:null"`
	IDPMentor *string `gorm:"column:idp_mentor;default:null"`
	IDPCoach  *string `gorm:"column:idp_coach;default:null"`

	ReadinessMonth             *int `gorm:"column:readiness_month;default:0"`
	EducationFulfillmentMonths *int `gorm:"column:education_fulfillment_months;default:0"`
	TotalReadinessUpdateMonths *int `gorm:"column:total_readiness_update_months;default:0"`

	Keterangan *string `gorm:"column:keterangan;default:null"`
	TmNm       *string `gorm:"column:tm_nm;default:null"`

	MentoringReportID *int64 `json:"mentoringReportId" gorm:"-"` // Not stored in DB, linked via mentee_names

	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time        `gorm:"column:updated_at;autoUpdateTime"`
	GapCompetencies []GapCompetency  `json:"gapCompetencies" gorm:"foreignKey:ReportID"`
	ReportScores    []ReportScore    `json:"reportScores" gorm:"foreignKey:ReportID;references:ID"`
	MentoringReport *MentoringReport `json:"mentoringReport,omitempty" gorm:"-"` // Not stored in DB, fetched via API
	BatchID         *int             `json:"batchId" gorm:"column:batch_id"`
	Batch           *Batch           `json:"batch,omitempty" gorm:"foreignKey:BatchID;references:ID"`
}

func (FullReport) TableName() string {
	return "reports"
}

func (MasterReport) TableName() string {
	return "reports"
}
