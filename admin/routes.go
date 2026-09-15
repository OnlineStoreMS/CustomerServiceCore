package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, shopH *ShopHandler, convH *ConversationHandler, autoH *AutoReplyHandler) {
	g.GET("/shops", shopH.List)
	g.POST("/shops", shopH.Create)
	g.GET("/shops/:id", shopH.Get)
	g.PATCH("/shops/:id", shopH.Update)
	g.POST("/shops/:id/rotate-bind-code", shopH.RotateBindCode)
	g.POST("/shops/:id/reset-plugin", shopH.ResetPlugin)

	g.GET("/conversations", convH.List)
	g.GET("/conversations/:id/messages", convH.Messages)
	g.POST("/conversations/:id/reply", convH.Reply)

	g.GET("/auto-reply-rules", autoH.List)
	g.POST("/auto-reply-rules", autoH.Create)
	g.POST("/auto-reply-rules/presets", autoH.Seed)
	g.PATCH("/auto-reply-rules/:id", autoH.Update)
	g.DELETE("/auto-reply-rules/:id", autoH.Delete)
}
