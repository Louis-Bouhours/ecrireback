package api

import (
	"github.com/Louis-Bouhours/ecrireback/auth"
	"github.com/gin-gonic/gin"
)

func ApiUserRegister(c *gin.Context) { auth.RegisterHandler(c) }
