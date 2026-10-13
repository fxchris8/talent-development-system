package config

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"backend/internal/controllers"
	trainingController "backend/internal/controllers/training"
	"backend/internal/middlewares"
	"backend/internal/repositories"
	"backend/internal/routers"
	"backend/internal/services"
	trainingService "backend/internal/services/training"
)

type BootstrapConfig struct {
	DB       *gorm.DB
	App      *gin.Engine
	Log      *logrus.Logger
	Validate *validator.Validate
	Config   *viper.Viper
}

func InitConfig() *viper.Viper {
	v := viper.New()
	v.SetConfigFile(".env")
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		log.Println("Tidak dapat membaca file .env, menggunakan variabel environment.")
	}
	return v
}

func InitDB(v *viper.Viper) *gorm.DB {
	user := v.GetString("DB_USER")
	pass := v.GetString("DB_PASS")
	host := v.GetString("DB_HOST")
	port := v.GetString("DB_PORT")
	name := v.GetString("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Gagal Menghubungkan Database: %v", err)
	}

	log.Println("Koneksi Database Berhasil!")
	return db
}

func Bootstrap(config *BootstrapConfig) {
	// --- Repository
	reportRepository := repositories.NewReportRepository(config.Log)
	userRepository := repositories.NewUserReposiotry(config.Log)
	mentoringReportRepository := repositories.NewMentoringReportRepository(config.Log)
	coachingReportRepository := repositories.NewCoachingReportRepository(config.DB, config.Log)
	questionRepository := repositories.NewQuestionRepository()
	optionRepository := repositories.NewOptionRepository()
	assessmentResultRepository := repositories.NewAssessmentResultRepository()
	masterRepository := repositories.NewMasterRepository(config.Log)
	trainingRepository := repositories.NewTrainingRepository(config.Log)
	gapCompetencyRepository := repositories.NewGapCompetencyRepository(config.DB, config.Log)
	trainingScheduleRepository := repositories.NewTrainingScheduleRepository(config.DB, config.Log)
	competencyProgramMappingRepository := repositories.NewCompetencyProgramMappingRepository(config.DB)
	competencyTypeRepository := repositories.NewCompetencyTypeRepository(config.DB)
	assessmentTypeRepository := repositories.NewAssessmentTypeRepository()
	seafarerAssessmentRepository := repositories.NewSeafarerAssessmentRepository()
	assignmentRepo := repositories.NewAssignmentRepository()
	aspectRepository := repositories.NewAspectRepository(config.Log)
	idpTrackingRepository := repositories.NewIDPTrackingRepository(config.DB, config.Log)
	apolloTrainingCacheRepository := repositories.NewApolloTrainingCacheRepository(config.DB, config.Log)
	seamenCacheRepository := repositories.NewSeamenCacheRepository(config.DB, config.Log)
	mutationCacheRepository := repositories.NewMutationCacheRepository(config.DB, config.Log)
	seamanRepository := repositories.NewSeamanRepository(config.DB)
	assessmentRepository := repositories.NewAssessmentRepository()
	batchRepository := repositories.NewBatchRepository(config.Log)
	batchSnapshotRepository := repositories.NewBatchSnapshotRepository(config.Log)
	newRecruiterRepository := repositories.NewNewRecruiterRepository()

	// --- Services (DB-based)
	adminService := services.NewAdminService(config.DB, config.Log)
	reportService := services.NewReportService(config.DB, config.Log, config.Validate, reportRepository, gapCompetencyRepository, seamenCacheRepository, mutationCacheRepository)
	seamanService := services.NewSeamanService(seamanRepository)
	userService := services.NewUserService(config.DB, config.Log, config.Validate, config.Config, userRepository)
	ssoService := services.NewSSOService(config.DB, config.Config, config.Log, userRepository, userService)
	mentoringReportService := services.NewMentoringReportService(config.DB, config.Log, config.Validate, mentoringReportRepository)
	coachingReportService := services.NewCoachingReportService(coachingReportRepository, config.Log)
	questionService := services.NewQuestionService(questionRepository, config.Validate)
	optionService := services.NewOptionService(optionRepository, config.Validate)
	assessmentResultService := services.NewAssessmentResultService(assessmentResultRepository, questionRepository, optionRepository, reportRepository, config.Log, config.Validate)

	assessmentService := services.NewAssessmentService(assessmentRepository, assessmentTypeRepository, config.Validate)
	assessmentTypeService := services.NewAssessmentTypeService(assessmentTypeRepository, config.Validate)
	seafarerAssessmentService := services.NewSeafarerAssessmentService(seafarerAssessmentRepository, reportRepository, config.Validate)
	quizService := services.NewQuizService(assessmentTypeRepository, assessmentRepository, questionRepository, optionRepository, config.Validate)
	masterService := services.NewMasterService(config.DB, config.Log, config.Validate, masterRepository)
	trainingServiceDB := services.NewTrainingService(config.DB, config.Log, config.Validate, trainingRepository)
	trainingPlanService := services.NewTrainingPlanService(gapCompetencyRepository, trainingScheduleRepository, competencyProgramMappingRepository, competencyTypeRepository, *reportRepository, config.DB, config.Log)
	assignmentService := services.NewAssignmentService(assignmentRepo, config.Validate)
	aspectService := services.NewAspectService(config.DB, config.Log, aspectRepository)
	batchService := services.NewBatchService(config.DB, config.Log, config.Validate, batchRepository)
	batchSnapshotService := services.NewBatchSnapshotService(config.DB, config.Log, batchRepository, batchSnapshotRepository)
	newRecruiterService := services.NewNewRecruiterService(newRecruiterRepository, config.Validate)

	// --- IDP Tracking Services
	apolloAPIBaseURL := config.Config.GetString("APOLLO_API_BASE_URL")
	apolloAPIService := services.NewApolloAPIService(config.DB, config.Log, apolloTrainingCacheRepository, apolloAPIBaseURL)
	idpCalculationService := services.NewIDPCalculationService(
		config.DB,
		config.Log,
		idpTrackingRepository,
		reportRepository,
		trainingScheduleRepository,
		coachingReportRepository,
		mentoringReportRepository,
		apolloAPIService,
	)

	nanikaAPIService := services.NewNanikaAPIService(config.DB, config.Log, seamenCacheRepository, mutationCacheRepository)

	cronService := services.NewCronService(config.Log, idpCalculationService, apolloAPIService, nanikaAPIService, batchSnapshotService, reportService)

	citrixSyncController := controllers.NewCitrixSyncController(cronService, config.Log)
	reportController := controllers.NewReportController(reportService, config.Log, apolloAPIService)
	userController := controllers.NewUserController(userService, ssoService, config.Log)
	mentoringReportController := controllers.NewMentoringReportController(mentoringReportService, config.Log)
	coachingReportController := controllers.NewCoachingReportController(coachingReportService, config.Log)
	questionController := controllers.NewQuestionController(questionService, optionService, config.DB)
	optionController := controllers.NewOptionController(optionService, config.DB)
	assessmentResultController := controllers.NewAssessmentResultController(assessmentResultService, config.Log, config.DB)
	questionOptionController := controllers.NewQuestionOptionController(config.DB, questionService, optionService, config.Log)
	assessmentController := controllers.NewAssessmentController(config.Log, config.DB, assessmentService, questionService, optionService)
	assessmentTypeController := controllers.NewAssessmentTypeController(config.Log, config.DB, assessmentTypeService)
	seafarerAssessmentController := controllers.NewSeafarerAssessmentController(config.Log, config.DB, seafarerAssessmentService)
	masterController := controllers.NewMasterController(masterService, config.Log)
	trainingControllerDB := controllers.NewTrainingController(trainingServiceDB, config.Log)
	trainingPlanController := controllers.NewTrainingPlanController(trainingPlanService, config.Log)
	competencyMappingController := controllers.NewCompetencyMappingController(config.DB, config.Log, competencyProgramMappingRepository, trainingRepository)
	assignmentController := controllers.NewAssignmentController(config.DB, assignmentService)
	competencyTypeController := controllers.NewCompetencyTypeController(competencyTypeRepository, config.Log)
	aspectController := controllers.NewAspectController(config.Log, aspectService)

	idpTrackingController := controllers.NewIDPTrackingController(config.Log, idpCalculationService)
	seamanController := controllers.NewSeamanController(seamanService)
	quizController := controllers.NewQuizController(config.Log, config.DB, quizService)
	adminController := controllers.NewAdminController(adminService, config.Log)
	scoringConfigController := controllers.NewScoringConfigController(config.Log, config.DB, quizService)
	batchController := controllers.NewBatchController(batchService, batchSnapshotService)
	newRecruiterController := controllers.NewNewRecruiterController(config.DB, newRecruiterService)

	// --- Generator Service & Controller (LLM/PDF)
	llmClient := trainingService.NewLLMClient(
		config.Log,
		config.Config.GetString("OPENROUTER_API_KEY"),
		config.Config.GetString("OPENROUTER_MODEL"),
		config.Config.GetString("OPENROUTER_BASE_URL"),
		config.Config.GetString("MODEL_PROVIDER"),
	)
	trainingGenService := trainingService.NewTrainingService(
		config.Log,
		llmClient,
		config.Config.GetString("BACKEND_PUBLIC_URL"),
	)
	quizGenService := trainingService.NewQuizService(
		config.Log,
		llmClient,
		config.Config.GetString("BACKEND_PUBLIC_URL"),
	)
	narrationAudioService := trainingService.NewNarrationAudioService(
		config.Log,
		config.Config.GetString("TTS_URL"),
		config.Config.GetString("BACKEND_PUBLIC_URL"),
	)
	trainingGenController := trainingController.NewTrainingController(trainingServiceDB, trainingGenService, quizGenService, narrationAudioService, config.Log)

	authMiddleware := middlewares.AuthMiddleware(config.Config.GetString("JWT_SECRET_KEY"))

	// --- Router setup
	routerConfig := &routers.RouterConfig{
		App:                          config.App,
		ReportController:             reportController,
		UserController:               userController,
		MentoringReportController:    mentoringReportController,
		CoachingReportController:     coachingReportController,
		TrainingController:           trainingControllerDB,  // DB
		TrainingGenController:        trainingGenController, // LLM Generator
		TrainingPlanController:       trainingPlanController,
		CompetencyMappingController:  competencyMappingController,
		CompetencyTypeController:     competencyTypeController,
		QuestionController:           questionController,
		OptionController:             optionController,
		AssessmentResultController:   assessmentResultController,
		QuestionOptionController:     questionOptionController,
		AssessmentController:         assessmentController,
		AssessmentTypeController:     assessmentTypeController,
		SeafarerAssessmentController: seafarerAssessmentController,
		MasterController:             masterController,
		AspectController:             aspectController,
		AuthMiddleware:               authMiddleware,
		OptionalAuthMiddleware:       middlewares.OptionalAuth(config.Config.GetString("JWT_SECRET_KEY")),
		AssignmentController:         assignmentController,

		IDPTrackingController:   idpTrackingController,
		SeamanController:        seamanController,
		QuizController:          quizController,
		ScoringConfigController: scoringConfigController,
		NewRecruiterController:  newRecruiterController,
		CitrixSyncController:    citrixSyncController,
		BatchController:         batchController,
	}
	routerConfig.Setup()

	routers.SetupAdminRouter(config.App, adminController, authMiddleware)

	// --- Start Cron Service
	if err := cronService.Start(); err != nil {
		config.Log.Fatalf("Failed to start cron service: %v", err)
	}
	config.Log.Info("Application bootstrap completed successfully")
}
