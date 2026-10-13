package routers

import (
	"backend/internal/controllers"

	"github.com/gin-gonic/gin"
)

func QuestionRouter(router *gin.Engine, questionController *controllers.QuestionController, auth gin.HandlerFunc) {
	// The question bank holds the answer key (option scores): admins only, reads included.
	// Quiz pages get their questions, without scores, from /api/quiz and /api/assessments/public.
	questionRoutes := router.Group("/api/questions").Use(auth)
	{
		questionRoutes.POST("", questionController.Create)
		questionRoutes.GET("", questionController.FindAll)
		questionRoutes.GET("/:questionId", questionController.FindById)
		questionRoutes.GET("/role/:role", questionController.FindByRole)
		questionRoutes.GET("/assessment/:assessmentId", questionController.FindByAssessmentId)
		questionRoutes.PUT("/:questionId", questionController.Update)
		questionRoutes.PUT("/:questionId/aspect", questionController.UpdateAspect)
		questionRoutes.PUT("/bulk/aspect", questionController.BulkUpdateAspect)
		questionRoutes.DELETE("/:questionId", questionController.Delete)
		questionRoutes.DELETE("/bulk", questionController.BulkDelete)
	}
}
