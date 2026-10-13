package routers

import (
	"backend/internal/controllers"
	traininggen "backend/internal/controllers/training"

	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	App                          *gin.Engine
	ReportController             *controllers.ReportController
	UserController               *controllers.UserController
	MentoringReportController    *controllers.MentoringReportController
	CoachingReportController     *controllers.CoachingReportController
	TrainingController           *controllers.TrainingController // DB
	TrainingGenController        *traininggen.TrainingController // LLM Generator
	TrainingPlanController       *controllers.TrainingPlanController
	CompetencyMappingController  *controllers.CompetencyMappingController
	CompetencyTypeController     *controllers.CompetencyTypeController
	QuestionController           *controllers.QuestionController
	OptionController             *controllers.OptionController
	AssessmentResultController   *controllers.AssessmentResultController
	QuestionOptionController     *controllers.QuestionOptionController
	AssessmentController         *controllers.AssessmentController
	AssessmentTypeController     *controllers.AssessmentTypeController
	SeafarerAssessmentController *controllers.SeafarerAssessmentController
	MasterController             *controllers.MasterController
	AssignmentController         *controllers.AssignmentController
	AspectController             *controllers.AspectController
	IDPTrackingController        *controllers.IDPTrackingController
	SeamanController             *controllers.SeamanController
	QuizController               *controllers.QuizController
	ScoringConfigController      *controllers.ScoringConfigController
	NewRecruiterController       *controllers.NewRecruiterController
	CitrixSyncController         *controllers.CitrixSyncController
	AuthMiddleware               gin.HandlerFunc
	OptionalAuthMiddleware       gin.HandlerFunc
	BatchController              *controllers.BatchController
}

func (c *RouterConfig) Setup() {
	// Health check endpoint (no auth required)
	c.App.GET("/health", controllers.HealthCheck)

	c.App.Static("/files", "./public")
	c.App.Static("/storage", "./storage")
	c.SetupGuestRouter()
	c.SetupAuthRouter()
	c.SetupAssignmentRouter()

	c.SetupMasterRouter()
	c.SetupBatchRouter()
	c.SetupQuizRouter()
	c.SetupScoringConfigRouter()
}

