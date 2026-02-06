package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
)

func listContactsHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "100")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100
		}

		contacts, err := app.DB().SearchContacts("", limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"contacts": contacts})
	}
}

func searchContactsHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
			return
		}

		limitStr := c.DefaultQuery("limit", "50")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 50
		}

		contacts, err := app.DB().SearchContacts(query, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"contacts": contacts, "query": query})
	}
}

func getContactHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jid := c.Param("jid")

		contact, err := app.DB().GetContact(jid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
			return
		}

		c.JSON(http.StatusOK, contact)
	}
}

type setAliasRequest struct {
	Alias string `json:"alias" binding:"required"`
}

func setContactAliasHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jid := c.Param("jid")
		var req setAliasRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := app.DB().SetAlias(jid, req.Alias); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"jid": jid, "alias": req.Alias})
	}
}

func refreshContactsHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		contacts, err := app.WA().GetAllContacts(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		count := 0
		for jid, info := range contacts {
			_ = app.DB().UpsertContact(jid.String(), jid.User, info.PushName, info.FullName, info.FirstName, info.BusinessName)
			count++
		}

		c.JSON(http.StatusOK, gin.H{"refreshed": count})
	}
}
