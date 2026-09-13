package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, shopH *ShopHandler, convH *ConversationHandler) {
	g.GET("/shops", shopH.List)
	g.POST("/shops", shopH.Create)
	g.GET("/shops/:id", shopH.Get)
	g.PATCH("/shops/:id", shopH.Update)
	g.POST("/shops/:id/rotate-bind-code", shopH.RotateBindCode)
	g.POST("/shops/:id/reset-plugin", shopH.ResetPlugin)

	g.GET("/conversations", convH.List)
	g.GET("/conversations/:id/messages", convH.Messages)
}