func (c *RouterConfig) SetupGuestRouter() {
	// TODO: Setup Login

	auth := c.App.Group("auth")
	{
		auth.POST("/login", c.UserController.Login)
	}

	sso := c.App.Group("api/auth/sso")
	{
		sso.GET("/initiate", c.UserController.InitiateSSO)
		sso.GET("/callback", c.UserController.SSOCallback)
	}

	// Public: the quiz/assessment pages (no login) look a seafarer up by seafarer code and get
	// identity fields only; the full talent profile is returned to logged-in admins.
	c.App.GET("reports/seafarer-code/:seafarerCode", c.OptionalAuthMiddleware, c.ReportController.FindBySeafarerCode)

	// Everything else about reports is for logged-in admins only.
	report := c.App.Group("reports").Use(c.AuthMiddleware)
	{
		report.GET("", c.ReportController.FindAll)
		report.GET("/idp-count", c.ReportController.IDPCount)
		report.GET("/program-exits", c.ReportController.ProgramExits)
		report.GET("/seaman-code/:seamanCode", c.ReportController.FindBySeamanCode)
		report.GET("/seafarer-code/:seafarerCode/training-data", c.ReportController.GetTrainingData)
		report.GET("/seafarer-code/:seafarerCode/training-summary", c.ReportController.GetTrainingSummary)
		report.POST("/refresh-personal-data", c.ReportController.RefreshPersonalData)
		report.POST("/upload", c.ReportController.CreateAll)
	}

	c.App.GET("trainings", c.TrainingGenController.FindAll)

	// Creating, editing and AI generation (LLM cost) require a logged-in user
	trainings := c.App.Group("trainings").Use(c.AuthMiddleware)
	{
		trainings.POST("", c.TrainingController.Create)
		trainings.POST("/generate", c.TrainingGenController.Generate)
		trainings.POST("/generate-quiz", c.TrainingGenController.GenerateQuiz)
		trainings.POST("/narration", c.TrainingGenController.Narration)
		trainings.POST("/narration-audio", c.TrainingGenController.NarrationAudio)
		trainings.PUT("/:no", c.TrainingController.Update)
		trainings.PUT("/:no/referensi", c.TrainingController.UpdateReferensi)
		trainings.DELETE("/:no", c.TrainingController.Delete)
	}

	trainingPlan := c.App.Group("api/training-plan").Use(c.AuthMiddleware)
	{
		trainingPlan.GET("", c.TrainingPlanController.GetTrainingPlan)
		trainingPlan.POST("/generate-schedules", c.TrainingPlanController.GenerateSchedules)
		trainingPlan.PUT("/swap-schedules", c.TrainingPlanController.SwapSchedules)
		trainingPlan.GET("/competency-mapping", c.TrainingPlanController.GetCompetencyMapping)
		trainingPlan.GET("/programs", c.TrainingPlanController.GetAvailablePrograms)
		trainingPlan.GET("/export-excel", c.TrainingPlanController.ExportTrainingPlanExcel)
		trainingPlan.PUT("/toggle-started/:id", c.TrainingPlanController.ToggleTrainingStarted) // Toggle is_started flag
		trainingPlan.GET("/overdue-count", c.TrainingPlanController.GetOverdueCount)            // Get overdue unstarted count
	}

	// IDP Tracking endpoints (for readiness refresh)
	idpTracking := c.App.Group("api/idp-tracking").Use(c.AuthMiddleware)
	{
		idpTracking.POST("/refresh/:reportId", c.IDPTrackingController.RefreshReadiness) // Refresh specific report
		idpTracking.POST("/refresh-all", c.IDPTrackingController.RefreshAllReadiness)    // Refresh all reports
	}

	// Competency Mapping CMS endpoints
	competencyMapping := c.App.Group("api/competency-mappings").Use(c.AuthMiddleware)
	{
		competencyMapping.GET("", c.CompetencyMappingController.GetMappingsByProgram)
		competencyMapping.GET("/all", c.CompetencyMappingController.GetAllMappings)
		competencyMapping.POST("", c.CompetencyMappingController.CreateMapping)
		competencyMapping.PUT("/:id", c.CompetencyMappingController.UpdateMapping)
		competencyMapping.DELETE("/:id", c.CompetencyMappingController.DeleteMapping)
	}

	// Competency Types endpoints (Public - read only)
	competencyTypes := c.App.Group("api/competency-types")
	{
		competencyTypes.GET("", c.CompetencyTypeController.GetAll)               // Get all competency types
		competencyTypes.GET("/active", c.CompetencyTypeController.GetActive)     // Get only active
		competencyTypes.GET("/:id", c.CompetencyTypeController.GetByID)          // Get by ID
		competencyTypes.GET("/code/:code", c.CompetencyTypeController.GetByCode) // Get by code
	}

	// All trainings endpoint for dropdowns
	c.App.GET("api/trainings", c.CompetencyMappingController.GetAllTrainings)

	mentoringReports := c.App.Group("mentoring-reports").Use(c.AuthMiddleware)
	{
		mentoringReports.POST("", c.MentoringReportController.Create)
		mentoringReports.GET("", c.MentoringReportController.FindAll)
		mentoringReports.GET("/:id", c.MentoringReportController.FindById)
		mentoringReports.PUT("", c.MentoringReportController.Update)
		mentoringReports.DELETE("/:id", c.MentoringReportController.Delete)
		mentoringReports.GET("/by-mentee", c.MentoringReportController.FindByMenteeName)
		mentoringReports.GET("/reports/:reportId", c.MentoringReportController.FindByReportID)
	}

	coachingReports := c.App.Group("coaching-reports").Use(c.AuthMiddleware)
	{
		coachingReports.POST("", c.CoachingReportController.Create)
		coachingReports.GET("", c.CoachingReportController.GetAll)
		coachingReports.GET("/:id", c.CoachingReportController.GetByID)
		coachingReports.PUT("/:id", c.CoachingReportController.Update)
		coachingReports.DELETE("/:id", c.CoachingReportController.Delete)
		coachingReports.GET("/reports/:reportId", c.CoachingReportController.GetByReportID)
	}

	// Combined question-option routes (Public access - only read operations)
	questionsWithOptions := c.App.Group("api/questions-with-options")
	{
		questionsWithOptions.GET("", c.QuestionOptionController.FindAllQuestionsWithOptions)
	}

	assessment := c.App.Group("api/assessments")
	{
		assessment.GET("/public/:role", c.AssessmentController.FindByRolePublic)
		assessment.GET("", c.AssessmentController.FindAllAssessments)
	}

	// Assessment Types endpoints (Public - read only)
	assessmentTypes := c.App.Group("api/assessment-types")
	{
		assessmentTypes.GET("", c.AssessmentTypeController.FindAll)
		assessmentTypes.GET("/check-status/:id", c.AssessmentTypeController.CheckStatus)
		assessmentTypes.GET("/:id", c.AssessmentTypeController.FindByID)
	}

	// Seafarer Assessments endpoints (Public - read only)
	seafarerAssessments := c.App.Group("api/seafarer-assessments")
	{
		seafarerAssessments.GET("", c.SeafarerAssessmentController.FindAll)
		seafarerAssessments.GET("/check-assignment/:seafarerCode/:assessmentTypeId", c.SeafarerAssessmentController.CheckAssignment)
		seafarerAssessments.GET("/check-assignment/:seafarerCode/:assessmentTypeId/:role", c.SeafarerAssessmentController.CheckAssignmentWithRole)
		seafarerAssessments.POST("/increment-attempts/:seafarerCode/:assessmentTypeId", c.SeafarerAssessmentController.IncrementAttempts)
		seafarerAssessments.GET("/:id", c.SeafarerAssessmentController.FindByID)
		seafarerAssessments.GET("/by-seafarer/:seafarerCode", c.SeafarerAssessmentController.FindBySeafarerCode)
		seafarerAssessments.GET("/by-assessment-type/:assessmentTypeId", c.SeafarerAssessmentController.FindByAssessmentTypeID)
	}

	newRecruitersPublic := c.App.Group("api/new-recruiters")
	{
		newRecruitersPublic.GET("/check-assignment/:token/:assessmentTypeId", c.NewRecruiterController.CheckAssignment)
		newRecruitersPublic.GET("/check-assignment/:token/:assessmentTypeId/:role", c.NewRecruiterController.CheckAssignmentWithRole)
		newRecruitersPublic.POST("/increment-attempts/:token/:assessmentTypeId", c.NewRecruiterController.IncrementAttempts)
		newRecruitersPublic.POST("/assessment-results/submit", c.NewRecruiterController.SubmitAssessment)
		newRecruitersPublic.POST("/quiz/submit", c.NewRecruiterController.SubmitQuiz)
	}

	assessmentResults := c.App.Group("assessment-results")
	{
		assessmentResults.POST("/submit", c.AssessmentResultController.Submit)
	}

	// Seamen endpoints (Public - read only)
	seamen := c.App.Group("api/seamen").Use(c.AuthMiddleware)
	{
		seamen.GET("/available", c.SeamanController.SearchAvailableSeamen)
	}

	// Register Question and Option routes
	QuestionRouter(c.App, c.QuestionController, c.AuthMiddleware)
	OptionRouter(c.App, c.OptionController, c.AuthMiddleware)
	AspectRouter(c.App, c.AspectController, c.AuthMiddleware)
}
func (r *RouterConfig) SetupMasterRouter() {

	group := r.App.Group("/api/master-reports").Use(r.AuthMiddleware)

	{
		group.GET("", r.MasterController.FindAll)
		group.GET("/mentoring-programs", r.MasterController.GetMentoringPrograms)
		group.GET("/:id", r.MasterController.FindById)
		group.POST("", r.MasterController.Create)
		group.PUT("/:id", r.MasterController.Update)
		group.DELETE("/:id", r.MasterController.Delete)
		group.POST("/bulk-assign-batch", r.MasterController.BulkAssignBatch)

	}
}

