package api

import (
	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/gin-gonic/gin"
)

func LogoutUserRoutes(r *gin.Engine) {
	r.POST("/api/logout", auth.LogoutHandler)
}
