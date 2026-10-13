package routers

import (
	"backend/internal/controllers"

	"github.com/gin-gonic/gin"
)

// The admin endpoints can wipe whole tables, so they always require a logged-in user.
func SetupAdminRouter(app *gin.Engine, adminController *controllers.AdminController, auth gin.HandlerFunc) {
	admin := app.Group("admin").Use(auth)
	{
		admin.GET("/statistics", adminController.GetStatistics)
		admin.DELETE("/reports", adminController.DeleteAllReports)
		admin.DELETE("/idp-tracking", adminController.DeleteAllIDPTracking)
		admin.DELETE("/assessment-results", adminController.DeleteAllAssessmentResults)
	}
}
