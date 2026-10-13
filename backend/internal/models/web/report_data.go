package web

import "time"

type IDPCountData struct {
	MDP int `json:"mdp"`
	FDP int `json:"fdp"`
	SDP int `json:"sdp"`
	// Total counts every report matching the batch/search filter, including people without a program.
	Total int `json:"total"`
}

// ProgramExit is one row of the "left the program" history: who was promoted out of which
// IDP program, from which rank to which rank, and when.
type ProgramExit struct {
	ReportID        int        `json:"reportId"`
	SeafarerCode    string     `json:"seafarerCode"`
	Nama            string     `json:"nama"`
	ExitedProgram   string     `json:"exitedProgram"`
	FromPosition    string     `json:"fromPosition"`    // rank held while in the program ("" when unknown)
	ToPosition      string     `json:"toPosition"`      // rank they were promoted to
	CurrentPosition string     `json:"currentPosition"` // rank today
	VesselName      string     `json:"vesselName"`
	PromotedAt      *time.Time `json:"promotedAt"` // date of the promotion in CITRIX
	ExitedAt        *time.Time `json:"exitedAt"`   // when the system took them out of the program
	// Reason is why the person is no longer an active participant:
	//   "completed" - finished every training of the program (a graduate)
	//   "promoted"  - promoted before finishing
	Reason      string     `json:"reason"`
	Promoted    bool       `json:"promoted"`    // has (also) been promoted out of the program
	CompletedAt *time.Time `json:"completedAt"` // when the program was finished, if known
}

// ReportIdentity is what the public seafarer lookup returns: enough to confirm the person,
// nothing from the talent profile.
type ReportIdentity struct {
	ID           int    `json:"id"`
	SeamanCode   string `json:"seamanCode"`
	SeafarerCode string `json:"seafarerCode"`
	Nama         string `json:"nama"`
	Jabatan      string `json:"jabatan"`
	VesselName   string `json:"vesselName"`
}

type ReportData struct {
	ID           int    `json:"id" gorm:"column:id;primaryKey"`
	SeamanCode   string `json:"seamanCode" gorm:"column:seaman_code"`
	SeafarerCode string `json:"seafarerCode" gorm:"column:seafarer_code"`
	Nama         string `json:"nama" gorm:"column:nama"`
	IDP          string `json:"idp" gorm:"column:idp"`
	Jabatan      string `json:"jabatan" gorm:"column:jabatan"`
	VesselName   string `json:"vesselName" gorm:"column:vessel_name"`
	Certificate  string `json:"certificate" gorm:"column:certificate"`
	Age          string `json:"age" gorm:"column:age"`
	TanggalLahir string `json:"tanggalLahir" gorm:"column:tanggal_lahir"`
	StartDate    *time.Time `json:"startDate" gorm:"column:start_date"`

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
	ValueAssessmentScore  int    `json:"valueAssessmentScore" gorm:"-"`
	AssessmentCenter      int    `json:"assessmentCenter" gorm:"column:assessment_center"`
	PotentialScore        int    `json:"potentialScore" gorm:"column:potential_score"`
	HavQuadran            int    `json:"havQuadran" gorm:"column:hav_quadran"`
	HavMapping            string `json:"havMapping" gorm:"column:hav_mapping"`
	CompetencyGapAnalysis string `json:"competencyGapAnalysis" gorm:"column:competency_gap_analysis"`
	TotalGap              int    `json:"totalGap" gorm:"column:total_gap"`
	Strength              string `json:"strength" gorm:"column:strength"`
	TalentClassified      string `json:"talentClassified" gorm:"column:talent_classified"`
	IDPProgram            string `json:"idpProgram" gorm:"column:idp_program"`
	TrainingStatus        string `json:"trainingStatus" gorm:"column:training_status"`
	ExitedProgram         string `json:"exitedProgram" gorm:"column:exited_program"`
	HavQuadran2           int    `json:"havQuadran2" gorm:"column:hav_quadran2"`
	TalentClassified2     string `json:"talentClassified2" gorm:"column:talent_classified2"`
	Readiness             string `json:"readiness" gorm:"column:readiness"`
	ReadinessMonth             *int   `json:"readinessMonth" gorm:"column:readiness_month"`
	EducationFulfillmentMonths *int   `json:"educationFulfillmentMonths" gorm:"column:education_fulfillment_months"`
	TotalReadinessUpdateMonths *int   `json:"totalReadinessUpdateMonths" gorm:"column:total_readiness_update_months"`
	CertificateEligible   string `json:"certificateEligible" gorm:"column:certificate_eligible"`

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
}
