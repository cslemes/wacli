package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/store"
)

func listMessagesHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		chatJID := c.Query("chat")
		limitStr := c.DefaultQuery("limit", "100")
		afterStr := c.Query("after")
		beforeStr := c.Query("before")

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100
		}

		var after, before *time.Time
		if afterStr != "" {
			t, err := time.Parse(time.RFC3339, afterStr)
			if err == nil {
				after = &t
			}
		}
		if beforeStr != "" {
			t, err := time.Parse(time.RFC3339, beforeStr)
			if err == nil {
				before = &t
			}
		}

		msgs, err := app.DB().ListMessages(store.ListMessagesParams{
			ChatJID: chatJID,
			Limit:   limit,
			After:   after,
			Before:  before,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"messages": msgs,
			"fts":      app.DB().HasFTS(),
		})
	}
}

func searchMessagesHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
			return
		}

		chatJID := c.Query("chat")
		limitStr := c.DefaultQuery("limit", "100")

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100
		}

		msgs, err := app.DB().SearchMessages(store.SearchMessagesParams{
			Query:   query,
			ChatJID: chatJID,
			Limit:   limit,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"messages": msgs,
			"query":    query,
		})
	}
}

func getMessageHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		msgID := c.Param("id")
		chatJID := c.Query("chat")

		if chatJID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chat query parameter is required"})
			return
		}

		msg, err := app.DB().GetMessage(chatJID, msgID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}

		c.JSON(http.StatusOK, msg)
	}
}
