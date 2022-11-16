package routes

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"reader/internal/app/reader/models"
	"reader/internal/pkg/routes"
	"reader/internal/pkg/utils"
)

// Login client login binding
type Login struct {
	Email    string `form:"Email" binding:"required"`
	Password string `form:"Passwd" binding:"required"`
}

// UserInfo user information
type UserInfo struct {
	ID        string `json:"userId"`
	Name      string `json:"userName"`
	ProfileID string `json:"userProfileId"`
	Email     string `json:"userEmail"`
}

const (
	authPrefix = "GoogleLogin auth="
)

func checkAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: use c.Abort series functions

		auth := c.Request.Header.Get("Authorization")
		if !strings.HasPrefix(auth, authPrefix) {
			c.JSON(routes.InvalidCredentialsError("Authorization header"))
			return
		}

		sID := strings.TrimPrefix(auth, authPrefix)
		s := strings.Split(sID, "/")
		if len(s) != 2 {
			c.JSON(routes.InvalidCredentialsError("Authorization header"))
			return
		}

		user, err := models.GetUser(s[0])
		if err != nil || user == nil {
			c.JSON(routes.InvalidCredentialsError(""))
			return
		}

		salt := os.Getenv("APP_SALT")
		hash := utils.Sha1(fmt.Sprintf("%s%s%s", salt, user.Email, user.Password))
		if s[1] != hash {
			c.JSON(routes.InvalidCredentialsError(""))
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func checkToken(user *models.User, token string) bool {
	salt := os.Getenv("APP_SALT")
	hash := utils.Sha1(fmt.Sprintf("%s%d%s", salt, user.ID, user.Password))
	return token == utils.PadString(hash, "Z", 57, false)
}

func generateToken(user *models.User) string {
	salt := os.Getenv("APP_SALT")
	hash := utils.Sha1(fmt.Sprintf("%s%d%s", salt, user.ID, user.Password))
	return utils.PadString(hash, "Z", 57, false)
}

func clientLogin(c *gin.Context) {
	var login Login
	if err := c.ShouldBind(&login); err != nil {
		c.JSON(routes.InvalidParameterError("Email or Passwd"))
		return
	}

	user, err := models.GetUser(login.Email)
	if err != nil || user == nil {
		c.JSON(routes.InvalidCredentialsError(""))
		return
	}
	if !utils.VerifyPassword(login.Password, user.Password) {
		c.JSON(routes.InvalidCredentialsError(""))
		return
	}

	salt := os.Getenv("APP_SALT")
	hash := utils.Sha1(fmt.Sprintf("%s%s%s", salt, user.Email, user.Password))
	sid := fmt.Sprintf("%s/%s", user.Email, hash)
	credentials := fmt.Sprintf("SID=%s\nLSID=null\nAuth=%s\n", sid, sid)

	c.String(http.StatusOK, credentials)
}

func token(c *gin.Context) {
	userData, ok := c.Get("user")
	if !ok {
		c.JSON(routes.InternalServerError())
		return
	}

	token := generateToken(userData.(*models.User))

	c.String(http.StatusOK, token)
}

func userInfo(c *gin.Context) {
	userData, ok := c.Get("user")
	if !ok {
		c.JSON(routes.InternalServerError())
		return
	}

	user := userData.(*models.User)

	userID := strconv.FormatInt(user.ID, 10)
	info := &UserInfo{
		ID:        userID,
		Name:      userID,
		ProfileID: userID,
		Email:     user.Email,
	}

	output := c.Request.URL.Query().Get("output")
	switch output {
	case "":
		fallthrough
	case "json":
		c.JSON(http.StatusOK, info)
		return
	default:
		c.JSON(routes.InvalidParameterError("output"))
	}
}
