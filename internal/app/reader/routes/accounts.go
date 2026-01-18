package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"reader/internal/app/reader/models"
	"reader/internal/pkg/apperror"
)

func (h *Handler) checkAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := h.app.Serv.CheckUserAuth(c.Request.Context(), c.Request.Header.Get("Authorization"))
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func (h *Handler) clientLogin(c *gin.Context) {
	var req struct {
		Email    string `form:"Email" binding:"required"`
		Password string `form:"Passwd" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		_ = c.Error(err)
		return
	}

	credentials, err := h.app.Serv.ClientLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.String(http.StatusOK, credentials)
}

func (h *Handler) token(c *gin.Context) {
	user, ok := c.Get("user")
	if !ok {
		_ = c.Error(apperror.InternalServerError(fmt.Errorf("missing user in context")))
		return
	}
	c.String(http.StatusOK, h.app.Serv.GenerateToken(user.(*models.User)))
}

func (h *Handler) userInfo(c *gin.Context) {
	userData, ok := c.Get("user")
	if !ok {
		_ = c.Error(apperror.InternalServerError(fmt.Errorf("missing user in context")))
		return
	}

	if output := c.Query("output"); output != "" && output != "json" {
		_ = c.Error(apperror.BadRequest("invalid output format").WithTarget("output"))
		return
	}

	user := userData.(*models.User)
	userID := strconv.FormatInt(user.ID, 10)

	c.JSON(http.StatusOK, gin.H{
		"userId":        userID,
		"userName":      userID,
		"userProfileId": userID,
		"userEmail":     user.Email,
	})
}