// --- Router assignment baru
func (r *RouterConfig) SetupAssignmentRouter() {
	group := r.App.Group("/api/assignments").Use(r.AuthMiddleware)
	{
		group.GET("", r.AssignmentController.ListPaged)
		group.POST("", r.AssignmentController.Create)
		group.POST("/bulk", r.AssignmentController.BulkCreate)
		group.PUT("/:id", r.AssignmentController.Update)
		group.DELETE("/:id", r.AssignmentController.Delete)
	}
}

func (c *RouterConfig) SetupAuthRouter() {
	// Protected Auth Routes
	auth := c.App.Group("auth").Use(c.AuthMiddleware)
	{
		auth.POST("/logout", c.UserController.Logout)
	}

	// User Management Routes (admin only)
	citrix := c.App.Group("api/citrix").Use(c.AuthMiddleware)
	{
		citrix.POST("/sync", c.CitrixSyncController.Sync)
	}

	userMgmt := c.App.Group("api/users").Use(c.AuthMiddleware)
	{
		userMgmt.POST("", c.UserController.CreateUser)
		userMgmt.GET("", c.UserController.GetAllUsers)
		userMgmt.GET("/:id", c.UserController.GetUserByID)
		userMgmt.PUT("/:id", c.UserController.UpdateUser)
		userMgmt.DELETE("/:id", c.UserController.DeleteUser)
	}

	assessmentAuth := c.App.Group("api/assessments").Use(c.AuthMiddleware)
	{
		assessmentAuth.GET("/unassigned-list", c.AssessmentController.FindUnassigned)
		assessmentAuth.POST("/assign", c.AssessmentController.AssignToType)
		assessmentAuth.GET("/:role", c.AssessmentController.FindByRole)
		assessmentAuth.PUT("/:assessmentId", c.AssessmentController.UpdateAssessment)
		assessmentAuth.PATCH("/:assessmentId/tutorial", c.AssessmentController.UpdateTutorial)
		assessmentAuth.POST("", c.AssessmentController.CreateAssessment)
		assessmentAuth.DELETE("/:assessmentId", c.AssessmentController.DeleteAssessment)
		assessmentAuth.POST("/upload-image", c.AssessmentController.UploadAssessmentImage)
	}

	assessmentResults := c.App.Group("assessment-results").Use(c.AuthMiddleware)
	{
		assessmentResults.GET("/seafarer/:seafarerCode", c.AssessmentResultController.FindBySeafarerCode)
		assessmentResults.GET("/report/:seafarerCode", c.AssessmentResultController.GetValueAssessmentReportBySeafarerCode)
	}

	// Protected Assessment Types endpoints
	assessmentTypesAuth := c.App.Group("api/assessment-types").Use(c.AuthMiddleware)
	{
		assessmentTypesAuth.POST("", c.AssessmentTypeController.Create)
		assessmentTypesAuth.PUT("/:id", c.AssessmentTypeController.Update)
		assessmentTypesAuth.DELETE("/:id", c.AssessmentTypeController.Delete)
	}

	// Protected Seafarer Assessments endpoints
	seafarerAssessmentsAuth := c.App.Group("api/seafarer-assessments").Use(c.AuthMiddleware)
	{
		seafarerAssessmentsAuth.POST("", c.SeafarerAssessmentController.Assign)
		seafarerAssessmentsAuth.PUT("/:id/status", c.SeafarerAssessmentController.UpdateStatus)
		seafarerAssessmentsAuth.DELETE("/:id", c.SeafarerAssessmentController.Delete)
	}

	// Protected Combined question-option routes
	questionsWithOptionsAuth := c.App.Group("api/questions-with-options").Use(c.AuthMiddleware)
	{
		questionsWithOptionsAuth.POST("", c.QuestionOptionController.CreateQuestionWithOptions)
		questionsWithOptionsAuth.PUT("/:questionId", c.QuestionOptionController.UpdateQuestionWithOptions)
		questionsWithOptionsAuth.DELETE("/:questionId", c.QuestionOptionController.DeleteQuestionWithOptions)
		questionsWithOptionsAuth.DELETE("/bulk-delete", c.QuestionOptionController.BulkDelete)
	}

	newRecruitersAuth := c.App.Group("api/new-recruiters").Use(c.AuthMiddleware)
	{
		newRecruitersAuth.GET("", c.NewRecruiterController.FindAll)
		newRecruitersAuth.GET("/search", c.NewRecruiterController.Search)
		newRecruitersAuth.POST("", c.NewRecruiterController.Create)
		newRecruitersAuth.PUT("/:id", c.NewRecruiterController.Update)
		newRecruitersAuth.POST("/bulk-assign-batch", c.NewRecruiterController.BulkAssignBatch)
		newRecruitersAuth.DELETE("/:id", c.NewRecruiterController.Delete)
		newRecruitersAuth.GET("/assignments", c.NewRecruiterController.FindAssignments)
		newRecruitersAuth.POST("/assignments", c.NewRecruiterController.CreateAssignment)
		newRecruitersAuth.DELETE("/assignments/:id", c.NewRecruiterController.DeleteAssignment)
	}
}

