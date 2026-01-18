package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}
