package router

import (
	"customerservicecore/admin"
	adminmw "customerservicecore/admin/middleware"
	"customerservicecore/internal/config"
	jwtmgr "customerservicecore/internal/pkg/jwt"
	"customerservicecore/internal/pkg/pluginsecret"
	"customerservicecore/internal/repo"
	"customerservicecore/internal/service"
	"customerservicecore/plugin"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	codec, err := pluginsecret.NewCodec(cfg.Auth.PluginSecretKey)
	if err != nil {
		panic(err)
	}
	repos := repo.New(db)
	shopSvc := service.NewShopService(repos, codec)
	shopH := admin.NewShopHandler(shopSvc)
	convH := admin.NewConversationHandler(shopSvc)
	pluginH := plugin.NewHandler(shopSvc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "customerservicecore"})
	})

	v1 := r.Group("/api/v1")
	adminGroup := v1.Group("/admin")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, shopH, convH)

	pluginGroup := v1.Group("/plugin")
	pluginGroup.POST("/bind", pluginH.Bind)
	authed := pluginGroup.Group("")
	authed.Use(pluginH.AuthRequired())
	authed.POST("/heartbeat", pluginH.Heartbeat)
	authed.POST("/messages", pluginH.Messages)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Plugin-Key,X-Plugin-Secret")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
