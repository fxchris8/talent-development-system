package main

import (
	"backend/internal/config"
	"backend/internal/models/domain"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const csvDir = "cmd/seed-csv/data"

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	// SEED_ON_START controls what the seeder does (it runs on every container start):
	//   auto   (default) - fill only tables that are still empty. Rows an admin deleted are
	//                      NOT re-created, and existing data is never touched.
	//   always           - old behaviour: re-create every seed row that is missing.
	//   never            - do nothing.
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_ON_START")))
	if mode == "" {
		mode = "auto"
	}
	if mode == "never" {
		log.Info("SEED_ON_START=never, skipping database seeding")
		return
	}

	db := config.NewDatabase(viperConfig, log)

	// Disable foreign key checks to avoid issues with circular dependencies or legacy schema
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	defer db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Remove invalid foreign key constraint causing seeding errors
	// This constraint is likely a remnant of an old schema and conflicts with the current model
	// We ignore errors here in case the constraint doesn't exist
	db.Exec("ALTER TABLE questions DROP FOREIGN KEY fk_question_answers_question")

	log.Info("Starting CSV-based database seeding...")

	// Seed in order of dependencies
	// IMPORTANT: AssessmentTypes must be seeded before Reports
	// because Reports trigger needs assessment_types data to create report_score
	seeders := []struct {
		name     string
		filename string
		fn       func(*gorm.DB, string) error
		model    interface{}
	}{
		{"Users", "users.csv", seedUsersFromCSV, &domain.User{}},
		{"AssessmentTypes", "assessment_types.csv", seedAssessmentTypesFromCSV, &domain.AssessmentType{}},
		{"Assessments", "assessments.csv", seedAssessmentsFromCSV, &domain.Assessment{}},
		{"CompetencyTypes", "competency_types.csv", seedCompetencyTypesFromCSV, &domain.CompetencyType{}},
		{"Trainings", "trainings.csv", seedTrainingsFromCSV, &domain.Training{}},
		{"CompetencyProgramMappings", "competency_program_mappings.csv", seedCompetencyProgramMappingsFromCSV, &domain.CompetencyProgramMapping{}},
		{"TrainingSchedules", "training_schedules.csv", seedTrainingSchedulesFromCSV, &domain.TrainingSchedule{}},
		{"Aspects", "aspects.csv", seedAspectsFromCSV, &domain.Aspect{}},
		{"Questions", "questions.csv", seedQuestionsFromCSV, &domain.Question{}},
		{"Options", "options.csv", seedOptionsFromCSV, &domain.Option{}},
		{"Reports", "reports.csv", seedReportsFromCSV, &domain.Report{}},
		{"GapCompetencies", "gap_competencies.csv", seedGapCompetenciesFromCSV, &domain.GapCompetency{}},
		{"SeafarerAssessments", "seafarer_assessments.csv", seedSeafarerAssessmentsFromCSV, &domain.SeafarerAssessment{}},
	}

	for _, seeder := range seeders {
		log.Infof("Seeding %s from %s...", seeder.name, seeder.filename)
		filePath := filepath.Join(csvDir, seeder.filename)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Warnf("CSV file not found: %s, skipping...", filePath)
			continue
		}

		if mode != "always" {
			var count int64
			if err := db.Model(seeder.model).Count(&count).Error; err == nil && count > 0 {
				log.Infof("%s already has %d row(s), not seeding (SEED_ON_START=%s)", seeder.name, count, mode)
				continue
			}
		}

		if err := seeder.fn(db, filePath); err != nil {
			log.Fatalf("Failed to seed %s: %v", seeder.name, err)
		}
		log.Infof("Successfully seeded %s", seeder.name)
	}

	log.Info("CSV-based database seeding completed successfully")
}

// Helper function to read CSV file
func readCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Allow variable number of fields per record
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Skip header row
	if len(records) > 0 {
		return records[1:], nil
	}
	return records, nil
}

// Helper functions for parsing
func parseStringPtr(s string) *string {
	if s == "" || s == "NULL" {
		return nil
	}
	return &s
}

func parseString(s string) string {
	if s == "" || s == "NULL" {
		return ""
	}
	return s
}

