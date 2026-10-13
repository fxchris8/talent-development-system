package routers

import (
	"backend/internal/controllers"

	"github.com/gin-gonic/gin"
)

func OptionRouter(router *gin.Engine, optionController *controllers.OptionController, auth gin.HandlerFunc) {
	// The question bank holds the answer key (option scores): admins only, reads included.
	// Quiz pages get their questions, without scores, from /api/quiz and /api/assessments/public.
	optionRoutes := router.Group("/api/options").Use(auth)
	{
		optionRoutes.POST("", optionController.Create)
		optionRoutes.GET("", optionController.FindAll)
		optionRoutes.GET("/:optionId", optionController.FindById)
		optionRoutes.GET("/question/:questionId", optionController.FindByQuestionId)
		optionRoutes.PUT("/:optionId", optionController.Update)
		optionRoutes.DELETE("/:optionId", optionController.Delete)
	}
}
