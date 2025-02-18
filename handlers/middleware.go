package handlers

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !strings.HasPrefix(token, "Bearer ") {
		c.AbortWithStatus(401)
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")
	if token != os.Getenv("SERVICE_TOKEN") {
		c.AbortWithStatus(401)
		return
	}

	c.Next()
}
