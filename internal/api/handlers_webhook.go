package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/wa"
)

// GrafanaAlert represents the incoming Grafana webhook payload
type GrafanaAlert struct {
	Title       string            `json:"title"`
	State       string            `json:"state"`
	Message     string            `json:"message"`
	RuleURL     string            `json:"ruleUrl"`
	EvalMatches []GrafanaMatch    `json:"evalMatches"`
	Tags        map[string]string `json:"tags"`
	// For newer Grafana versions (v9+)
	Alerts []struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	} `json:"alerts"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
}

type GrafanaMatch struct {
	Value  interface{} `json:"value"`
	Metric string      `json:"metric"`
	Tags   interface{} `json:"tags"`
}

// webhookGrafanaHandler handles Grafana webhook alerts
func webhookGrafanaHandler(app *app.App, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read raw body for debugging
		bodyBytes, _ := c.GetRawData()
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		var alert GrafanaAlert
		if err := c.ShouldBindJSON(&alert); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid Grafana webhook format: " + err.Error(),
				"payload": string(bodyBytes),
			})
			return
		}

		// Get recipient from multiple sources (priority order):
		// 1. Query parameter ?to=
		// 2. HTTP header X-WhatsApp-To
		// 3. Grafana commonAnnotations.whatsapp_to
		// 4. Grafana alert annotations.whatsapp_to
		// 5. Grafana tags.whatsapp_to
		recipient := c.Query("to")
		if recipient == "" {
			recipient = c.GetHeader("X-WhatsApp-To")
		}
		if recipient == "" && alert.CommonAnnotations != nil {
			recipient = alert.CommonAnnotations["whatsapp_to"]
		}
		if recipient == "" && len(alert.Alerts) > 0 && alert.Alerts[0].Annotations != nil {
			recipient = alert.Alerts[0].Annotations["whatsapp_to"]
		}
		if recipient == "" && alert.Tags != nil {
			recipient = alert.Tags["whatsapp_to"]
		}
		if recipient == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "recipient required (use ?to=, X-WhatsApp-To header, or whatsapp_to annotation/tag in Grafana)",
				"payload": string(bodyBytes),
				"help":    "Add ?to=PHONE to URL or whatsapp_to annotation in Grafana alert",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		toJID, err := wa.ParseUserOrJID(recipient)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
			return
		}

		// Format the message
		message := formatGrafanaMessage(alert)

		msgID, err := app.WA().SendText(ctx, toJID, message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sent":  true,
			"to":    toJID.String(),
			"id":    msgID,
			"alert": alert.Title,
		})
	}
}

// formatGrafanaMessage formats a Grafana alert into a WhatsApp message
func formatGrafanaMessage(alert GrafanaAlert) string {
	var sb strings.Builder

	// Status emoji
	emoji := "🔔"
	switch alert.State {
	case "alerting":
		emoji = "🚨"
	case "ok":
		emoji = "✅"
	case "no_data":
		emoji = "⚠️"
	}

	sb.WriteString(fmt.Sprintf("%s *Grafana Alert*\n\n", emoji))
	sb.WriteString(fmt.Sprintf("*%s*\n", alert.Title))
	sb.WriteString(fmt.Sprintf("Status: %s\n\n", strings.ToUpper(alert.State)))

	if alert.Message != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", alert.Message))
	}

	// Add matched metrics if available
	if len(alert.EvalMatches) > 0 {
		sb.WriteString("*Metrics:*\n")
		for _, match := range alert.EvalMatches {
			sb.WriteString(fmt.Sprintf("• %s: %v\n", match.Metric, match.Value))
		}
		sb.WriteString("\n")
	}

	// Add tags if available
	if len(alert.Tags) > 0 {
		sb.WriteString("*Tags:*\n")
		for key, value := range alert.Tags {
			sb.WriteString(fmt.Sprintf("• %s: %s\n", key, value))
		}
		sb.WriteString("\n")
	}

	if alert.RuleURL != "" {
		sb.WriteString(fmt.Sprintf("🔗 %s", alert.RuleURL))
	}

	return sb.String()
}

// GenericWebhookRequest allows flexible webhook integration
type GenericWebhookRequest struct {
	To      string                 `json:"to" form:"to"`
	Message string                 `json:"message" form:"message"`
	Data    map[string]interface{} `json:"data"`
}

// webhookGenericHandler is a flexible webhook handler
func webhookGenericHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GenericWebhookRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Try to get 'to' from query if not in body
		if req.To == "" {
			req.To = c.Query("to")
		}

		if req.To == "" || req.Message == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "'to' and 'message' are required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		toJID, err := wa.ParseUserOrJID(req.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
			return
		}

		msgID, err := app.WA().SendText(ctx, toJID, req.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sent": true,
			"to":   toJID.String(),
			"id":   msgID,
		})
	}
}
