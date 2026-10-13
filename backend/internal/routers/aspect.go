package routers

import (
	"backend/internal/controllers"
	"backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func AspectRouter(router *gin.Engine, aspectController *controllers.AspectController, auth gin.HandlerFunc) {
	aspectRoutes := router.Group("/api/aspects").Use(middlewares.RequireAuthForWrites(auth))
	{
		aspectRoutes.POST("", aspectController.Create)
		aspectRoutes.GET("", aspectController.FindAll)
		aspectRoutes.GET("/:id", aspectController.FindByID)
		aspectRoutes.PUT("/:id", aspectController.Update)
		aspectRoutes.DELETE("/:id", aspectController.Delete)
		aspectRoutes.GET("/assessment/:assessmentId", aspectController.FindByAssessmentID)
		aspectRoutes.POST("/:id/assign-question", aspectController.AssignQuestionToAspect)
	}
}
