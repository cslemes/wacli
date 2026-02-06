package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
)

func listChatsHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.DefaultQuery("query", "")
		limitStr := c.DefaultQuery("limit", "100")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100
		}

		chats, err := app.DB().ListChats(query, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"chats": chats})
	}
}

func getChatHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jid := c.Param("jid")

		chat, err := app.DB().GetChat(jid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
			return
		}

		c.JSON(http.StatusOK, chat)
	}
}
