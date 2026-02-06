package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/wa"
	"go.mau.fi/whatsmeow/types"
)

func listGroupsHandler(app *app.App) gin.HandlerFunc {
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

		groups, err := app.WA().GetJoinedGroups(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"groups": groups})
	}
}

func getGroupHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jidStr := c.Param("jid")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		jid, err := types.ParseJID(jidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group JID"})
			return
		}

		group, err := app.WA().GetGroupInfo(ctx, jid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, group)
	}
}

type updateParticipantsRequest struct {
	Action       string   `json:"action" binding:"required"`
	Participants []string `json:"participants" binding:"required"`
}

func updateGroupParticipantsHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jidStr := c.Param("jid")
		var req updateParticipantsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

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

		groupJID, err := types.ParseJID(jidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group JID"})
			return
		}

		participants := make([]types.JID, 0, len(req.Participants))
		for _, p := range req.Participants {
			jid, err := wa.ParseUserOrJID(p)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant: " + p})
				return
			}
			participants = append(participants, jid)
		}

		var action wa.GroupParticipantAction
		switch req.Action {
		case "add":
			action = wa.GroupParticipantAdd
		case "remove":
			action = wa.GroupParticipantRemove
		case "promote":
			action = wa.GroupParticipantPromote
		case "demote":
			action = wa.GroupParticipantDemote
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
			return
		}

		results, err := app.WA().UpdateGroupParticipants(ctx, groupJID, participants, action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}

type updateGroupNameRequest struct {
	Name string `json:"name" binding:"required"`
}

func updateGroupNameHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jidStr := c.Param("jid")
		var req updateGroupNameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		groupJID, err := types.ParseJID(jidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group JID"})
			return
		}

		if err := app.WA().SetGroupName(ctx, groupJID, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"updated": true, "name": req.Name})
	}
}

func getGroupInviteHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jidStr := c.Param("jid")
		reset := c.Query("reset") == "true"

		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		groupJID, err := types.ParseJID(jidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group JID"})
			return
		}

		link, err := app.WA().GetGroupInviteLink(ctx, groupJID, reset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"link": link})
	}
}

type joinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

func joinGroupHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req joinGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		jid, err := app.WA().JoinGroupWithLink(ctx, req.InviteCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"joined": true, "jid": jid.String()})
	}
}

func leaveGroupHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		jidStr := c.Param("jid")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		groupJID, err := types.ParseJID(jidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group JID"})
			return
		}

		if err := app.WA().LeaveGroup(ctx, groupJID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"left": true})
	}
}
