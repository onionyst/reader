package routes

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes adds all routes to router
func SetupRoutes(router *gin.Engine) {
	rv := router.Group("api/greader.php")
	{
		rvAccount := rv.Group("accounts")
		{
			rvAccount.POST("ClientLogin", clientLogin)
		}

		rvReader := rv.Group("reader/api/0")
		rvReader.Use(checkAuth())
		{
			rvReader.POST("edit-tag", editTag)

			rvReader.POST("stream/items/contents", listStreamItemContents)
			rvReader.GET("stream/items/ids", listStreamItemIds)

			rvReader.GET("subscription/list", listSubscription)

			rvReader.GET("token", token)

			rvReader.GET("user-info", userInfo)
		}
	}

	router.GET("ping", ping)
}
