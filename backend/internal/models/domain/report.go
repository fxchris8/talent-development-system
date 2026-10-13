package domain

import "time"

type Report struct {
	ID           int        `json:"id" gorm:"column:id;primaryKey"`
	SeamanCode   string     `json:"seamanCode" gorm:"column:seaman_code;type:varchar(50);not null;uniqueIndex:idx_seaman_code"`
	SeafarerCode string     `json:"seafarerCode" gorm:"column:seafarer_code"`
	Nama         string     `json:"nama" gorm:"column:nama"`
	IDP          string     `json:"idp" gorm:"column:idp"`
	Jabatan      string     `json:"jabatan" gorm:"column:jabatan"`
	VesselName   string     `json:"vesselName" gorm:"column:vessel_name"`
	Certificate  string     `json:"certificate" gorm:"column:certificate"`
	Age          string     `json:"age" gorm:"column:age"`
	TanggalLahir string     `json:"tanggalLahir" gorm:"column:tanggal_lahir"`
	StartDate    *time.Time `json:"startDate" gorm:"column:start_date;type:date"`

	// Catatan Indisipliner
	WarningLetter string `json:"warningLetter" gorm:"column:warning_letter"`
	CaseHistory   string `json:"caseHistory" gorm:"column:case_history"`
	YearOfCase    string `json:"yearOfCase" gorm:"column:year_of_case"`

	// Data History Vessel (sementara string/JSON)
	VesselHistory string `json:"vesselHistory" gorm:"column:vessel_history"`

	// Kinerja
	KonditeReview         int    `json:"konditeReview" gorm:"column:kondite_review"`
	KpiVessel             int    `json:"kpiVessel" gorm:"column:kpi_vessel"`
	PerformanceScore      int    `json:"performanceScore" gorm:"column:performance_score"`
	ValueAssessment       int    `json:"valueAssessment" gorm:"column:value_assessment"`
	AssessmentCenter      int    `json:"assessmentCenter" gorm:"column:assessment_center"`
	PotentialScore        int    `json:"potentialScore" gorm:"column:potential_score"`
	HavQuadran            int    `json:"havQuadran" gorm:"column:hav_quadran"`
	HavMapping            string `json:"havMapping" gorm:"column:hav_mapping"`
	CompetencyGapAnalysis string `json:"competencyGapAnalysis" gorm:"column:competency_gap_analysis"`
	TotalGap              int    `json:"totalGap" gorm:"column:total_gap"`
	Strength              string `json:"strength" gorm:"column:strength"`
	TalentClassified      string `json:"talentClassified" gorm:"column:talent_classified"`
	IDPProgram            string `json:"idpProgram" gorm:"column:idp_program"`
	// Position in the IDP program: Ongoing / Completed / Exited (promoted before finishing).
	TrainingStatus      string     `json:"trainingStatus" gorm:"column:training_status;size:20"`
	ExitedProgram       string     `json:"exitedProgram" gorm:"column:exited_program;size:10"`
	ProgramExitedAt     *time.Time `json:"programExitedAt" gorm:"column:program_exited_at"`
	// True when the person had finished every training before the promotion took them out.
	CompletedBeforeExit bool       `json:"completedBeforeExit" gorm:"column:completed_before_exit;not null;default:false"`
	// When the status became Completed (all trainings of the program done).
	ProgramCompletedAt  *time.Time `json:"programCompletedAt" gorm:"column:program_completed_at"`
	HavQuadran2         int        `json:"havQuadran2" gorm:"column:hav_quadran2"`
	TalentClassified2   string     `json:"talentClassified2" gorm:"column:talent_classified2"`
	Readiness           string     `json:"readiness" gorm:"column:readiness"`
	CertificateEligible string     `json:"certificateEligible" gorm:"column:certificate_eligible"`

	// Training, Mentoring, Coaching
	TrainingCompleted  string `json:"trainingCompleted" gorm:"column:training_completed"`
	TrainingPlanned    string `json:"trainingPlanned" gorm:"column:training_planned"`
	MentoringCompleted string `json:"mentoringCompleted" gorm:"column:mentoring_completed"`
	MentoringPlanned   string `json:"mentoringPlanned" gorm:"column:mentoring_planned"`
	CoachingCompleted  string `json:"coachingCompleted" gorm:"column:coaching_completed"`
	CoachingPlanned    string `json:"coachingPlanned" gorm:"column:coaching_planned"`

	// Succession Plan
	DataIncumbent    string `json:"dataIncumbent" gorm:"column:data_incumbent"`
	SuccessionVessel string `json:"successionVessel" gorm:"column:succession_vessel"`
	SuccessionRank   string `json:"successionRank" gorm:"column:succession_rank"`

	// Individual Development Plan
	IDPStart  string `json:"idpStart" gorm:"column:idp_start"`
	IDPMentor string `json:"idpMentor" gorm:"column:idp_mentor"`
	IDPCoach  string `json:"idpCoach" gorm:"column:idp_coach"`

	CreatedAt                  *time.Time    `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt                  *time.Time    `json:"updatedAt" gorm:"column:updated_at"`
	User                       string        `json:"user" gorm:"column:user"`
	ReadinessMonth             *int          `json:"readinessMonth" gorm:"column:readiness_month"`
	EducationFulfillmentMonths *int          `json:"educationFulfillmentMonths" gorm:"column:education_fulfillment_months"`
	TotalReadinessUpdateMonths *int          `json:"totalReadinessUpdateMonths" gorm:"column:total_readiness_update_months"`
	Keterangan                 string        `json:"keterangan" gorm:"column:keterangan"`
	TMNM                       string        `json:"tmnm" gorm:"column:tm_nm"`
	ReportScores               []ReportScore `json:"reportScores,omitempty" gorm:"foreignKey:ReportID"`
	BatchID                    *int          `json:"batchId" gorm:"column:batch_id"`
	Batch                      *Batch        `json:"batch,omitempty" gorm:"foreignKey:BatchID;references:ID"`
}
