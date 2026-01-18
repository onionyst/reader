package routes

import (
	"github.com/gin-gonic/gin"

	"reader/internal/app/reader/app"
)

type Handler struct {
	app *app.App
}

func SetupRoutes(router *gin.Engine, app *app.App) {
	h := &Handler{
		app: app,
	}

	rv := router.Group("api/greader.php")
	{
		rvAccount := rv.Group("accounts")
		{
			rvAccount.POST("ClientLogin", h.clientLogin)
		}

		rvReader := rv.Group("reader/api/0")
		rvReader.Use(h.checkAuth())
		{
			rvReader.POST("edit-tag", h.editTag)

			rvReader.POST("stream/items/contents", h.listStreamItemContents)
			rvReader.GET("stream/items/ids", h.listStreamItemIds)

			rvReader.GET("subscription/list", h.listSubscription)

			rvReader.GET("token", h.token)

			rvReader.GET("user-info", h.userInfo)
		}
	}
}
