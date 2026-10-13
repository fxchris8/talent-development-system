package converter

import (
	"backend/internal/models/domain"
	"backend/internal/models/web"
)

func ToReportData(data *domain.Report) web.ReportData {
	return web.ReportData{
		ID:                   data.ID,
		SeamanCode:           data.SeamanCode,
		SeafarerCode:         data.SeafarerCode,
		Nama:                 data.Nama,
		IDP:				  data.IDP,
		Jabatan:              data.Jabatan,
		VesselName:           data.VesselName,
		Certificate:          data.Certificate,
		Age:                  data.Age,
		TanggalLahir:         data.TanggalLahir,
		StartDate:            data.StartDate,

		// Catatan Indisipliner
		WarningLetter:        data.WarningLetter,
		CaseHistory:          data.CaseHistory,
		YearOfCase:           data.YearOfCase,

		// Data History Vessel
		VesselHistory:        data.VesselHistory,

		// Kinerja
		KonditeReview:        data.KonditeReview,
		KpiVessel:            data.KpiVessel,
		PerformanceScore:     data.PerformanceScore,
		ValueAssessment:      data.ValueAssessment,
		AssessmentCenter:     data.AssessmentCenter,
		PotentialScore:       data.PotentialScore,
		HavQuadran:           data.HavQuadran,
		HavMapping:           data.HavMapping,
		CompetencyGapAnalysis:data.CompetencyGapAnalysis,
		TotalGap:             data.TotalGap,
		Strength:             data.Strength,
		TalentClassified:     data.TalentClassified,
		IDPProgram:           data.IDPProgram,
		TrainingStatus:       domain.EffectiveTrainingStatus(data.IDPProgram, data.TrainingStatus),
		ExitedProgram:        data.ExitedProgram,
		HavQuadran2:          data.HavQuadran2,
		TalentClassified2:    data.TalentClassified2,
		Readiness:            data.Readiness,
		TotalReadinessUpdateMonths: 		data.TotalReadinessUpdateMonths,
		CertificateEligible:  data.CertificateEligible,

		// Training, Mentoring, Coaching
		TrainingCompleted:    data.TrainingCompleted,
		TrainingPlanned:      data.TrainingPlanned,
		MentoringCompleted:   data.MentoringCompleted,
		MentoringPlanned:     data.MentoringPlanned,
		CoachingCompleted:    data.CoachingCompleted,
		CoachingPlanned:      data.CoachingPlanned,

		// Succession Plan
		DataIncumbent:        data.DataIncumbent,
		SuccessionVessel:     data.SuccessionVessel,
		SuccessionRank:       data.SuccessionRank,

		// Individual Development Plan
		IDPStart:             data.IDPStart,
		IDPMentor:            data.IDPMentor,
		IDPCoach:             data.IDPCoach,
	}
}

func ToReportDataList(list *[]domain.Report) []web.ReportData {
	res := make([]web.ReportData, len(*list))
	for i, r := range *list {
		res[i] = ToReportData(&r)
	}
	return res
}