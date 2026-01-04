package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}
