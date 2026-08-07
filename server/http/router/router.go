package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	commonauth "github.com/nusiss-capstone-project/identity-mservice/common/auth"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/config"
	_ "github.com/nusiss-capstone-project/usergroup-mservice/server/docs"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/http/api"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/http/data"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	serviceURIPrefix = "/usergroup-ms/v1"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(log.RecoveryMiddleware())
	r.Use(otelgin.Middleware(data.ServiceName))
	r.Use(log.HTTPResponseIDMiddleware())
	r.Use(corsMiddleware())

	adminAuth := commonauth.RequireRole([]string{
		commonauth.RoleCampaignOps, commonauth.RoleAdmin,
	})

	basicGroup := r.Group(serviceURIPrefix)
	{
		// High-frequency / non-business routes: no HTTP access log.
		basicGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
		basicGroup.GET("/swagger/*any", gs.WrapHandler(
			swaggerFiles.Handler,
			gs.URL("/usergroup-ms/v1/swagger/doc.json"),
		))
	}

	// Business routes: enable request access logging.
	apiGroup := basicGroup.Group("")
	apiGroup.Use(log.HTTPObservabilityMiddleware())
	{
		adminGroup := apiGroup.Group("/admin")
		adminGroup.Use(adminAuth)
		{
			adminGroup.POST("/usergroups", api.CreateUserGroup)
			adminGroup.GET("/usergroups", api.ListUserGroups)
			adminGroup.GET("/usergroups/:user_group_id", api.GetUserGroup)
			adminGroup.PUT("/usergroups/:user_group_id", api.UpdateUserGroup)
			adminGroup.POST("/usergroups/:user_group_id/publish", api.PublishUserGroup)
			adminGroup.POST("/usergroups/:user_group_id/offline", api.OfflineUserGroup)
			adminGroup.GET("/usergroups/:user_group_id/count", api.EstimateUserGroupSize)
		}
	}
	return r
}

func corsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
			commonauth.HeaderInternalUserID, commonauth.HeaderUserRole,
			log.RequestIDHeader, log.TraceIDHeader,
		},
		ExposeHeaders: []string{
			"Content-Length", commonauth.HeaderInternalUserID, commonauth.HeaderUserRole,
			log.RequestIDHeader, log.TraceIDHeader,
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

func allowedOrigins() []string {
	if config.Config == nil || config.Config.SystemConfig == nil {
		return []string{}
	}
	return config.Config.SystemConfig.AllowedOrigins
}
