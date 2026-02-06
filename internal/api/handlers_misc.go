package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
	"go.mau.fi/whatsmeow/types"
)

func authStatusHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		authed := false
		connected := false

		if app.WA() != nil {
			authed = app.WA().IsAuthed()
			connected = app.WA().IsConnected()
		}

		// Check if this is an HTMX request
		if c.GetHeader("HX-Request") == "true" {
			statusClass := "disconnected"
			statusText := "Disconnected"
			if authed {
				statusClass = "connected"
				statusText = "Connected"
			}

			html := fmt.Sprintf(`<div class="status-card">
	<span class="status-indicator %s"></span>
	<span class="status-text">%s</span>
</div>
<script>
	updateUI({authenticated: %v, connected: %v});
</script>`, statusClass, statusText, authed, connected)

			c.Header("Content-Type", "text/html")
			c.String(http.StatusOK, html)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"authenticated": authed,
			"connected":     connected,
		})
	}
}

type syncRequest struct {
	HistoryDays     int  `json:"history_days"`
	DownloadMedia   bool `json:"download_media"`
	RefreshContacts bool `json:"refresh_contacts"`
	RefreshGroups   bool `json:"refresh_groups"`
}

func syncHandler(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req syncRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.HistoryDays = 30
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
		defer cancel()

		if err := a.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if req.HistoryDays <= 0 {
			req.HistoryDays = 30
		}

		result, err := a.Sync(ctx, app.SyncOptions{
			Mode:            app.SyncModeOnce,
			DownloadMedia:   req.DownloadMedia,
			RefreshContacts: req.RefreshContacts,
			RefreshGroups:   req.RefreshGroups,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"synced":   result.MessagesStored,
			"messages": result.MessagesStored,
		})
	}
}

func downloadMediaHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaID := c.Param("id")
		chatJID := c.Query("chat")

		if chatJID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chat query parameter is required"})
			return
		}

		msg, err := app.DB().GetMessage(chatJID, mediaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}

		if msg.MediaType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message has no media"})
			return
		}

		c.JSON(http.StatusNotImplemented, gin.H{"error": "media download not yet implemented in API"})
	}
}

type backfillRequest struct {
	ChatJID string `json:"chat_jid"`
	Count   int    `json:"count"`
	LastID  string `json:"last_id"`
}

func backfillHistoryHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req backfillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		count := req.Count
		if count <= 0 {
			count = 100
		}

		lastMsg, err := app.DB().GetMessage(req.ChatJID, req.LastID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid last message ID"})
			return
		}

		lastKnown := types.MessageInfo{
			ID:        lastMsg.MsgID,
			Timestamp: lastMsg.Timestamp,
		}

		reqID, err := app.WA().RequestHistorySyncOnDemand(ctx, lastKnown, count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"requested":  true,
			"request_id": string(reqID),
			"count":      count,
		})
	}
}