func (r *RouterConfig) SetupQuizRouter() {
	group := r.App.Group("/api/quiz")
	{
		group.GET("/history", r.AuthMiddleware, r.QuizController.GetQuizHistory)
		group.GET("/history/:attemptId", r.AuthMiddleware, r.QuizController.GetQuizAttempt)
		group.GET("/:assessmentTypeId", r.QuizController.GetQuizData)
		group.POST("/submit", r.QuizController.SubmitQuiz)
	}
}

func (r *RouterConfig) SetupScoringConfigRouter() {
	group := r.App.Group("/api/scoring-config").Use(r.AuthMiddleware)
	{
		group.GET("/:assessmentTypeId", r.ScoringConfigController.GetScoringConfig)
		group.PUT("/:assessmentTypeId", r.ScoringConfigController.UpdateScoringConfig)
		group.POST("/validate-formula", r.ScoringConfigController.ValidateFormula)
	}
}

func (r *RouterConfig) SetupBatchRouter() {
	group := r.App.Group("/api/batches").Use(r.AuthMiddleware)
	{
		group.GET("", r.BatchController.FindAll)
		group.POST("", r.BatchController.Create)
		group.GET("/:id", r.BatchController.FindByID)
		group.PUT("/:id", r.BatchController.Update)
		group.GET("/:id/snapshots", r.BatchController.GetSnapshots)
	}
}
