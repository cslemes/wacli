package api

import (
	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
)

func SetupRoutes(router *gin.Engine, app *app.App, cfg *Config) {
	// Public routes (no auth required)
	router.GET("/health", healthHandler)
	router.StaticFile("/", "./web/index.html")
	router.Static("/static", "./web/static")

	// API v1 group (with authentication)
	v1 := router.Group("/api/v1")
	v1.Use(APIKeyAuth(cfg.APIKeys))
	{
		// Messages
		v1.GET("/messages", listMessagesHandler(app))
		v1.GET("/messages/search", searchMessagesHandler(app))
		v1.GET("/messages/:id", getMessageHandler(app))

		// Send messages
		v1.POST("/send/text", sendTextHandler(app))
		v1.POST("/send/file", sendFileHandler(app))

		// Webhooks
		v1.POST("/webhook/grafana", webhookGrafanaHandler(app, cfg))
		v1.POST("/webhook/generic", webhookGenericHandler(app))

		// Contacts
		v1.GET("/contacts", listContactsHandler(app))
		v1.GET("/contacts/search", searchContactsHandler(app))
		v1.GET("/contacts/:jid", getContactHandler(app))
		v1.POST("/contacts/:jid/alias", setContactAliasHandler(app))
		v1.POST("/contacts/refresh", refreshContactsHandler(app))

		// Chats
		v1.GET("/chats", listChatsHandler(app))
		v1.GET("/chats/:jid", getChatHandler(app))

		// Groups
		v1.GET("/groups", listGroupsHandler(app))
		v1.GET("/groups/:jid", getGroupHandler(app))
		v1.POST("/groups/:jid/participants", updateGroupParticipantsHandler(app))
		v1.POST("/groups/:jid/name", updateGroupNameHandler(app))
		v1.GET("/groups/:jid/invite", getGroupInviteHandler(app))
		v1.POST("/groups/join", joinGroupHandler(app))
		v1.POST("/groups/:jid/leave", leaveGroupHandler(app))

		// Auth & sync
		v1.GET("/auth/status", authStatusHandler(app))
		v1.GET("/auth/qr", getQRCodeHandler(app))
		v1.POST("/auth/pair", pairWithCodeHandler(app))
		v1.GET("/auth/wait", waitForPairingHandler(app))
		v1.POST("/auth/logout", logoutHandler(app))
		v1.POST("/sync", syncHandler(app))

		// Media
		v1.GET("/media/:id", downloadMediaHandler(app))

		// History
		v1.POST("/history/backfill", backfillHistoryHandler(app))
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "wacli-api",
	})
}