func parseInt(s string) int {
	if s == "" || s == "NULL" {
		return 0
	}
	val, _ := strconv.Atoi(s)
	return val
}

func parseIntPtr(s string) *int {
	if s == "" || s == "NULL" {
		return nil
	}
	val, _ := strconv.Atoi(s)
	return &val
}

func parseInt64Ptr(s string) *int64 {
	if s == "" || s == "NULL" {
		return nil
	}
	val, _ := strconv.ParseInt(s, 10, 64)
	return &val
}

func parseUint64(s string) uint64 {
	if s == "" || s == "NULL" {
		return 0
	}
	val, _ := strconv.ParseUint(s, 10, 64)
	return val
}

func parseUint64Ptr(s string) *uint64 {
	if s == "" || s == "NULL" {
		return nil
	}
	val, _ := strconv.ParseUint(s, 10, 64)
	return &val
}

func parseFloat64(s string) *float64 {
	if s == "" || s == "NULL" {
		return nil
	}
	val, _ := strconv.ParseFloat(s, 64)
	return &val
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes"
}

func parseTime(s string) *time.Time {
	if s == "" || s == "NULL" {
		return nil
	}
	// Try multiple time formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
}

// Seeder functions
func seedUsersFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 3 {
			continue
		}
		user := domain.User{
			ID:       parseInt(record[0]),
			Username: parseString(record[1]),
			Password: parseString(record[2]),
		}
		if err := db.FirstOrCreate(&user, domain.User{Username: user.Username}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedAssessmentTypesFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		assessmentType := domain.AssessmentType{
			ID:                 parseUint64(record[0]),
			AssessmentTypeName: parseString(record[1]),
			StartTime:          parseTime(record[2]),
			EndTime:            parseTime(record[3]),
			MaxAttempts:        parseIntPtr(record[4]),
		}
		if err := db.FirstOrCreate(&assessmentType, domain.AssessmentType{ID: assessmentType.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedAssessmentsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 6 {
			continue
		}
		assessment := domain.Assessment{
			AssessmentID:      parseUint64(record[0]),
			AssessTypeID:      parseUint64Ptr(record[1]),
			Role:              parseString(record[2]),
			AssessmentName:    parseString(record[3]),
			UsingTimer:        parseBool(record[4]),
			TimerLimitMinutes: parseFloat64(record[5]),
		}
		if err := db.FirstOrCreate(&assessment, domain.Assessment{AssessmentID: assessment.AssessmentID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCompetencyTypesFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 6 {
			continue
		}
		competencyType := domain.CompetencyType{
			ID:          parseInt(record[0]),
			Code:        parseString(record[1]),
			Name:        parseString(record[2]),
			Description: parseString(record[3]),
			Category:    parseString(record[4]),
			IsActive:    parseBool(record[5]),
		}

		// Use Upsert logic: Create if not exists, Update if exists
		var existing domain.CompetencyType
		result := db.First(&existing, competencyType.ID)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new record
			if err := db.Create(&competencyType).Error; err != nil {
				return err
			}
		} else {
			// Update existing record
			if err := db.Model(&existing).Updates(map[string]interface{}{
				"code":        competencyType.Code,
				"name":        competencyType.Name,
				"description": competencyType.Description,
				"category":    competencyType.Category,
				"is_active":   competencyType.IsActive,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedTrainingsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 9 {
			continue
		}
		
		competencyTypeID := parseInt(record[1])
		level := parseInt(record[2])
		deskripsiPerilaku := parseString(record[3])
		toolsTraining := parseString(record[4])
		kode := parseString(record[5])
		topikTraining := parseString(record[6])
		
		// Parse referensi (index 7) - generate exact format as specified
		referensi := parseStringPtr(record[7])
		if referensi == nil || *referensi == "" || *referensi == "-" {
			// Fetch competency type for referensi generation
			var competencyType domain.CompetencyType
			if err := db.First(&competencyType, competencyTypeID).Error; err == nil {
				// Generate referensi with exact format specified by user
				defaultReferensi := fmt.Sprintf(
					"buatkan slide training online untuk judul %s untuk mengungkap kompetensi %s yaitu %s terutama untuk level %d yang %s\n\n"+
						"buatkan ppt antara 13-15 slide ini sebagai ppt video training online, dengan peserta trainingnya adalah perwira kapal container (SPIL) dengan susunan slidenya\n\n"+
						"yaitu slide 1: judul, slide 2: objective, slide 3: konsep penjelasan detail topik training, slide 4: challenges/tantangan, slide 5-9: 1 tool yang mengungkap %s (jelaskan steps2nya secara detail untuk perslide), slide 11: studi kasus, slide 12: pemecahan masalah dalam studi kasus menggunakan metode/tools yang dijelaskan, slide 13: penutup/closing.",
					topikTraining,
					competencyType.Name,
					competencyType.Description,
					level,
					deskripsiPerilaku,
					topikTraining,
				)
				referensi = &defaultReferensi
			}
		}
		
		training := domain.Training{
			No:                parseInt(record[0]),
			CompetencyTypeID:  competencyTypeID,
			Level:             level,
			DeskripsiPerilaku: deskripsiPerilaku,
			ToolsTraining:     toolsTraining,
			Kode:              kode,
			TopikTraining:     topikTraining,
			Referensi:         referensi,
			GeneratedFileURL:  parseStringPtr(record[8]),
			GeneratedPdfURL:   parseStringPtr(record[9]),
		}
		if len(record) > 10 {
			training.GeneratedAt = parseTime(record[10])
		}
		if err := db.FirstOrCreate(&training, domain.Training{No: training.No}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCompetencyProgramMappingsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		mapping := domain.CompetencyProgramMapping{
			ID:                  parseInt(record[0]),
			CompetencyTypeID:    parseInt(record[1]),
			Program:             parseString(record[2]),
			TrainingMaterial1ID: parseIntPtr(record[3]),
			TrainingMaterial2ID: parseIntPtr(record[4]),
			Category:            "M", // Default value, will be calculated dynamically by training_plan_service
		}
		if err := db.FirstOrCreate(&mapping, domain.CompetencyProgramMapping{ID: mapping.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedAspectsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		aspect := domain.Aspect{
			ID:           parseInt(record[0]),
			Name:         parseString(record[1]),
			Weight:       parseInt(record[2]),
			AssessmentID: parseInt(record[3]),
			QuestionID:   parseIntPtr(record[4]),
		}
		if err := db.FirstOrCreate(&aspect, domain.Aspect{ID: aspect.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedQuestionsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 8 {
			continue
		}
		question := domain.Question{
			QuestionID:   parseInt(record[0]),
			Role:         parseString(record[1]),
			QuestionText: parseString(record[2]),
			Category:     parseStringPtr(record[3]),
			IsImage:      parseStringPtr(record[4]),
			ImageURL:     parseStringPtr(record[5]),
			AssessmentID: parseUint64Ptr(record[6]),
			AspectID:     parseInt64Ptr(record[7]),
		}
		if err := db.FirstOrCreate(&question, domain.Question{QuestionID: question.QuestionID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedOptionsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 7 {
			continue
		}

		option := domain.Option{
			OptionID:     parseInt(record[0]),
			QuestionID:   parseInt(record[1]),
			OptionLetter: parseString(record[2]),
			OptionText:   parseString(record[3]),
			Score:        parseInt(record[4]),
			IsImage:      parseInt(record[5]),
			ImageUrl:     parseString(record[6]),
		}
		if err := db.FirstOrCreate(&option, domain.Option{OptionID: option.OptionID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedReportsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 51 {
			continue
		}
		report := domain.Report{
			ID:                         parseInt(record[0]),
			SeamanCode:                 parseString(record[1]),
			SeafarerCode:               parseString(record[2]),
			Nama:                       parseString(record[3]),
			IDP:                        parseString(record[4]),
			Jabatan:                    parseString(record[5]),
			VesselName:                 parseString(record[6]),
			Certificate:                parseString(record[7]),
			Age:                        parseString(record[8]),
			TanggalLahir:               parseString(record[9]),
			StartDate:                  parseTime(record[10]),
			WarningLetter:              parseString(record[11]),
			CaseHistory:                parseString(record[12]),
			YearOfCase:                 parseString(record[13]),
			VesselHistory:              parseString(record[14]),
			KonditeReview:              parseInt(record[15]),
			KpiVessel:                  parseInt(record[16]),
			PerformanceScore:           parseInt(record[17]),
			ValueAssessment:            parseInt(record[18]),
			AssessmentCenter:           parseInt(record[19]),
			PotentialScore:             parseInt(record[20]),
			HavQuadran:                 parseInt(record[21]),
			HavMapping:                 parseString(record[22]),
			CompetencyGapAnalysis:      parseString(record[23]),
			TotalGap:                   parseInt(record[24]),
			Strength:                   parseString(record[25]),
			TalentClassified:           parseString(record[26]),
			IDPProgram:                 parseString(record[27]),
			HavQuadran2:                parseInt(record[28]),
			TalentClassified2:          parseString(record[29]),
			Readiness:                  parseString(record[30]),
			CertificateEligible:        parseString(record[31]),
			TrainingCompleted:          parseString(record[32]),
			TrainingPlanned:            parseString(record[33]),
			MentoringCompleted:         parseString(record[34]),
			MentoringPlanned:           parseString(record[35]),
			CoachingCompleted:          parseString(record[36]),
			CoachingPlanned:            parseString(record[37]),
			DataIncumbent:              parseString(record[38]),
			SuccessionVessel:           parseString(record[39]),
			SuccessionRank:             parseString(record[40]),
			IDPStart:                   parseString(record[41]),
			IDPMentor:                  parseString(record[42]),
			IDPCoach:                   parseString(record[43]),
			CreatedAt:                  parseTime(record[44]),
			UpdatedAt:                  parseTime(record[45]),
			User:                       parseString(record[46]),
			ReadinessMonth:             parseIntPtr(record[47]),
			EducationFulfillmentMonths: parseIntPtr(record[48]),
			TotalReadinessUpdateMonths: parseIntPtr(record[49]),
			Keterangan:                 parseString(record[50]),
		}
		if len(record) > 51 {
			report.TMNM = parseString(record[51])
		}
		if err := db.FirstOrCreate(&report, domain.Report{ID: report.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedSeafarerAssessmentsFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		assessment := domain.SeafarerAssessment{
			ID:               parseUint64(record[0]),
			SeafarerCode:     parseString(record[1]),
			AssessmentTypeID: parseUint64(record[2]),
			Status:           parseString(record[3]),
			AttemptsCount:    parseUint64(record[4]),
		}
		if err := db.FirstOrCreate(&assessment, domain.SeafarerAssessment{ID: assessment.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedTrainingSchedulesFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 6 {
			continue
		}
		
		scheduledDate := parseTime(record[5])
		if scheduledDate == nil {
			continue
		}

		var competencyType domain.CompetencyType
		if err := db.Where("code = ?", record[2]).First(&competencyType).Error; err != nil {
			continue
		}

		competencyTypeID := int64(competencyType.ID)
		schedule := domain.TrainingSchedule{
			ID:               parseInt(record[0]),
			Program:          parseString(record[1]),
			CompetencyCode:   parseString(record[2]),
			CompetencyTypeID: &competencyTypeID,
			TrainingTopic:    parseString(record[3]),
			MaterialType:     parseInt(record[4]),
			ScheduledDate:    *scheduledDate,
		}
		
		if err := db.FirstOrCreate(&schedule, domain.TrainingSchedule{ID: schedule.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedGapCompetenciesFromCSV(db *gorm.DB, filePath string) error {
	records, err := readCSV(filePath)
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		gap := domain.GapCompetency{
			ID:               parseInt(record[0]),
			ReportID:         parseInt(record[1]),
			CompetencyTypeID: parseInt(record[2]),
			GapLevel:         parseString(record[3]),
			Priority:         parseInt(record[4]),
		}
		// Use ON DUPLICATE KEY UPDATE to respect unique constraint on (report_id, competency_type_id)
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "report_id"}, {Name: "competency_type_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"gap_level", "priority", "updated_at"}),
		}).Create(&gap).Error; err != nil {
			return err
		}
	}
	return nil
}

